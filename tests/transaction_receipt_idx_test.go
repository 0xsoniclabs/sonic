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

	chainId := net.GetChainId()
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)
	sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

	numSimpleTxs := 10
	transactions := prepareTransactions(t, chainId, sender, gasPrice, numSimpleTxs)

	// Send first half of simple transactions
	for i := range numSimpleTxs / 2 {
		err = client.SendTransaction(t.Context(), transactions[i])
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

	// Send second half of simple transactions
	for i := range numSimpleTxs / 2 {
		err = client.SendTransaction(t.Context(), transactions[numSimpleTxs/2+i])
		require.NoError(t, err)
	}

	// Wait for receipt of the internal transaction
	receipt, err := net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.NotNil(t, receipt)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	transactions = block.Transactions()

	// Make sure internal transaction has the correct recipient address
	require.Equal(t, driverauth.ContractAddress, *tx.To(),
		"Transaction recipient address should match the driverauth address")

	// Make sure the block contains simple transactions and the internal one
	require.Greater(t, len(transactions), numSimpleTxs/2,
		"Block should contain some simple transactions and the internal one")

	for i, tx := range transactions {
		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(t, err)

		// Check that the receipt index is equal to the transaction index
		require.Equal(t, uint(i), receipt.TransactionIndex,
			"Receipt index does not match transaction index for tx %d", i)
	}
}

func prepareTransactions(t *testing.T, chainId *big.Int, sender *Account, gasPrice *big.Int, num int) []*types.Transaction {
	transactions := make([]*types.Transaction, num)
	for nonce := range num {
		txData := &types.LegacyTx{
			Nonce:    uint64(nonce),
			Gas:      100000,
			GasPrice: gasPrice,
			To:       &common.Address{0x42},
			Value:    big.NewInt(1),
		}

		tx := signTransaction(t, chainId, txData, sender)
		transactions[nonce] = tx
	}
	return transactions
}
