package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

	// Run one transaction to not interfere with still pending delayed genesis.
	receipt, err := net.EndowAccount(common.Address{}, big.NewInt(1e18))
	require.NoError(t, err)
	before := receipt.BlockNumber.Uint64()

	initialEpoch, err := getEpochOfBlock(client, int(before))
	require.NoError(t, err)

	// Send transaction instructing the network to advance one epoch.
	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(t, err)
	txOpts, err := net.GetTransactOptions(&net.account)
	require.NoError(t, err)
	tx, err := contract.AdvanceEpochs(txOpts, big.NewInt(int64(1)))
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Wait for the epoch to progress while sending in new transactions, hoping
	// to get a transaction into the epoch-sealing block.
	for {
		current, err := client.BlockNumber(t.Context())
		require.NoError(t, err)

		currentEpoch, err := getEpochOfBlock(client, int(current))
		require.NoError(t, err)
		if currentEpoch > initialEpoch {
			break
		}

		_, err = net.EndowAccount(common.Address{}, big.NewInt(1e18))
		require.NoError(t, err)
	}

	after, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Search for block containing the internal sealing transactions.
	var sealingBlock *types.Block
	for cur := before; cur <= after; cur++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(cur)))
		require.NoError(t, err)

		if len(block.Transactions()) == 0 {
			continue
		}
		first := block.Transactions()[0]

		sender, err := getSenderOfTransaction(client, first.Hash())
		require.NoError(t, err)
		if sender == (common.Address{}) {
			sealingBlock = block
			break
		}
	}
	require.NotNil(t, sealingBlock, "No block found with internal transactions")

	// There should be at least 2 internal transactions + one extra transaction.
	// Internal transactions are send by the zero address.
	require.Greater(t, len(sealingBlock.Transactions()), 2)

	transactions := sealingBlock.Transactions()
	sender, err := getSenderOfTransaction(client, transactions[0].Hash())
	require.NoError(t, err)
	require.Equal(t, common.Address{}, sender)

	sender, err = getSenderOfTransaction(client, transactions[1].Hash())
	require.NoError(t, err)
	require.Equal(t, common.Address{}, sender)

	sender, err = getSenderOfTransaction(client, transactions[2].Hash())
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, sender)

	// Check that the index numbers of the receipts match the transaction index.
	for i, tx := range transactions {
		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(t, err)
		require.Equal(t, uint(i), receipt.TransactionIndex,
			"Receipt index does not match transaction index for tx %d", i,
		)
	}
}

func getSenderOfTransaction(
	client *ethclient.Client,
	txHash common.Hash,
) (common.Address, error) {
	details := struct {
		From common.Address
	}{}
	err := client.Client().Call(&details, "eth_getTransactionByHash", txHash)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get transaction details: %w", err)
	}
	return details.From, nil
}
