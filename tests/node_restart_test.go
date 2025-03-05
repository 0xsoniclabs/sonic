package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestNodeRestart_CanRestartAndRestoreItsState(t *testing.T) {
	const numBlocks = 3
	const numRestarts = 2
	require := require.New(t)

	net := StartIntegrationTestNet(t)

	// All transaction hashes indexed by their blocks.
	receipts := map[int]types.Receipts{}

	// Run through multiple restarts.
	for i := 0; i < numRestarts; i++ {
		for range numBlocks {
			receipt, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
			if err != nil {
				t.Fatalf("failed to endow account; %v", err)
			}
			block := int(receipt.BlockNumber.Int64())
			receipts[block] = append(receipts[block], receipt)
		}
		require.NoError(net.Restart())
	}

	// Check that access to all blocks is possible.
	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	lastBlock, err := client.BlockByNumber(context.Background(), nil)
	require.NoError(err)
	require.GreaterOrEqual(lastBlock.NumberU64(), uint64(numBlocks*numRestarts))

	for i := range lastBlock.NumberU64() {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		require.NoError(err)

		for _, receipt := range receipts[int(i)] {
			position := receipt.TransactionIndex
			require.Less(int(position), len(block.Transactions()), "block %d", i)
			got := block.Transactions()[position].Hash()
			require.Equal(got, receipt.TxHash, "block %d, tx %d", i, position)
		}
	}
}
