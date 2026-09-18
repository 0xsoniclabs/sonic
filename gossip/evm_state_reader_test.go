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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

// The block being processed is always one ahead of the latest block index:
// block processing advances that index only after the block is written. Every
// reader off the RPC path therefore lands on exactly n == latest, and none of
// them checks the result for nil. Tightening the guard in getBlock to n >=
// latest, or gating it on anything that can trail the latest index such as the
// archive height, stops block production rather than an RPC call.
func TestEvmStateReader_HeadIsServedThroughEveryEntryPointUsedOffTheRpcPath(t *testing.T) {
	const head = idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(t, err)

	block := inter.NewBlockBuilder().WithNumber(uint64(head)).Build()
	store.SetBlock(head, block)
	store.SetBlockIndex(block.Hash(), head)
	setLatestBlockIndex(store, head)

	reader := &EvmStateReader{store: store}

	// Block processing reads the parent header by number, in
	// c_block_callbacks.go and again in evmmodule.Start.
	require.NotNil(t, reader.Header(common.Hash{}, uint64(head)),
		"block processing must be able to read the parent header")

	// The emitter builds its priority context from the head block, and the
	// transaction pool resets to it and reorgs from it by hash.
	require.NotNil(t, reader.CurrentBlock(),
		"the emitter and the transaction pool must be able to read the head")
	require.NotNil(t, reader.Block(block.Hash(), uint64(head)),
		"the transaction pool must be able to read the head by hash")
}

// TestEvmModule_Start_ReadsTheParentHeaderAtTheLatestBlockIndex pins the
// composition of the guard with its most fragile consumer: evmmodule.Start
// dereferences the parent header without a nil check, so a guard that hides
// the block at the latest index panics block processing instead of failing a
// query. The resulting block's parent hash proves the header was read.
func TestEvmModule_Start_ReadsTheParentHeaderAtTheLatestBlockIndex(t *testing.T) {
	const head = idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(t, err)

	parent := inter.NewBlockBuilder().
		WithNumber(uint64(head)).
		WithBaseFee(big.NewInt(1e9)).
		Build()
	store.SetBlock(head, parent)
	store.SetBlockIndex(parent.Hash(), head)
	setLatestBlockIndex(store, head)

	reader := &EvmStateReader{store: store}
	require.NotNil(t, reader.Header(common.Hash{}, uint64(head)),
		"the parent header must be readable; evmmodule.Start would segfault below")

	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().BeginBlock(uint64(head + 1))
	statedb.EXPECT().EndBlock(uint64(head + 1))
	statedb.EXPECT().GetStateHash()

	processor := evmmodule.New().Start(
		head+1,
		parent.Time+1,
		0,
		statedb,
		reader,
		func(*core_types.Log) {},
		opera.FakeNetRules(opera.GetSonicUpgrades()),
		&params.ChainConfig{},
		common.Hash{},
		nil,
	)

	block, _, _ := processor.Finalize()
	require.Equal(t, parent.Hash(), block.ParentHash,
		"block processing must read the parent header at the latest block index")
}
