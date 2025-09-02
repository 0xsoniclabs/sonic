// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// CreateTransaction fills the given tx with acceptable values for the given
// session, signs it with the given account, and returns the signed transaction.
// The values modified if defaults are:
//   - ChainID: It replaces the ChainID of the transaction with the chainID of
//     the given session.
//   - If nonce is zeroed: It configures the nonce of the transaction to be the
//     current nonce of the sender account
//   - If gas price or gas fee cap is zeroed: It configures the gas price of the
//     transaction to be the suggested gas price
//   - If gas is zeroed: It configures the gas of the transaction to be the
//     minimum gas required to execute the transaction
//     Filled gas is a static minimum value, it does not account for the gas
//     costs of the contract opcodes.
func CreateTransaction(t *testing.T, session IntegrationTestNetSession, tx types.TxData, account *Account) *types.Transaction {
	t.Helper()
	signedTx := SignTransaction(
		t,
		session.GetChainId(),
		SetTransactionDefaults(t, session, tx, account),
		account,
	)
	return signedTx
}

// SignTransaction is a testing helper that signs a transaction with the
// key from the provided account
func SignTransaction(
	t *testing.T,
	chainId *big.Int,
	payload types.TxData,
	from *Account,
) *types.Transaction {
	t.Helper()
	res, err := types.SignTx(
		types.NewTx(payload),
		types.NewPragueSigner(chainId),
		from.PrivateKey)
	require.NoError(t, err)
	return res
}

// SetTransactionDefaults defaults the transaction common fields to meaningful values
//
//   - If nonce is zeroed: It configures the nonce of the transaction to be the
//     current nonce of the sender account
//   - If gas price or gas fee cap is zeroed: It configures the gas price of the
//     transaction to be the suggested gas price
//   - If gas is zeroed: It configures the gas of the transaction to be the
//     minimum gas required to execute the transaction
//     Filled gas is a static minimum value, it does not account for the gas
//     costs of the contract opcodes.
//
// Notice that this function is generic, returning the same type as the input, this
// allows further manual configuration of the transaction fields after the defaults are set.
func SetTransactionDefaults[T types.TxData](
	t *testing.T,
	net IntegrationTestNetSession,
	txPayload T,
	sender *Account,
) T {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// use a types.Transaction type to access polymorphic getters
	tmpTx := types.NewTx(txPayload)
	nonce := tmpTx.Nonce()
	if tmpTx.Nonce() == 0 {
		nonce, err = client.PendingNonceAt(t.Context(), sender.Address())
		require.NoError(t, err)
	}

	gasPrice := tmpTx.GasPrice()
	if gasPrice == nil || gasPrice.Sign() == 0 {
		gasPrice, err = client.SuggestGasPrice(t.Context())
		require.NoError(t, err)
	}

	gas := tmpTx.Gas()
	if gas == 0 {
		gas = computeMinimumGas(t, net, txPayload)
	}

	switch tx := types.TxData(txPayload).(type) {
	case *types.LegacyTx:
		tx.Nonce = nonce
		tx.Gas = gas
		tx.GasPrice = gasPrice
	case *types.AccessListTx:
		tx.Nonce = nonce
		tx.Gas = gas
		tx.GasPrice = big.NewInt(500e9)
	case *types.DynamicFeeTx:
		tx.Nonce = nonce
		tx.Gas = gas
		tx.GasFeeCap = gasPrice
	case *types.BlobTx:
		tx.Nonce = nonce
		tx.Gas = gas
		tx.GasFeeCap = uint256.MustFromBig(gasPrice)
	case *types.SetCodeTx:
		tx.Nonce = nonce
		tx.Gas = gas
		tx.GasFeeCap = uint256.MustFromBig(gasPrice)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}

	return txPayload
}

