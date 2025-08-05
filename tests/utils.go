package tests

import (
	"encoding/json"
	"fmt"
	"iter"
	"math/big"
	"testing"

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
	var authList []types.AccessTuple
	var authorizations []types.SetCodeAuthorization
	var isCreate bool
	switch tx := tx.(type) {
	case *types.LegacyTx:
		data = tx.Data
		isCreate = tx.To == nil
	case *types.AccessListTx:
		data = tx.Data
		authList = tx.AccessList
		isCreate = tx.To == nil
	case *types.DynamicFeeTx:
		data = tx.Data
		authList = tx.AccessList
		isCreate = tx.To == nil
	case *types.BlobTx:
		data = tx.Data
		authList = tx.AccessList
		isCreate = false
	case *types.SetCodeTx:
		data = tx.Data
		authList = tx.AccessList
		authorizations = tx.AuthList
		isCreate = false
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}

	minimumGas, err := core.IntrinsicGas(data, authList, authorizations, isCreate, true, true, true)
	require.NoError(t, err)

	if session.GetUpgrades().Allegro {
		floorDataGas, err := core.FloorDataGas(data)
		require.NoError(t, err)
		minimumGas = max(minimumGas, floorDataGas)
	}

	return minimumGas
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

	for range 10 {
		var rules opera.Rules
		err = client.Client().Call(&rules, "eth_getRules", "latest")
		require.NoError(err)
		if len(rules.Name) > 0 {
			return rules
		}
	}

	t.Fatal("Failed to retrieve network rules after 10 attempts")
	return opera.Rules{}
}

func GetEpochOfBlock(client *PooledEhtClient, blockNumber int) (int, error) {
	var result struct {
		Epoch hexutil.Uint64
	}
	err := client.Client().Call(
		&result,
		"eth_getBlockByNumber",
		fmt.Sprintf("0x%x", blockNumber),
		false,
	)
	if err != nil {
		return 0, err
	}
	return int(result.Epoch), nil
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
