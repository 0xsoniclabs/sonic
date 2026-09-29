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
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRollbackProcessedBundles_Succeeds_WithoutBundleHistory(t *testing.T) {
	store, err := gossip.NewMemStore(t)
	require.NoError(t, err)

	require.NoError(t, rollbackProcessedBundles(store, 3, 100))
}

func TestRollbackProcessedBundles_RevertsHistory_WhenTargetIsDeepEnough(t *testing.T) {
	require := require.New(t)
	store := newStoreWithBundleHistory(t, 1, 2*bundle.MaxBlockRangeLength)

	target := 1 + bundle.MaxBlockRangeLength
	wantHash, found := store.GetProcessedBundleHistoryHash(target)
	require.True(found)

	require.NoError(rollbackProcessedBundles(store, 3, target))

	gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(target, gotBlock)
	require.Equal(wantHash, gotHash)
}

func TestRollbackProcessedBundles_RefusesTarget_WhenBundleHistoryIsTooShallow(t *testing.T) {
	require := require.New(t)
	const head = 2 * bundle.MaxBlockRangeLength
	store := newStoreWithBundleHistory(t, 1, head)
	setEpochStarts(t, store, 10, 200) // < epoch e starts at block 200*e

	// the earliest safe block is 1025, so epoch 6 starting at block 1200 is the earliest safe one
	err := rollbackProcessedBundles(store, 3, 600)
	require.ErrorContains(err, "epoch 3 (block 600) is too deep to heal safely")
	require.ErrorContains(err, "earliest safe epoch is 6")

	// the store is left untouched
	block, _ := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(uint64(head), block)
}

func TestRollbackProcessedBundles_RefusesTarget_WhenNoEpochIsSafeYet(t *testing.T) {
	store := newStoreWithBundleHistory(t, 1, 2*bundle.MaxBlockRangeLength)
	setEpochStarts(t, store, 5, 100) // < all epochs start before the earliest safe block 1025

	err := rollbackProcessedBundles(store, 3, 300)
	require.ErrorContains(t, err, "no epoch starts at or after the earliest safe block 1025 yet")
}

func TestRollbackProcessedBundles_ReportsError_WhenRollbackFails(t *testing.T) {
	const head = 2 * bundle.MaxBlockRangeLength
	store := newStoreWithBundleHistory(t, 1, head)

	// the target is deep enough, but no history hash is retained beyond the head
	err := rollbackProcessedBundles(store, 3, head+1)
	require.ErrorContains(t, err, "failed to revert processed bundles history")
	require.ErrorContains(t, err, "no processed bundles history hash retained for block 2049")
}

// newStoreWithBundleHistory creates a store which processed a bundle at the
// first block and empty blocks up to the head block.
func newStoreWithBundleHistory(t *testing.T, first, head uint64) *gossip.Store {
	t.Helper()
	store, err := gossip.NewMemStore(t)
	require.NoError(t, err)
	store.AddProcessedBundles(first, map[common.Hash]bundle.PositionInBlock{{0x01}: {}})
	for block := first + 1; block <= head; block++ {
		store.AddProcessedBundles(block, nil)
	}
	return store
}

// setEpochStarts records the given number of epochs in the store's epoch
// history, with epoch e starting at block e*blocksPerEpoch; the last one is the
// current epoch.
func setEpochStarts(t *testing.T, store *gossip.Store, epochs idx.Epoch, blocksPerEpoch uint64) {
	t.Helper()
	for epoch := idx.Epoch(1); epoch <= epochs; epoch++ {
		blockState := iblockproc.BlockState{
			LastBlock: iblockproc.BlockCtx{Idx: idx.Block(uint64(epoch) * blocksPerEpoch)},
		}
		epochState := iblockproc.EpochState{Epoch: epoch}
		store.SetHistoryBlockEpochState(epoch, blockState, epochState)
		store.SetBlockEpochState(blockState, epochState)
	}
}
