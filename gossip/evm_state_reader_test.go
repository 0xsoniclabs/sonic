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

package gossip

import (
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestEvmStateReader_Block_IsNotServedBeforeTheLatestBlockIndexCoversIt checks
// that a block written to the store is not served before the latest block index
// is advanced to it. Block processing stores the block, its transactions,
// receipts and logs before it advances that index; serving the block in between
// would let a client see a transaction while queries resolving "latest" still
// answer from the preceding block.
func TestEvmStateReader_Block_IsNotServedBeforeTheLatestBlockIndexCoversIt(t *testing.T) {
	const blockNumber = idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(t, err)

	block := inter.NewBlockBuilder().WithNumber(uint64(blockNumber)).Build()
	store.SetBlock(blockNumber, block)
	store.SetBlockIndex(block.Hash(), blockNumber)

	reader := &EvmStateReader{store: store}

	// The block is in the store, but the head still points at its predecessor.
	setLatestBlockIndex(store, blockNumber-1)
	require.Nil(t, reader.Block(common.Hash{}, uint64(blockNumber)),
		"a block beyond the latest block index must not be served")
	require.Nil(t, reader.Header(common.Hash{}, uint64(blockNumber)),
		"the header of a block beyond the latest block index must not be served")

	// Once the index covers the block, it becomes visible.
	setLatestBlockIndex(store, blockNumber)
	require.NotNil(t, reader.Block(common.Hash{}, uint64(blockNumber)))
	require.NotNil(t, reader.Header(common.Hash{}, uint64(blockNumber)))
}
