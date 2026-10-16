// Copyright 2026 Sonic Operations Ltd
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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
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
			require.NoError(err, "failed to endow account")

			block := int(receipt.BlockNumber.Int64())
			receipts[block] = append(receipts[block], receipt)
		}
		require.NoError(net.Restart())
	}

	// Check that access to all blocks is possible.
	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	lastBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)
	require.GreaterOrEqual(lastBlock.NumberU64(), uint64(numBlocks*numRestarts))

	for i := range lastBlock.NumberU64() {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(err)

		for _, receipt := range receipts[int(i)] {
			position := receipt.TransactionIndex
			require.Less(int(position), len(block.Transactions()), "block %d", i)
			got := block.Transactions()[position].Hash()
			require.Equal(got, receipt.TxHash, "block %d, tx %d", i, position)
		}
	}
}

// Q1: a node stops within a bounded time while clients keep requesting, and the
// requests it cannot finish fail instead of hanging.
func TestNodeRestart_StopsUnderRPCLoad(t *testing.T) {
	net := StartIntegrationTestNet(t)

	_, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
	require.NoError(t, err)

	httpClient, err := net.GetClient()
	require.NoError(t, err)
	defer httpClient.Close()
	wsClient, err := net.GetWebSocketClient()
	require.NoError(t, err)
	defer wsClient.Close()

	heads := make(chan *types.Header, 16)
	sub, err := wsClient.SubscribeNewHead(t.Context(), heads)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				// Errors are expected once the node is going down; hanging is not.
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				_, _ = httpClient.BlockByNumber(ctx, nil)
				_, _ = httpClient.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: big.NewInt(0), ToBlock: big.NewInt(1)})
				cancel()
			}
		}()
	}
	time.Sleep(500 * time.Millisecond)

	start := time.Now()
	require.NoError(t, net.Restart())
	require.Less(t, time.Since(start), 2*time.Minute)

	close(stop)
	wg.Wait()

	// The node serves again after the restart.
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	_, err = client.BlockNumber(t.Context())
	require.NoError(t, err)
}