// ComputeMinimumGas computes the minimum gas required to execute a transaction,
// this accounts for all gas costs except for the contract opcodes gas costs.
func computeMinimumGas(t *testing.T, session IntegrationTestNetSession, tx types.TxData) uint64 {

	var data []byte
	var accessList []types.AccessTuple
	var authorizations []types.SetCodeAuthorization
	var isCreate bool
	switch tx := tx.(type) {
	case *types.LegacyTx:
		data = tx.Data
		isCreate = tx.To == nil
	case *types.AccessListTx:
		data = tx.Data
		accessList = tx.AccessList
		isCreate = tx.To == nil
	case *types.DynamicFeeTx:
		data = tx.Data
		accessList = tx.AccessList
		isCreate = tx.To == nil
	case *types.BlobTx:
		data = tx.Data
		accessList = tx.AccessList
		isCreate = false
	case *types.SetCodeTx:
		data = tx.Data
		accessList = tx.AccessList
		authorizations = tx.AuthList
		isCreate = false
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}

	minimumGas, err := core.IntrinsicGas(data, accessList, authorizations, isCreate, true, true, true)
	require.NoError(t, err)

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	var currentRules opera.Rules
	err = client.Client().Call(&currentRules, "eth_getRules", "latest")
	require.NoError(t, err)

	if currentRules.Upgrades.Allegro {
		floorDataGas, err := core.FloorDataGas(data)
		require.NoError(t, err)
		minimumGas = max(minimumGas, floorDataGas)
	}

	return minimumGas
}

// WaitUntilTransactionIsRetiredFromPool waits until the transaction no longer exists in the transaction pool.
// Because the transaction pool eviction is asynchronous, executed transactions may remain in the pool
// for some time after they have been executed.
// function will eventually time out if the transaction is not retired and an error will be returned.
func WaitUntilTransactionIsRetiredFromPool(t *testing.T, client *PooledEhtClient, tx *types.Transaction) error {
	t.Helper()

	txHash := tx.Hash()
	txSender, err := types.Sender(types.NewPragueSigner(tx.ChainId()), tx)
	require.NoError(t, err, "failed to get transaction sender address")

	// txpool_content returns a map containing two maps:
	// - pending: transactions that are pending to be executed
	// - queued: transactions that are queued to be executed
	// each of the internal maps group transactions by sender address
	var content map[string]map[string]map[string]*ethapi.RPCTransaction
	return WaitFor(t.Context(), func(ctx context.Context) (bool, error) {

		err := client.Client().Call(&content, "txpool_content")
		if err != nil {
			return false, err
		}

		found := false
		if txs, isPending := content["pending"][txSender.Hex()]; isPending {
			for _, tx := range txs {
				if tx.Hash == txHash {
					found = true
					break
				}
			}
		}
		if txs, isQueued := content["queued"][txSender.Hex()]; isQueued {
			for _, tx := range txs {
				if tx.Hash == txHash {
					found = true
					break
				}
			}
		}

		return !found, nil
	})

}

// UpdateNetworkRules sends a transaction to update the network rules.
func UpdateNetworkRules(t *testing.T, net *IntegrationTestNet, rulesChange any) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	b, err := json.Marshal(rulesChange)
	require.NoError(err)

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(err)

	receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.UpdateNetworkRules(ops, b)
	})

	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)
}

// GetNetworkRules retrieves the current network rules from the node.
func GetNetworkRules(t *testing.T, net IntegrationTestNetSession) opera.Rules {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	var rules opera.Rules
	err = WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		err = client.Client().Call(&rules, "eth_getRules", "latest")
		if err != nil {
			return false, err
		}
		return len(rules.Name) > 0, nil
	})

	require.NoError(err, "failed to get network rules")
	return rules
}

func GetEpochOfBlock(t *testing.T, client *PooledEhtClient, blockNumber int) int {
	var result struct {
		Epoch hexutil.Uint64
	}
	err := client.Client().Call(
		&result,
		"eth_getBlockByNumber",
		fmt.Sprintf("0x%x", blockNumber),
		false,
	)
	require.NoError(t, err, "failed to get block number", blockNumber)
	return int(result.Epoch)
}

