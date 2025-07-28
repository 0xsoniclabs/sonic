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
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// signTransaction is a testing helper that signs a transaction with the
// key from the provided account
func signTransaction(
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

// setTransactionDefaults defaults the transaction common fields to meaningful values
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
func setTransactionDefaults[T types.TxData](
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

// waitUntilTransactionIsRetiredFromPool waits until the transaction no longer exists in the transaction pool.
// Because the transaction pool eviction is asynchronous, executed transactions may remain in the pool
// for some time after they have been executed.
// function will eventually time out if the transaction is not retired and an error will be returned.
func waitUntilTransactionIsRetiredFromPool(t *testing.T, client *PooledEhtClient, tx *types.Transaction) error {
	t.Helper()

	txHash := tx.Hash()
	txSender, err := types.Sender(types.NewPragueSigner(tx.ChainId()), tx)
	require.NoError(t, err, "failed to get transaction sender address")

	startTime := time.Now()
	timeout := 500 * time.Millisecond
	for {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("transaction %s was not retired from the pool in time", txHash.String())
		}

		// txpool_content returns a map containing two maps:
		// - pending: transactions that are pending to be executed
		// - queued: transactions that are queued to be executed
		// each of the internal maps group transactions by sender address
		var content map[string]map[string]map[string]*ethapi.RPCTransaction
		err := client.Client().Call(&content, "txpool_content")
		require.NoError(t, err, "failed to get txpool content")

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
		if !found {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func TestIntegrationTestNetTools(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		})

	t.Run("setTransactionDefaults sets the transaction defaults", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_setTransactionDefaults(t, net)
	})

	t.Run("waitUntilTransactionIsRetiredFromPool waits from completion", func(t *testing.T) {
		t.Parallel()
		test_WaitUntilTransactionIsRetiredFromPool_waitsFromCompletion(t, net)
	})
}

func testIntegrationTestNet_setTransactionDefaults(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	type modificationFunction func(t *testing.T, tx *types.TxData)

	transactionType := func(txType byte) modificationFunction {
		return func(t *testing.T, tx *types.TxData) {
			switch txType {
			case types.LegacyTxType:
				*tx = &types.LegacyTx{}
			case types.AccessListTxType:
				*tx = &types.AccessListTx{}
			case types.DynamicFeeTxType:
				*tx = &types.DynamicFeeTx{}
			case types.BlobTxType:
				*tx = &types.BlobTx{}
			case types.SetCodeTxType:
				*tx = &types.SetCodeTx{}
			default:
				t.Fatalf("unexpected transaction type: %d", txType)
			}
		}
	}

	noData := func() modificationFunction {
		return func(t *testing.T, tx *types.TxData) {}
	}

	withData := func(size, zeroes int) modificationFunction {

		makeData := func(t *testing.T, size, numZeroes int) []byte {
			require.GreaterOrEqual(t, size, zeroes)
			// please add one 0, so that init-code starts with STOP
			require.Greater(t, zeroes, 0)

			data := make([]byte, size)
			for i := numZeroes; i < size; i++ {
				data[i] = 1
			}
			return data
		}

		return func(t *testing.T, tx *types.TxData) {
			switch tx := (*tx).(type) {
			case *types.LegacyTx:
				tx.Data = makeData(t, size, zeroes)
			case *types.AccessListTx:
				tx.Data = makeData(t, size, zeroes)
			case *types.DynamicFeeTx:
				tx.Data = makeData(t, size, zeroes)
			case *types.BlobTx:
				tx.Data = makeData(t, size, zeroes)
			case *types.SetCodeTx:
				tx.Data = makeData(t, size, zeroes)
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
		}
	}

	noAccessList := func() modificationFunction {
		return func(t *testing.T, tx *types.TxData) {}
	}

	withAccessList := func(accounts, keysPerAccount int) modificationFunction {

		makeAccessList := func(t *testing.T, accounts, keysPerAccount int) []types.AccessTuple {
			accessList := make([]types.AccessTuple, accounts)
			for i := range accessList {
				accessList[i] = types.AccessTuple{
					Address:     NewAccount().Address(),
					StorageKeys: make([]common.Hash, keysPerAccount),
				}
				for j := range accessList[i].StorageKeys {
					accessList[i].StorageKeys[j] = common.BigToHash(big.NewInt(int64(j)))
				}
			}
			return accessList
		}
		return func(t *testing.T, tx *types.TxData) {
			switch tx := (*tx).(type) {
			case *types.LegacyTx:
				// ignore
			case *types.AccessListTx:
				tx.AccessList = makeAccessList(t, accounts, keysPerAccount)
			case *types.DynamicFeeTx:
				tx.AccessList = makeAccessList(t, accounts, keysPerAccount)
			case *types.BlobTx:
				tx.AccessList = makeAccessList(t, accounts, keysPerAccount)
			case *types.SetCodeTx:
				tx.AccessList = makeAccessList(t, accounts, keysPerAccount)
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
		}
	}
	withAuthorizations := func(chainId *big.Int, accounts int) modificationFunction {
		makeAuthList := func(t *testing.T, chainId *big.Int, accounts int) []types.SetCodeAuthorization {
			authList := make([]types.SetCodeAuthorization, accounts)
			for i := range authList {
				account := NewAccount()

				auth, err := types.SignSetCode(account.PrivateKey,
					types.SetCodeAuthorization{
						ChainID: *uint256.MustFromBig(chainId),
						Address: common.BigToAddress(big.NewInt(int64(i))),
						Nonce:   0,
					})
				require.NoError(t, err)
				authList[i] = auth
			}
			return authList
		}

		return func(t *testing.T, tx *types.TxData) {
			switch tx := (*tx).(type) {
			case *types.LegacyTx:
				// ignore
			case *types.AccessListTx:
				// ignore
			case *types.DynamicFeeTx:
				// ignore
			case *types.BlobTx:
				// ignore
			case *types.SetCodeTx:
				tx.AuthList = makeAuthList(t, chainId, accounts)
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
		}
	}

	withoutTo := func() modificationFunction {
		return func(t *testing.T, tx *types.TxData) {
			switch tx := (*tx).(type) {
			case *types.LegacyTx:
				tx.To = nil
			case *types.AccessListTx:
				tx.To = nil
			case *types.DynamicFeeTx:
				tx.To = nil
			case *types.BlobTx, *types.SetCodeTx:
				// ignore without to
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
		}
	}

	withTo := func(address common.Address) modificationFunction {
		return func(t *testing.T, tx *types.TxData) {
			switch tx := (*tx).(type) {
			case *types.LegacyTx:
				tx.To = &address
			case *types.AccessListTx:
				tx.To = &address
			case *types.DynamicFeeTx:
				tx.To = &address
			case *types.BlobTx:
				tx.To = address
			case *types.SetCodeTx:
				tx.To = address
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
		}
	}

	t.Run("filled transactions can be executed", func(t *testing.T) {
		t.Parallel()
		tests := generateTestDataBasedOnModificationCombinations(
			func() types.TxData { return nil },
			[][]modificationFunction{
				// Transaction type
				{
					transactionType(types.LegacyTxType),
					transactionType(types.AccessListTxType),
					transactionType(types.DynamicFeeTxType),
					transactionType(types.BlobTxType),
					transactionType(types.SetCodeTxType),
				},
				// Data
				{noData(), withData(100, 1)},
				// To
				{withoutTo(), withTo(NewAccount().Address())},
				// AccessList
				{noAccessList(), withAccessList(1, 1), withAccessList(8, 4)},
				// Authorizations (for transactions that require them, one minimum)
				{withAuthorizations(chainId, 1), withAuthorizations(chainId, 8)},
			},
			func(tc types.TxData, pieces []modificationFunction) types.TxData {
				for _, piece := range pieces {
					piece(t, &tc)
				}
				return tc
			},
		)

		session := net.SpawnSession(t)

		nonce, err := client.NonceAt(t.Context(), session.GetSessionSponsor().Address(), nil)
		require.NoError(t, err)

		pending := []common.Hash{}

		for tx := range tests {
			switch tx := tx.(type) {
			case *types.LegacyTx:
				tx.Nonce = nonce
			case *types.AccessListTx:
				tx.Nonce = nonce
			case *types.DynamicFeeTx:
				tx.Nonce = nonce
			case *types.BlobTx:
				tx.Nonce = nonce
			case *types.SetCodeTx:
				tx.Nonce = nonce
			default:
				t.Fatalf("unexpected transaction type: %T", tx)
			}
			nonce++

			txData := setTransactionDefaults(t, session, tx, session.GetSessionSponsor())
			tx := signTransaction(t, chainId, txData, session.GetSessionSponsor())

			// the filled values suffice to get the transaction accepted and executed
			err := client.SendTransaction(t.Context(), tx)
			require.NoError(t, err)
			pending = append(pending, tx.Hash())
		}

		for _, txHash := range pending {
			receipt, err := session.GetReceipt(txHash)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		}
	})

	t.Run("zero nonce is defaulted", func(t *testing.T) {
		// this generation is tested isolated because the previous test case
		// utilizes manual nonce setting to issue multiple transactions asynchronously
		t.Parallel()
		session := net.SpawnSession(t)
		// account has a non-zero nonce
		receipt, err := session.EndowAccount(common.Address{}, big.NewInt(1))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		txData := setTransactionDefaults(t, session, &types.LegacyTx{}, session.GetSessionSponsor())
		tx := signTransaction(t, chainId, txData, session.GetSessionSponsor())

		nonce, err := client.NonceAt(t.Context(), session.GetSessionSponsor().Address(), nil)
		require.NoError(t, err)

		require.Equal(t, nonce, tx.Nonce())
	})

	t.Run("non-zero nonce is not defaulted", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)
		// endowments modify the account nonce
		receipt, err := session.EndowAccount(common.Address{}, big.NewInt(1))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		receipt, err = session.EndowAccount(common.Address{}, big.NewInt(1))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		txData := setTransactionDefaults(t, session, &types.LegacyTx{
			Nonce: 1,
		}, session.GetSessionSponsor())
		tx := signTransaction(t, chainId, txData, session.GetSessionSponsor())

		// the filled values suffice to get the transaction accepted and executed
		_, err = session.Run(tx)
		require.ErrorContains(t, err, "nonce too low")
	})

	t.Run("non-zero gas is not defaulted ", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)

		txData := setTransactionDefaults(t, session, &types.LegacyTx{
			Gas: 1,
		}, session.GetSessionSponsor())
		tx := signTransaction(t, chainId, txData, session.GetSessionSponsor())

		// the filled values suffice to get the transaction accepted and executed
		_, err := session.Run(tx)
		require.ErrorContains(t, err, " intrinsic gas too low")
	})

	t.Run("non-zero gas-price is not defaulted ", func(t *testing.T) {
		t.Parallel()

		session := net.SpawnSession(t)
		txData := setTransactionDefaults(t, session, &types.LegacyTx{
			GasPrice: big.NewInt(1),
		}, session.GetSessionSponsor())
		tx := signTransaction(t, chainId, txData, session.GetSessionSponsor())

		// the filled values suffice to get the transaction accepted and executed
		_, err := session.Run(tx)
		require.ErrorContains(t, err, "underpriced")
	})
}

func test_WaitUntilTransactionIsRetiredFromPool_waitsFromCompletion(
	t *testing.T, net *IntegrationTestNet) {

	session := net.SpawnSession(t)

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	account := makeAccountWithBalance(t, session, big.NewInt(1e18))

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	txData := setTransactionDefaults(t, session, &types.LegacyTx{}, account)
	txData.Nonce = 1
	txInvalidNonce := signTransaction(t, chainId, txData, account)

	err = client.SendTransaction(t.Context(), txInvalidNonce)
	require.NoError(t, err)

	// Because nonce is set to current nonce + 1, the transaction will not be executed
	// waiting must time out
	err = waitUntilTransactionIsRetiredFromPool(t, client, txInvalidNonce)
	require.ErrorContains(t, err, fmt.Sprintf("transaction %s was not retired from the pool in time", txInvalidNonce.Hash().String()))

	txData.Nonce = 0
	txCorrectNonce := signTransaction(t, chainId, txData, account)

	err = client.SendTransaction(t.Context(), txCorrectNonce)
	require.NoError(t, err)

	// Once the valid nonce transaction is sent, both transactions will be executed
	// and retired from the pool
	err = waitUntilTransactionIsRetiredFromPool(t, client, txInvalidNonce)
	require.NoError(t, err)
	err = waitUntilTransactionIsRetiredFromPool(t, client, txCorrectNonce)
	require.NoError(t, err)
}
