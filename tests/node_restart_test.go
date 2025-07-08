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