// MakeAccountWithBalance creates a new account and endows it with the given balance.
// Creating the account this way allows to get access to the private key to sign transactions.
func MakeAccountWithBalance(t *testing.T, net IntegrationTestNetSession, balance *big.Int) *Account {
	t.Helper()
	account := NewAccount()
	receipt, err := net.EndowAccount(account.Address(), balance)
	require.NoError(t, err)
	require.Equal(t,
		receipt.Status, types.ReceiptStatusSuccessful,
		"endowing account failed")
	return account
}

// GenerateTestDataBasedOnModificationCombinations generates all possible versions of a
// given type based on the combinations of modifications.
// The iterator works around a function modify(T, []Piece) T, which shall modify
// an newly constructed instance of T with the provided piece-modifiers.
//
// Arguments:
//   - constructor: a function that constructs a new instance of T, for each version
//     to be based on an unmodified instance.
//   - pieces: a list of lists of pieces, where each list of pieces represents a
//     domain of possible modifications.
//   - modify: a function that modifies an instance of T with the provided pieces.
//
// Returns:
// - an iterator that yields all possible versions of T based on the combinations
func GenerateTestDataBasedOnModificationCombinations[T any, Piece any](
	constructor func() T,
	pieces [][]Piece,
	modify func(tx T, modifier []Piece) T,
) iter.Seq[T] {

	return func(yield func(data T) bool) {
		_cartesianProductRecursion(nil, pieces,
			func(pieces []Piece) bool {
				v := constructor()
				v = modify(v, pieces)
				return yield(v)
			})
	}
}

func _cartesianProductRecursion[T any](current []T, elements [][]T, callback func(data []T) bool) bool {
	if len(elements) == 0 {
		return callback(current)
	}

	var next [][]T
	if len(elements) > 1 {
		next = elements[1:]
	}

	for _, element := range elements[0] {
		if !_cartesianProductRecursion(append(current, element), next, callback) {
			return false
		}
	}
	return true
}

// WaitFor repeatedly calls the predicate function until it returns true, it errors
// or the timeout is reached.
//
// The predicate function receives a context (to forward expiration into internal
// calls) and returns a found boolean and an error (if any).
// - return (false, nil) when the stopping condition is not satisfied
// - return (false, err) when the predicate function encountered an error
// - return (true, nil) when the stopping condition is satisfied
//
// Total wait time is hard-coded to a very generous 100 seconds, this is to allow
// tests with -race not to timeout because their very slow progress. This value is
// arbitrary and was selected by the previous version of this algorithm.
func WaitFor(ctx context.Context, predicate func(context.Context) (bool, error)) error {

	timedContext, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	// implement some backoff strategy: sleeps get longer the longer it
	// takes to receive the event
	backoff := 5 * time.Millisecond

	for {
		ok, err := predicate(timedContext)
		if ok || err != nil {
			return err
		}
		select {
		case <-timedContext.Done():
			return fmt.Errorf("wait timeout")
		case <-time.After(backoff):
			// The predicate was not satisfied, backoff and try again.
			backoff = backoff * 2
		}
	}
}

// AdvanceEpochAndWaitForBlocks sends a transaction to advance to the next epoch.
// It also waits until the new epoch is really reached and the next two blocks are produced.
// It is useful to test a situation when the rule change is applied to the next block after the epoch change.
func AdvanceEpochAndWaitForBlocks(t *testing.T, net *IntegrationTestNet) {
	t.Helper()

	require := require.New(t)

	net.AdvanceEpoch(t, 1)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	currentBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	// wait the next two blocks as some rules (such as min base fee) are applied
	// to the next block after the epoch change becomes effective
	err = WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		newBlock, err := client.BlockByNumber(t.Context(), nil)
		if err != nil {
			return false, err
		}
		return newBlock.Number().Int64() > currentBlock.Number().Int64()+1, nil
	})
	require.NoError(err, "failed to wait for the next two blocks after epoch change")
}
