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

package db

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHealGossipDb_RevertsProcessedBundles(t *testing.T) {
	require := require.New(t)
	dbs, cfg := newDbsWithBundleHistory(t, addCompleteBundleHistory(1))

	// epoch 6 starts at block 1200, which is after the earliest safe block
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(err)
	wantHash, found := store.GetProcessedBundleHistoryHash(1200)
	require.True(found)
	require.NoError(store.Close())

	epoch := idx.Epoch(6)
	_, lastBlock, err := healGossipDb(dbs, cfg, &epoch, 10_000)
	require.NoError(err)
	require.Equal(idx.Block(1200), lastBlock)

	store, err = gossip.NewStore(dbs, cfg)
	require.NoError(err)
	gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(uint64(1200), gotBlock)
	require.Equal(wantHash, gotHash)
}

func TestHealGossipDb_ReportsError_WhenBundleHistoryIsIncomplete(t *testing.T) {
	dbs, cfg := newDbsWithBundleHistory(t, addIncompleteBundleHistory)

	// epoch 3 starts at block 600, for which the bundles processed in the
	// preceding replay protection window are no longer retained
	epoch := idx.Epoch(3)
	_, _, err := healGossipDb(dbs, cfg, &epoch, 10_000)
	require.ErrorContains(t, err, "cannot revert processed bundles history to epoch 3")
	require.ErrorContains(t, err, "earliest safe block is 1025")
}

func TestHealGossipDb_RevertsProcessedBundles_WhenTargetIsWithinFirstWindowAfterFirstBundle(t *testing.T) {
	require := require.New(t)
	dbs, cfg := newDbsWithBundleHistory(t, addCompleteBundleHistory(1))

	// epoch 3 starts at block 600, which is within the first window after the
	// first bundle of the chain, for which the complete history is retained
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(err)
	wantHash, found := store.GetProcessedBundleHistoryHash(600)
	require.True(found)
	require.NoError(store.Close())

	epoch := idx.Epoch(3)
	_, lastBlock, err := healGossipDb(dbs, cfg, &epoch, 10_000)
	require.NoError(err)
	require.Equal(idx.Block(600), lastBlock)

	store, err = gossip.NewStore(dbs, cfg)
	require.NoError(err)
	gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(uint64(600), gotBlock)
	require.Equal(wantHash, gotHash)
}

func TestHealGossipDb_RevertsProcessedBundles_WhenTargetPrecedesFirstBundle(t *testing.T) {
	require := require.New(t)
	dbs, cfg := newDbsWithBundleHistory(t, addCompleteBundleHistory(700))

	// epoch 3 starts at block 600, before the first bundle of the chain
	epoch := idx.Epoch(3)
	_, lastBlock, err := healGossipDb(dbs, cfg, &epoch, 10_000)
	require.NoError(err)
	require.Equal(idx.Block(600), lastBlock)

	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(err)
	gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
	require.Zero(gotBlock)
	require.Equal(common.Hash{}, gotHash)
	require.False(store.HasBundleRecentlyBeenProcessed(common.Hash{0x01}))
}

// addCompleteBundleHistory returns a function adding the history of a chain
// which processed its first bundle at the given block and empty blocks up to
// block 2048.
func addCompleteBundleHistory(firstBundleBlock uint64) func(*gossip.Store) {
	return func(store *gossip.Store) {
		store.AddProcessedBundles(firstBundleBlock, map[common.Hash]bundle.PositionInBlock{{0x01}: {}})
		for block := firstBundleBlock + 1; block <= 2*bundle.MaxBlockRangeLength; block++ {
			store.AddProcessedBundles(block, nil)
		}
	}
}

// addIncompleteBundleHistory adds the history of blocks 1 to 2048 continuing
// a history hash of earlier blocks, whose processed bundles are not retained.
func addIncompleteBundleHistory(store *gossip.Store) {
	store.SetProcessedBundlesHistoryHash(0, common.Hash{0x01})
	for block := uint64(1); block <= 2*bundle.MaxBlockRangeLength; block++ {
		store.AddProcessedBundles(block, nil)
	}
}

// newDbsWithBundleHistory creates databases of a node with the bundle history
// added by the given function, with epoch e starting at block 200*e for
// 10 epochs.
func newDbsWithBundleHistory(
	t *testing.T,
	addBundleHistory func(*gossip.Store),
) (kvdb.FlushableDBProducer, gossip.StoreConfig) {
	t.Helper()
	dbs := flushable.NewSyncedPool(memorydb.NewProducer(""), []byte{0})
	cfg := gossip.MemTestStoreConfig(t.TempDir())
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(t, err)
	addBundleHistory(store)
	for epoch := idx.Epoch(1); epoch <= 10; epoch++ {
		blockState := iblockproc.BlockState{
			LastBlock: iblockproc.BlockCtx{Idx: idx.Block(200 * uint64(epoch))},
		}
		epochState := iblockproc.EpochState{Epoch: epoch}
		store.SetHistoryBlockEpochState(epoch, blockState, epochState)
		store.SetBlockEpochState(blockState, epochState)
	}
	store.FlushBlockEpochState()
	require.NoError(t, store.Close())
	return dbs, cfg
}
