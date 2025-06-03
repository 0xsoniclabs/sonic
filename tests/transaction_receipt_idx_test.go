package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestReceipt_InternalTransactionsDoNotChangeReceiptIndex(t *testing.T) {
	upgrades := opera.GetSonicUpgrades()
	net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)
	sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

	startBlockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	numSimpleTxs := 10

	// Send simple transactions
	for nonce := range numSimpleTxs / 2 {
		txData := &types.LegacyTx{
			Nonce:    uint64(nonce),
			Gas:      100000,
			GasPrice: gasPrice,
			To:       &common.Address{0x42},
			Value:    big.NewInt(1),
		}

		tx := signTransaction(t, chainId, txData, sender)
		err = client.SendTransaction(t.Context(), tx)
		require.NoError(t, err)
	}

	// Send internal transaction (advance epoch)
	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(t, err)
	txOpts, err := net.GetTransactOptions(&net.account)
	require.NoError(t, err)
	tx, err := contract.AdvanceEpochs(txOpts, big.NewInt(int64(1)))
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Send simple transactions
	for nonce := range numSimpleTxs / 2 {
		txData := &types.LegacyTx{
			Nonce:    uint64(numSimpleTxs/2 + nonce),
			Gas:      100000,
			GasPrice: gasPrice,
			To:       &common.Address{0x42},
			Value:    big.NewInt(1),
		}

		tx := signTransaction(t, chainId, txData, sender)
		err = client.SendTransaction(t.Context(), tx)
		require.NoError(t, err)
	}

	// Use blocking call to ensure all transactions have been processed
	receipt, err := net.EndowAccount(common.Address{0x42}, big.NewInt(1))
	require.NoError(t, err)
	require.NotNil(t, receipt)

	endBlockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Sanity checks
	require.Greater(t, endBlockNumber, startBlockNumber,
		"No blocks were created during the test")
	require.Less(t, endBlockNumber-startBlockNumber, uint64(numSimpleTxs),
		"Too many blocks were created during the test")

	for blockNumber := startBlockNumber; blockNumber <= endBlockNumber; blockNumber++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
		require.NoError(t, err)

		for i, tx := range block.Transactions() {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(t, err)

			// Check that the receipt index is equal to the transaction index
			require.Equal(t, uint(i), receipt.TransactionIndex,
				"Receipt index does not match transaction index for tx %d in block %d", i, blockNumber)
		}
	}
}
