package tests

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestRejectedTx(t *testing.T) {
	require := require.New(t)

	// start network
	net := StartIntegrationTestNet(t)

	// create a client
	client, err := net.GetClient()
	require.NoError(err, "failed to get client")
	defer client.Close()

	chainId := getChainId(t, client)

	type testCase struct {
		makeTx func(t *testing.T, account *Account, nonce uint64, price int64) *types.Transaction
	}

	makeTestCaseWithAllTypesOfTx := func(value int64) map[string]testCase {
		valueStr := fmt.Sprint(value)
		cases := map[string]testCase{
			"LegacyTxWithValue" + valueStr: {
				makeTx: func(t *testing.T, account *Account, nonce uint64, price int64) *types.Transaction {
					factory := &txFactory{senderKey: account.PrivateKey, chainId: chainId}
					return factory.makeLegacyTransactionWithPrice(t, nonce, price, value)
				},
			},
			"AccessListTxWithValue" + valueStr: {
				makeTx: func(t *testing.T, account *Account, nonce uint64, price int64) *types.Transaction {
					factory := &txFactory{senderKey: account.PrivateKey, chainId: chainId}
					return factory.makeAccessListTransactionWithPrice(t, nonce, price, value)
				},
			}, "DynamicTxWithValue" + valueStr: {
				makeTx: func(t *testing.T, account *Account, nonce uint64, price int64) *types.Transaction {
					factory := &txFactory{senderKey: account.PrivateKey, chainId: chainId}
					return factory.makeDynamicFeeTransactionWithPrice(t, nonce, price, value)
				},
			}, "BlobTxWithValue" + valueStr: {
				makeTx: func(t *testing.T, account *Account, nonce uint64, price int64) *types.Transaction {
					factory := &txFactory{senderKey: account.PrivateKey, chainId: chainId}
					return factory.makeBlobTransactionWithPrice(t, nonce, price, value)
				},
			},
		}
		return cases
	}

	testCases := map[string]testCase{}
	for name, test := range makeTestCaseWithAllTypesOfTx(0) {
		testCases[name] = test
	}
	for name, test := range makeTestCaseWithAllTypesOfTx(42) {
		testCases[name] = test
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			maxFeeCap := getMaxFee(t, client)
			newAccount := NewAccount()
			nonce := getNonce(t, client, newAccount.Address())
			testRejectedTx(t, net, newAccount.Address(), tc.makeTx(t, newAccount, nonce, maxFeeCap))
		})
	}
}

func testRejectedTx(t *testing.T, net *IntegrationTestNet, account common.Address, tx *types.Transaction) {
	require := require.New(t)

	// verify estimated cost
	estimatedCost := tx.Gas()*tx.GasFeeCap().Uint64() + tx.Value().Uint64()
	if tx.Type() == types.BlobTxType {
		estimatedCost += tx.BlobGasFeeCap().Uint64() * tx.BlobGas()
	}
	require.Equal(tx.Cost().Uint64(), estimatedCost, "transaction estimation is not equal to test estimation")

	// provide just enough balance to NOT cover the cost
	_, err := net.EndowAccount(account, int64(estimatedCost-1))
	require.NoError(err, "failed to endow account")

	// run transaction to be rejected
	receipt, err := net.Run(tx)
	require.ErrorContains(err, "insufficient funds")
	require.Nil(receipt)

	// provide enough balance to cover the cost
	_, err = net.EndowAccount(account, int64(1))
	require.NoError(err, "failed to endow account")

	// run transaction to be successful
	receipt, err = net.Run(tx)
	require.NoError(err)

	// verify receipt
	require.Equal(receipt.Status, uint64(1))
	require.Equal(tx.Gas(), receipt.GasUsed)
	require.GreaterOrEqual(tx.GasFeeCap().Uint64(), receipt.EffectiveGasPrice.Uint64())
	require.GreaterOrEqual(tx.Cost().Uint64(), receipt.EffectiveGasPrice.Uint64()*receipt.GasUsed)
}

func getChainId(t *testing.T, client *ethclient.Client) *big.Int {
	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID::")
	return chainId
}

func getMaxFee(t *testing.T, client *ethclient.Client) (maxFeeCap int64) {
	block, err := client.BlockByNumber(context.Background(), nil)
	require.NoError(t, err, "failed to get block by number")
	baseFee := block.BaseFee().Int64()
	maxFeeCap = int64(float64(baseFee) * 1.05)
	return
}

func getNonce(t *testing.T, client *ethclient.Client, account common.Address) (nonce uint64) {
	nonce, err := client.NonceAt(context.Background(), account, nil)
	require.NoError(t, err, "failed to get nonce:")
	return
}
