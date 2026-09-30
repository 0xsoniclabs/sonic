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
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStore_HasBundleRecentlyBeenProcessed_ReturnsTrueIfFound(t *testing.T) {
	require := require.New(t)
	hash := common.Hash{1, 2, 3}
	bytes := []byte{1, 2, 3}

	store, table, _, _, _ := storeTableLogMocks(t)

	table.EXPECT().Get(getEntryKey(hash)).Return(bytes, nil)
	require.True(store.HasBundleRecentlyBeenProcessed(hash))

	table.EXPECT().Get(getEntryKey(hash)).Return(nil, nil)
	require.False(store.HasBundleRecentlyBeenProcessed(hash))
}

func TestStore_HasBundleRecentlyBeenProcessed_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to check processed bundle", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to check processed bundle: %v", []any{"error", injectedErr}),
		func() { store.HasBundleRecentlyBeenProcessed(common.Hash{1, 2, 3}) })
}

func TestStore_GetBundleExecutionInfo_ReturnsInfoForKnownBundles(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	history := map[uint64]map[common.Hash]bundle.PositionInBlock{}

	// construct a history of bundles in boundary block number and boundary positions
	for i, blockNum := range []uint64{0, 1, bundle.MaxBlockRangeLength / 2, bundle.MaxBlockRangeLength - 2, bundle.MaxBlockRangeLength - 1} {
		history[blockNum] = map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(uint64(i * 3)): {
				Offset: 0,
				Count:  1,
			},
			uint64ToHash(uint64(i*3 + 1)): {
				Offset: 2,
				Count:  3,
			},
			uint64ToHash(uint64(i*3 + 2)): {
				Offset: 4,
				Count:  5,
			},
		}
	}

	// initialize storage with the provided history
	table := store.table.ProcessedBundles
	batch := table.NewBatch()
	for blockNum, infos := range history {
		_, err := store.addNewBundles(blockNum, infos, batch)
		require.NoError(err)
	}
	require.NoError(batch.Write())

	for blockNum, infos := range history {
		for hash, expected := range infos {
			// check every element in the history can be retrieved correctly by the store
			t.Run(fmt.Sprintf("BlockNumber=%d/Hash=%x", blockNum, hash), func(t *testing.T) {
				want := bundle.ExecutionInfo{
					ExecutionPlanHash: hash,
					BlockNumber:       blockNum,
					Position:          expected,
				}
				info := store.GetBundleExecutionInfo(hash)
				require.Equal(want, *info)
			})
		}
	}
}

func TestStore_GetBundleExecutionInfo_ReturnsNilForUnknownBundles(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)
	hash := common.Hash{1, 2, 3}
	got := store.GetBundleExecutionInfo(hash)
	require.Nil(got)
}

func TestStore_GetBundleExecutionInfo_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to get execution info for bundle", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to get execution info for bundle: %v", []any{"error", injectedErr}),
		func() { store.GetBundleExecutionInfo(common.Hash{1, 2, 3}) })
}

func TestStore_GetBundleExecutionInfo_LogsOnInvalidDataLength(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	table.EXPECT().Get(gomock.Any()).Return([]byte{1, 2, 3}, nil)

	expectCrit(log, "invalid data length for execution info", "length", 3)

	require.PanicsWithValue(t,
		fmt.Sprintf("invalid data length for execution info: %v", []any{"length", 3}),
		func() { store.GetBundleExecutionInfo(common.Hash{1, 2, 3}) })
}

func TestStore_AddProcessedBundles_AddsNewBundlesToStorage(t *testing.T) {
	// this test sets 4 core expectations:
	// 1. new bundle is added to the storage
	// 2. the index for the block number is updated
	// 3. the history hash is updated
	// 4. when the history is large enough, outdated entries are deleted

	for _, block := range []uint64{
		0, 1,
		DefaultProcessedBundlesRetention - 1,
		DefaultProcessedBundlesRetention,
		DefaultProcessedBundlesRetention + 1,
	} {
		t.Run(fmt.Sprintf("BlockNumber=%d", block), func(t *testing.T) {
			store, table, _, batch, it := storeTableLogMocks(t)

			table.EXPECT().NewBatch().Return(batch)
			table.EXPECT().Get(gomock.Any())

			batch.EXPECT().Put(nil, BlockHashTableValueMatcher{blockNum: block})
			batch.EXPECT().Put(getBundleHistoryHashKey(block), gomock.Any())
			batch.EXPECT().Write().Return(nil)

			info1 := bundle.ExecutionInfo{
				ExecutionPlanHash: uint64ToHash(block),
				BlockNumber:       block,
			}
			batch.EXPECT().Put(
				getEntryKey(info1.ExecutionPlanHash),
				BundleExecutionInfoMatcher{expected: info1},
			)
			batch.EXPECT().Put(
				getIndexKey(block, info1.ExecutionPlanHash),
				[]byte{0},
			)
			// when the history is large enough, the store starts deleting outdated entries.
			if block >= DefaultProcessedBundlesRetention-1 {
				toDelete := block - DefaultProcessedBundlesRetention + 1

				table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it)
				next := it.EXPECT().Next().Return(true)
				it.EXPECT().Next().Return(false).After(next).AnyTimes()
				it.EXPECT().Key().Return(getIndexKey(toDelete, uint64ToHash(toDelete)))
				table.EXPECT().Get(getEntryKey(uint64ToHash(toDelete))).Return(nil, nil).AnyTimes()

				hash := uint64ToHash(toDelete)
				batch.EXPECT().Delete(getEntryKey(hash)).Return(nil)
				batch.EXPECT().Delete(getIndexKey(toDelete, hash)).Return(nil)
				it.EXPECT().Error().Return(nil)
				it.EXPECT().Release()

				// Second pass: 'h' iterator for bundle-less blocks (empty in mock world).
				it2 := NewMockdbIterator(gomock.NewController(t))
				it2.EXPECT().Next().Return(false)
				it2.EXPECT().Error().Return(nil)
				it2.EXPECT().Release()
				table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(0)).Return(it2)
			}

			store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{
				info1.ExecutionPlanHash: {},
			})
		})
	}
}

func TestStore_addProcessedBundles_ReportsError_OnBatchPutError(t *testing.T) {
	store, table, _, batch, _ := storeTableLogMocks(t)

	historyHashErr := errors.New("new entry put error")
	blockHistoryHashErr := errors.New("block history hash put error")
	// Put calls for nil key and block-hash key are issued; both return an error.
	batch.EXPECT().Put(nil, gomock.Any()).Return(historyHashErr)
	batch.EXPECT().Put(getBundleHistoryHashKey(1), gomock.Any()).
		Return(blockHistoryHashErr)

	table.EXPECT().NewBatch().Return(batch)
	oldHistoryEntry := []byte{8 + 32 - 1: 0x42}
	table.EXPECT().Get(gomock.Any()).Return(oldHistoryEntry, nil)

	err := store.addProcessedBundles(1, nil)
	require.ErrorContains(t, err, "failed to update hash of processed bundles")
	require.ErrorIs(t, err, historyHashErr)
	require.ErrorIs(t, err, blockHistoryHashErr)
}

func TestStore_addProcessedBundles_ReportsError_OnBatchWriteError(t *testing.T) {
	store, table, _, batch, _ := storeTableLogMocks(t)

	injectedErr := errors.New("batch write error")
	// Both Put calls (nil key and block-hash key) are issued.
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	batch.EXPECT().Write().Return(injectedErr)

	table.EXPECT().NewBatch().Return(batch)
	oldHistoryEntry := []byte{8 + 32 - 1: 0x42}
	table.EXPECT().Get(gomock.Any()).Return(oldHistoryEntry, nil)

	err := store.addProcessedBundles(1, nil)
	require.ErrorContains(t, err, "failed to write batch for updating processed bundles")
	require.ErrorIs(t, err, injectedErr)
}

func TestStore_addProcessedBundles_ReportsError_WhenAddingNewBundlesFails(t *testing.T) {
	store, table, _, batch, _ := storeTableLogMocks(t)

	injectedErr := errors.New("entry put error")
	table.EXPECT().Get(gomock.Any()).Return([]byte{8 + 32 - 1: 0x42}, nil)
	table.EXPECT().NewBatch().Return(batch)
	batch.EXPECT().Put(getEntryKey(common.Hash{0x01}), gomock.Any()).Return(injectedErr)
	batch.EXPECT().Put(getIndexKey(1, common.Hash{0x01}), gomock.Any()).Return(nil)
	// no batch.Write() is expected

	err := store.addProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{{0x01}: {}})
	require.ErrorIs(t, err, injectedErr)
}

func TestStore_addProcessedBundles_ReportsError_WhenDeletingOutdatedBundlesFails(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	injectedErr := errors.New("injected iteration error")
	table.EXPECT().Get(gomock.Any()).Return([]byte{8 + 32 - 1: 0x42}, nil)
	table.EXPECT().NewBatch().Return(batch)
	table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it)
	it.EXPECT().Next().Return(false)
	it.EXPECT().Error().Return(injectedErr)
	it.EXPECT().Release()
	// no batch.Write() is expected

	err := store.addProcessedBundles(DefaultProcessedBundlesRetention, nil)
	require.ErrorIs(t, err, injectedErr)
}

func TestStore_AddProcessedBundles_LogsCrit_WhenUpdateFails(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)

	injectedErr := errors.New("batch write error")
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	batch.EXPECT().Write().Return(injectedErr)
	table.EXPECT().NewBatch().Return(batch)
	oldHistoryEntry := []byte{8 + 32 - 1: 0x42}
	table.EXPECT().Get(gomock.Any()).Return(oldHistoryEntry, nil)

	// block processing can not recover from a failed update, so the node halts
	log.EXPECT().Crit("failed to update processed bundles", gomock.Any()).
		Do(func(string, ...any) { panic("deliberately stopped by unit test") })
	require.PanicsWithValue(t, "deliberately stopped by unit test",
		func() { store.AddProcessedBundles(1, nil) })
}

func TestStore_AddProcessedBundles_RemovesOlderHistoryHash_EvenForBlockNumberWithoutBundles(t *testing.T) {
	// This test verifies that as new blocks are added, the history hash of old
	// blocks are removed even if those blocks don't have bundles.

	store, err := NewMemStore(t)
	require.NoError(t, err)
	// Run enough blocks that some 'h' entries from bundle-less blocks get pruned.
	const currentBlock = DefaultProcessedBundlesRetention + 20
	const firstBundle = uint64(10)
	for block := range uint64(currentBlock) + 1 {
		// add a single block with bundles
		if block == firstBundle {
			store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{
				uint64ToHash(block): {},
			})
		} else {
			// add blocks without bundles to advance the block number and trigger pruning
			store.AddProcessedBundles(block, nil)
		}

		// Check that the earliest bundle history block is updated accordingly.
		earliestHashBlockNumber, _, found := store.GetEarliestBundleHistoryHash()
		if block < firstBundle {
			require.False(t, found)
		} else {
			require.True(t, found)

			// Check that the block number of the earliest hash is moving one
			// step at a time.
			// The actual value is not important here, as the correctness of
			// the boundary is tested by other tests, in particular
			// TestStore_ProcessedBundles_RetainsAllHashesToVerifyContainedExecutionPlans
			wantEarliest := firstBundle
			if block >= DefaultProcessedBundlesRetention+firstBundle {
				wantEarliest = block - DefaultProcessedBundlesRetention + 1
			}
			require.Equal(t, wantEarliest, earliestHashBlockNumber)
		}
	}
}

func TestStore_AddProcessedBundles_HistoryHashIsConsistentWithPerBlockHash(t *testing.T) {
	// This test verifies that as new bundles are added the history hash reported
	// by GetLatestProcessedBundleHistoryHash is consistent with the hash stored
	// for each block in the 'h' entries.

	store, err := NewMemStore(t)
	require.NoError(t, err)

	var historicHashes []common.Hash
	for block := range bundle.MaxBlockRangeLength * 2 {
		// randomly add bundles to some blocks, but not all
		if rand.Uint64()%2 == 0 {
			store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{
				uint64ToHash(block): {},
			})
		} else {
			store.AddProcessedBundles(block, nil)
		}

		_, currentHistoryHash := store.GetLatestProcessedBundleHistoryHash()
		historicHashes = append(historicHashes, currentHistoryHash)

		for past := range block + 1 {
			historyHashForBlock, found := store.GetProcessedBundleHistoryHash(past)
			if found {
				require.Equal(t, historicHashes[past], historyHashForBlock)
			}
		}
	}
}

func TestStore_GetLatestProcessedBundleHistoryHash_InitiallyZero(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum, hash := store.GetLatestProcessedBundleHistoryHash()
	require.Zero(blockNum)
	require.Zero(hash)
}

func TestStore_GetLatestProcessedBundleHistoryHash_CorrectlyParsesHash(t *testing.T) {
	store, table, _, _, _ := storeTableLogMocks(t)

	for i := range 2 * bundle.MaxBlockRangeLength {
		block := uint64(i)
		hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("hash for block %d", block)))

		encoded := append(
			bigendian.Uint64ToBytes(block),
			hash.Bytes()...,
		)

		table.EXPECT().Get(nil).Return(encoded, nil)
		gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()

		require.Equal(t, block, gotBlock)
		require.Equal(t, hash, gotHash)
	}
}

func TestStore_GetLatestProcessedBundleHistoryHash_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to get hash of processed bundles", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to get hash of processed bundles: %v", []any{"error", injectedErr}),
		func() { store.GetLatestProcessedBundleHistoryHash() })
}

func TestStore_GetLatestProcessedBundleHistoryHash_LogsOnInvalidStateLength(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	table.EXPECT().Get(gomock.Any()).Return([]byte{1, 2, 3}, nil)
	store.table.ProcessedBundles = table

	expectCrit(log, "invalid state length for processed bundles", "length", 3)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("invalid state length for processed bundles: %v", []any{"length", 3}),
		func() { store.GetLatestProcessedBundleHistoryHash() })
}

func TestStore_GetProcessedBundleHistoryHash_FetchesDataFromTable(t *testing.T) {
	ctrl := gomock.NewController(t)
	table := NewMockstoreTable(ctrl)

	store := &Store{}
	store.table.ProcessedBundles = table

	wantHash := common.Hash{123, 45}
	table.EXPECT().Get(getBundleHistoryHashKey(12)).Return(wantHash[:], nil)

	gotHash, found := store.GetProcessedBundleHistoryHash(12)
	require.True(t, found)
	require.Equal(t, wantHash, gotHash)
}

func TestStore_GetProcessedBundleHistoryHash_ReportsMissingForEmptyStore(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash, found := store.GetProcessedBundleHistoryHash(12)
	require.False(found, "should return false when there is no data")
	require.Zero(hash)
}

func TestStore_GetProcessedBundleHistoryHash_ReportsMissingIfValueIsMalformed(t *testing.T) {
	store, table, _, _, _ := storeTableLogMocks(t)

	// one too short, one too long
	table.EXPECT().Get(getBundleHistoryHashKey(12)).Return([]byte{1, 2, 3}, nil)
	table.EXPECT().Get(getBundleHistoryHashKey(14)).Return([]byte{40: 1}, nil)

	hash, found := store.GetProcessedBundleHistoryHash(12)
	require.False(t, found)
	require.Zero(t, hash)

	hash, found = store.GetProcessedBundleHistoryHash(14)
	require.False(t, found)
	require.Zero(t, hash)
}

func TestStore_GetProcessedBundleHistoryHash_LogCritOnDbReadError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedError := fmt.Errorf("injected unit test issue")
	gomock.InOrder(
		table.EXPECT().Get(getBundleHistoryHashKey(12)).Return(nil, injectedError),
		log.EXPECT().
			Crit("failed to get hash of processed bundles for block", "error", injectedError).
			DoAndReturn(func(string, ...any) {
				panic("stopped deliberately by unit test")
			}),
	)

	require.PanicsWithValue(t,
		"stopped deliberately by unit test",
		func() { store.GetProcessedBundleHistoryHash(12) },
	)
}

func TestStore_GetEarliestBundleHistoryHash_ScansTableForFirstEntry(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	wantBlock := uint64(1234)
	wantHash := common.Hash{1, 2, 3}

	table := NewMockstoreTable(ctrl)
	it := NewMockdbIterator(ctrl)

	store := &Store{}
	store.table.ProcessedBundles = table

	gomock.InOrder(
		table.EXPECT().NewIterator([]byte{'h'}, nil).Return(it),
		it.EXPECT().Next().Return(true),
		it.EXPECT().Key().Return(getBundleHistoryHashKey(wantBlock)),
		it.EXPECT().Value().Return(wantHash.Bytes()),
		it.EXPECT().Release(),
	)

	gotBlock, gotHash, found := store.GetEarliestBundleHistoryHash()
	require.True(found)
	require.Equal(wantBlock, gotBlock)
	require.Equal(wantHash, gotHash)
}

func TestStore_GetEarliestBundleHistoryHash_ReturnsFalseWhenEmpty(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	_, _, ok := store.GetEarliestBundleHistoryHash()
	require.False(ok, "should return false when no bundles have been executed yet")
}

func TestStore_GetEarliestBundleHistoryHash_ReturnsZero_WhenNoBundlesExecuted(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	// add blocks without bundles
	for block := range bundle.MaxBlockRangeLength + 2 {
		store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{})
		blockNum, hash := store.GetLatestProcessedBundleHistoryHash()
		require.Zero(blockNum)
		require.Zero(hash)
	}

	gotBlock, gotHash, ok := store.GetEarliestBundleHistoryHash()
	require.False(ok)
	require.Equal(uint64(0), gotBlock, "oldest block should be 0")
	require.Zero(gotHash, "oldest hash should be zero")
}

func TestStore_GetEarliestBundleHistoryHash_IterationErrorResultsInCriticalLog(t *testing.T) {
	store, table, log, _, iter := storeTableLogMocks(t)

	injectedError := fmt.Errorf("injected")

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter),
		iter.EXPECT().Next().Return(false),
		iter.EXPECT().Error().Return(injectedError),
		log.EXPECT().Crit("failed to iterate bundle history hashes", "error", injectedError).
			DoAndReturn(func(string, ...any) { panic("forced stop by test") }),
		// Release would not be called if Crit kills the process, but since we
		// only panic, it will be called.
		iter.EXPECT().Release(),
	)

	require.PanicsWithValue(t, "forced stop by test", func() {
		store.GetEarliestBundleHistoryHash()
	})
}

func TestStore_GetEarliestBundleHistoryHash_InvalidKeyLengthResultsInCriticalLog(t *testing.T) {
	store, table, log, _, iter := storeTableLogMocks(t)

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter),
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return([]byte{1, 2, 3}), // invalid size
		log.EXPECT().Crit("invalid per-block history hash key length", "length", 3).
			DoAndReturn(func(string, ...any) { panic("forced stop by test") }),
		// Release would not be called if Crit kills the process, but since we
		// only panic, it will be called.
		iter.EXPECT().Release(),
	)

	require.PanicsWithValue(t, "forced stop by test", func() {
		store.GetEarliestBundleHistoryHash()
	})
}

func TestStore_GetEarliestBundleHistoryHash_InvalidValueLengthResultsInCriticalLog(t *testing.T) {
	store, table, log, _, iter := storeTableLogMocks(t)

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter),
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(make([]byte, 9)),
		iter.EXPECT().Value().Return([]byte{1, 2, 3}), // invalid value length
		log.EXPECT().Crit("invalid per-block history hash value length", "length", 3).
			DoAndReturn(func(string, ...any) { panic("forced stop by test") }),
		// Release would not be called if Crit kills the process, but since we
		// only panic, it will be called.
		iter.EXPECT().Release(),
	)

	require.PanicsWithValue(t, "forced stop by test", func() {
		store.GetEarliestBundleHistoryHash()
	})
}

func TestStore_addNewBundles_EncodesInfoCorrectly(t *testing.T) {
	store, _, _, _, _ := storeTableLogMocks(t)

	for blockNum := range 200 {
		hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("hash for block %d", blockNum)))
		info := bundle.ExecutionInfo{
			ExecutionPlanHash: hash,
			BlockNumber:       uint64(blockNum),
			Position: bundle.PositionInBlock{
				Offset: 4,
				Count:  5,
			},
		}

		batch := NewMockstoreBatch(gomock.NewController(t))
		batch.EXPECT().Put(getEntryKey(hash), BundleExecutionInfoMatcher{expected: info})
		batch.EXPECT().Put(getIndexKey(uint64(blockNum), hash), []byte{0})

		infoMap := map[common.Hash]bundle.PositionInBlock{
			hash: info.Position,
		}
		_, err := store.addNewBundles(uint64(blockNum), infoMap, batch)
		require.NoError(t, err)
	}
}

func TestStore_addNewBundles_ReturnsExpectedHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	cases := map[string]struct {
		executedBundles map[common.Hash]bundle.PositionInBlock
	}{
		"empty map": {
			executedBundles: map[common.Hash]bundle.PositionInBlock{},
		},
		"single entry": {
			executedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {Offset: 4, Count: 5},
			},
		},
		"two entries": {
			executedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {Offset: 4, Count: 5},
				{4, 5, 6}: {Offset: 6, Count: 7},
			},
		},
		"more than two entries": {
			executedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {Offset: 4, Count: 5},
				{4, 5, 6}: {Offset: 6, Count: 7},
				{7, 8, 9}: {Offset: 8, Count: 9},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			batch := NewMockstoreBatch(gomock.NewController(t))
			batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).
				Times(2 * len(c.executedBundles)) // 2 times per hash
			addedHash, err := store.addNewBundles(1, c.executedBundles, batch)
			require.NoError(err)

			expectedHash := common.Hash{}
			for hash := range c.executedBundles {
				expectedHash = xorHash(expectedHash, hash)
			}
			require.Equal(expectedHash, addedHash)
		})
	}
}

func TestStore_addNewBundles_ReportsError_OnBatchPutError(t *testing.T) {
	store, _, _, batch, _ := storeTableLogMocks(t)

	injectedErrEntry := errors.New("entry put error")
	injectedErrIndex := errors.New("index put error")
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErrEntry)
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErrIndex)

	_, err := store.addNewBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1, 2, 3}: {Offset: 4, Count: 5},
	}, batch)
	require.ErrorContains(t, err, "failed to add processed bundle hash to batch")
	require.ErrorIs(t, err, injectedErrEntry)
	require.ErrorIs(t, err, injectedErrIndex)
}

func TestStore_deleteOutdatedBundles_RemovesBundles_WhenOld(t *testing.T) {

	caseTable := []struct {
		storedBundleBlockNumber uint64
		finishingBlock          uint64
		expectDeleted           bool
	}{
		// Following cases are the warm up phase of the storage
		// when current block number is not large enough to have a history to delete
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          DefaultProcessedBundlesRetention - 2,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          DefaultProcessedBundlesRetention - 2,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          DefaultProcessedBundlesRetention - 1,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention / 2,
			finishingBlock:          DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention - 1,
			finishingBlock:          DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention,
			finishingBlock:          DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		// Following cases are after the warm up phase, when current block
		// number is large enough to have a history to delete,
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          DefaultProcessedBundlesRetention - 1,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention / 2,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention - 1,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention + 1,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           true,
		},
		// Following cases are recent enough to not be deleted
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention + 2,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: DefaultProcessedBundlesRetention * 3 / 2,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2*DefaultProcessedBundlesRetention - 1,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2 * DefaultProcessedBundlesRetention,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
		// future block numbers should not cause deletion
		{
			storedBundleBlockNumber: 2*DefaultProcessedBundlesRetention + 1,
			finishingBlock:          2 * DefaultProcessedBundlesRetention,
			expectDeleted:           false,
		},
	}

	for _, c := range caseTable {
		name := fmt.Sprintf("storedBlock=%d/currentBlock=%d", c.storedBundleBlockNumber, c.finishingBlock)
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			batch := NewMockstoreBatch(ctrl)
			table := NewMockstoreTable(ctrl)
			it := NewMockdbIterator(ctrl)
			if c.finishingBlock >= DefaultProcessedBundlesRetention-1 {
				it.EXPECT().Release()
			}
			store := &Store{}
			store.cfg.ProcessedBundlesRetention = DefaultProcessedBundlesRetention
			store.table.ProcessedBundles = table

			existingBundleHash := common.Hash{1, 2, 3}

			// The algorithm would not contemplate any history
			// when the block number is short enough to not to require any cleanup
			if c.finishingBlock >= DefaultProcessedBundlesRetention-1 {
				it2 := NewMockdbIterator(ctrl)
				it2.EXPECT().Release()

				existingBundleKey := getIndexKey(c.storedBundleBlockNumber, existingBundleHash)
				gomock.InOrder(
					table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it),
					it.EXPECT().Next().Return(true),
					it.EXPECT().Key().Return(existingBundleKey),
					// AnyTimes: when entries are not to be deleted, the iteration is stopped
					it.EXPECT().Next().Return(false).AnyTimes(),
					it.EXPECT().Error().Return(nil),
				)
				// Second pass: 'h' iterator for bundle-less blocks (empty in mock world).
				table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(0)).Return(it2)
				it2.EXPECT().Next().Return(false)
				it2.EXPECT().Error().Return(nil)

				// this expectation is the core of the test:
				// it checks that the delete calls are made if and only if the
				// existing bundle is old enough to be deleted.
				if c.expectDeleted {
					batch.EXPECT().Delete(getIndexKey(c.storedBundleBlockNumber, existingBundleHash))
					batch.EXPECT().Delete(getEntryKey(existingBundleHash))
				}
			}

			err := store.deleteOutdatedBundles(c.finishingBlock, batch)
			require.NoError(t, err)
		})
	}
}

func TestStore_deleteOutdatedBundles_RemovesMultipleEntries_WhenNotCleanedForTooLong(t *testing.T) {
	ctrl := gomock.NewController(t)
	batch := NewMockstoreBatch(ctrl)
	table := NewMockstoreTable(ctrl)

	store := &Store{}
	store.cfg.ProcessedBundlesRetention = DefaultProcessedBundlesRetention
	store.table.ProcessedBundles = table

	it := NewMockdbIterator(ctrl)
	it.EXPECT().Release()
	table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it)

	for i := range 10 {
		it.EXPECT().Next().Return(true)
		it.EXPECT().Key().Return(getIndexKey(uint64(i), uint64ToHash(uint64(i))))
		batch.EXPECT().Delete(gomock.Any()) // index key
		batch.EXPECT().Delete(gomock.Any()) // entry key
	}
	it.EXPECT().Error().Return(nil)
	it.EXPECT().Next().Return(false)

	// Second pass: 'h' iterator for bundle-less blocks (empty in mock world).
	it2 := NewMockdbIterator(ctrl)
	it2.EXPECT().Error().Return(nil)
	it2.EXPECT().Release()
	table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(0)).Return(it2)
	it2.EXPECT().Next().Return(false)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+10, batch)
	require.NoError(t, err)
}

func TestStore_deleteOutdatedBundles_IgnoresIndexKeysOfWrongLength(t *testing.T) {
	// log mock is ignored because no log called should be triggered.
	store, table, _, batch, it := storeTableLogMocks(t)

	// 'i' iterator: returns a key of wrong length, then exhausts.
	gomock.InOrder(
		it.EXPECT().Next().Return(true),
		// This is the key that will be ignored, since it does not have the correct length.
		it.EXPECT().Key().Return([]byte{1, 2}),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(nil),
		it.EXPECT().Release(),
	)
	table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it)

	// 'h' iterator: empty in the mock world.
	it2 := NewMockdbIterator(gomock.NewController(t))
	it2.EXPECT().Next().Return(false)
	it2.EXPECT().Error().Return(nil)
	it2.EXPECT().Release()
	table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(0)).Return(it2)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+1, batch)
	require.NoError(t, err)
}

func TestStore_deleteOutdatedBundles_IgnoresHashKeysOfWrongLength(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	ctrl := gomock.NewController(t)
	it2 := NewMockdbIterator(ctrl)

	// 'h' iterator: returns a key of wrong length, then exhausts.
	gomock.InOrder(
		table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0)).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(nil),
		it.EXPECT().Release(),
		table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(0)).Return(it2),
		it2.EXPECT().Next().Return(true),
		// This is the key that will be ignored, since it does not have the correct length.
		it2.EXPECT().Key().Return([]byte{1, 2, 3}), // wrong length
		it2.EXPECT().Next().Return(false),
		it2.EXPECT().Error().Return(nil),
		it2.EXPECT().Release(),
	)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+1, batch)
	require.NoError(t, err)
}

func TestStore_deleteOutdatedBundles_ReportsError_OnBatchDeleteError(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	injectedErrDeleteEntry := errors.New("entry delete error")
	injectedErrDeleteIndex := errors.New("index delete error")
	batch.EXPECT().Delete(gomock.Any()).Return(injectedErrDeleteEntry)
	batch.EXPECT().Delete(gomock.Any()).Return(injectedErrDeleteIndex)

	gomock.InOrder(
		it.EXPECT().Next().Return(true),
		it.EXPECT().Key().Return(getIndexKey(1, common.Hash{1, 2, 3})),
		it.EXPECT().Release(),
	)
	table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+1, batch)
	require.ErrorContains(t, err, "failed to delete processed bundle hash")
	require.ErrorIs(t, err, injectedErrDeleteEntry)
	require.ErrorIs(t, err, injectedErrDeleteIndex)
}

func TestStore_deleteOutdatedBundles_ReportsError_OnIterationError(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	injectedError := fmt.Errorf("injected issue")

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(injectedError),
		it.EXPECT().Release(),
	)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+12, batch)
	require.ErrorContains(t, err, "failed to iterate processed bundles for deletion")
	require.ErrorIs(t, err, injectedError)
}

func TestStore_deleteOutdatedBundles_ReportsError_WhenDeletingHashesFails(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	ctrl := gomock.NewController(t)
	it2 := NewMockdbIterator(ctrl)

	injectedError := fmt.Errorf("injected issue")

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(nil),
		it.EXPECT().Release(),
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it2),
		it2.EXPECT().Next().Return(true),
		it2.EXPECT().Key().Return(make([]byte, 9)),
		batch.EXPECT().Delete(gomock.Any()).Return(injectedError),
		it2.EXPECT().Release(),
	)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+12, batch)
	require.ErrorContains(t, err, "failed to delete block history hash")
	require.ErrorIs(t, err, injectedError)
}

func TestStore_deleteOutdatedBundles_ReportsError_OnSecondIterationError(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)

	ctrl := gomock.NewController(t)
	it2 := NewMockdbIterator(ctrl)

	injectedError := fmt.Errorf("injected issue")

	gomock.InOrder(
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(nil),
		it.EXPECT().Release(),
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it2),
		it2.EXPECT().Next().Return(false),
		it2.EXPECT().Error().Return(injectedError),
		it2.EXPECT().Release(),
	)

	err := store.deleteOutdatedBundles(DefaultProcessedBundlesRetention+12, batch)
	require.ErrorContains(t, err, "failed to iterate block history hashes for deletion")
	require.ErrorIs(t, err, injectedError)
}

func TestStore_xorHash_ReturnsExpectedResult(t *testing.T) {
	require := require.New(t)

	xorForTest := func(a, b common.Hash) common.Hash {
		var res common.Hash
		for i := 0; i < len(res); i++ {
			res[i] = a[i] ^ b[i]
		}
		return res
	}

	cases := map[string]struct {
		hash1    common.Hash
		hash2    common.Hash
		expected common.Hash
	}{
		"all zeros": {
			hash1:    common.Hash{0, 0, 0},
			hash2:    common.Hash{0, 0, 0},
			expected: common.Hash{0, 0, 0},
		},
		"zero and non-zero": {
			hash1:    common.Hash{0, 0, 0},
			hash2:    common.Hash{7, 8, 9},
			expected: common.Hash{7, 8, 9},
		},
		"non-zero and zero": {
			hash1:    common.Hash{10, 11, 12},
			hash2:    common.Hash{0, 0, 0},
			expected: common.Hash{10, 11, 12},
		},
		"same non-zero": {
			hash1:    common.Hash{1, 1, 1},
			hash2:    common.Hash{1, 1, 1},
			expected: common.Hash{0, 0, 0},
		},
		"operation with 0xff": {
			hash1:    common.Hash{0xff, 0xff, 0xff},
			hash2:    common.Hash{0x1, 0x2, 0x3},
			expected: common.Hash{0xfe, 0xfd, 0xfc},
		},
		"32 bytes computed": {
			hash1:    common.Hash{0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			hash2:    common.Hash{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			expected: common.Hash{1, 0, 3, 2, 5, 4, 6, 9, 8, 11, 10, 13, 12, 15, 14, 17, 16, 19, 18, 21, 20, 23, 22, 25, 24, 27, 26, 29, 28, 31, 30, 1},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			expectedXor := xorForTest(c.hash1, c.hash2)
			require.Equal(expectedXor, c.expected)
			require.Equal(c.expected, xorHash(c.hash1, c.hash2))
		})
	}
}

func TestStore_computeNewBundleStateHash_CorrectlyProcessesEdgeCases(t *testing.T) {
	// this test checks that the computeNewBundleStateHash function correctly processes edge cases, such as:
	//  - blockNum being zero or very large
	//  - oldHash and addedHash having specific patterns (e.g., all zeros, all 0xff, etc.)
	//  - combinations of the above

	hashDomain := []common.Hash{
		{},
		{4, 5, 6},
		{0xff, 0xff, 0xff},
		common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	}
	blockNumberDomain := []uint64{
		0, 1, 512,
		bundle.MaxBlockRangeLength - 1,
		bundle.MaxBlockRangeLength,
		bundle.MaxBlockRangeLength + 1,
		math.MaxUint64}

	type testCase struct {
		oldHash   common.Hash
		addedHash common.Hash
		blockNum  uint64
	}
	testCases := map[string]testCase{}
	for _, oldHash := range hashDomain {
		for _, addedHash := range hashDomain {
			for _, blockNum := range blockNumberDomain {
				name := fmt.Sprintf("oldHash=%s/addedHash=%s/blockNum=%d",
					oldHash.Hex(), addedHash.Hex(), blockNum)
				testCases[name] = testCase{
					oldHash:   oldHash,
					addedHash: addedHash,
					blockNum:  blockNum,
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := computeNewBundleStateHash(tc.oldHash, tc.addedHash, tc.blockNum)
			ref := referenceComputeStateHash(tc.oldHash, tc.addedHash, tc.blockNum)
			require.Equal(t, ref, got, "actual implementation should match alternative implementation")
		})
	}
}

func TestStore_GetEntryKey_ReturnsExpectedKey(t *testing.T) {
	require := require.New(t)

	hash := common.Hash{1, 2, 3}
	expectedKey := append([]byte{'e'}, hash.Bytes()...)
	got := getEntryKey(hash)
	require.Equal(expectedKey, got)
	require.Len(got, 1+32) // 1 byte for prefix + 32 bytes for hash
}

func TestStore_GetBundleHistoryHashKey_ReturnsExpectedKey(t *testing.T) {
	blockNumbers := []uint64{0, 1, 512, math.MaxUint64 - 1, math.MaxUint64}
	for _, blockNum := range blockNumbers {
		t.Run(fmt.Sprintf("blockNum=%d", blockNum), func(t *testing.T) {
			expectedKey := append([]byte{'h'}, make([]byte, 8)...)
			binary.BigEndian.PutUint64(expectedKey[1:9], blockNum)
			got := getBundleHistoryHashKey(blockNum)
			require.Equal(t, expectedKey, got)
			require.Len(t, got, 1+8) // 1 byte prefix + 8 bytes block number
		})
	}
}

func TestStore_GetIndexKey_ReturnsExpectedKey(t *testing.T) {
	hash := common.HexToHash("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	blockNumbers := []uint64{0, 1, 512, math.MaxUint64 - 1, math.MaxUint64}
	for _, blockNum := range blockNumbers {
		t.Run(fmt.Sprintf("blockNum=%d", blockNum), func(t *testing.T) {
			expectedKey := append([]byte{'i'}, make([]byte, 8)...)
			binary.BigEndian.PutUint64(expectedKey[1:9], blockNum)
			expectedKey = append(expectedKey, hash.Bytes()...)
			got := getIndexKey(blockNum, hash)
			require.Equal(t, expectedKey, got)
			// 1 byte for prefix + 8 bytes for block number + 32 bytes for hash
			require.Len(t, got, 1+8+32)
		})
	}
}

func TestStore_ProcessedBundles_TablesAreInitiallyEmpty(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	iter := store.table.ProcessedBundles.NewIterator(nil, nil)
	require.False(iter.Next())
	require.NoError(iter.Error())

	iter = store.table.ProcessedBundles.NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(0))
	require.False(iter.Next())
	require.NoError(iter.Error())

	iter = store.table.ProcessedBundles.NewIterator([]byte{'e'}, nil)
	require.False(iter.Next())
	require.NoError(iter.Error())
}

func TestStore_ProcessedBundles_ZeroHistoryHashIsPreserved_WhenNoBundlesAreExecuted(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	for i := range bundle.MaxBlockRangeLength + 2 {
		// nil bundles executed
		store.AddProcessedBundles(i, nil)
		blockNum, hash := store.GetLatestProcessedBundleHistoryHash()
		require.Equal(uint64(0), blockNum)
		require.Equal(common.Hash{}, hash)

		// empty block bundles executed
		store.AddProcessedBundles(i, map[common.Hash]bundle.PositionInBlock{})
		blockNum, hash = store.GetLatestProcessedBundleHistoryHash()
		require.Equal(uint64(0), blockNum)
		require.Equal(common.Hash{}, hash)

	}
}

func TestStore_ProcessedBundles_UpdatesHistoryHash(t *testing.T) {
	require := require.New(t)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	cases := map[string]struct {
		bundles map[common.Hash]bundle.PositionInBlock
	}{
		"single new bundle": {
			bundles: map[common.Hash]bundle.PositionInBlock{
				hash1: {Offset: 0, Count: 1},
			},
		},
		"multiple new bundles": {
			bundles: map[common.Hash]bundle.PositionInBlock{
				hash2: {Offset: 0, Count: 1},
				hash3: {Offset: 1, Count: 1},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(err)
			_, initialHash := store.GetLatestProcessedBundleHistoryHash()

			store.AddProcessedBundles(1, tc.bundles)
			addedHash := common.Hash{}
			for hash := range tc.bundles {
				addedHash = xorHash(addedHash, hash)
			}
			expectedHash := referenceComputeStateHash(initialHash, addedHash, 1)
			_, gotHash := store.GetLatestProcessedBundleHistoryHash()
			require.Equal(expectedHash, gotHash)
		})
	}
}

func TestStore_ProcessedBundles_CommutativityOfAddedBundles(t *testing.T) {
	require := require.New(t)
	store1, err := NewMemStore(t)
	require.NoError(err)
	store2, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}

	store1.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		hash1: {Offset: 0, Count: 1},
		hash2: {Offset: 1, Count: 1},
	})

	store2.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		hash2: {Offset: 1, Count: 1},
		hash1: {Offset: 0, Count: 1},
	})

	_, hashA := store1.GetLatestProcessedBundleHistoryHash()
	_, hashB := store2.GetLatestProcessedBundleHistoryHash()
	require.Equal(hashA, hashB)
}

func TestStore_ProcessedBundles_HashIsUpdatedWithNewBlocks(t *testing.T) {
	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	// this test relies on the incremental nature uint64ToHash to
	// generate distinct hashes which also yield different xor values
	seenHashes := make(map[common.Hash]struct{})
	for i := range 4 * bundle.MaxBlockRangeLength {
		store.AddProcessedBundles(i, map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(uint64(i)): {Offset: 0, Count: 1},
		})

		block, got := store.GetLatestProcessedBundleHistoryHash()
		require.Equal(uint64(i), block)
		require.NotContains(seenHashes, got)
		seenHashes[got] = struct{}{}
	}
}

func TestStore_ProcessedBundles_RetainsAllBundlesRequiredToCoverTheMaximumBlockRange(t *testing.T) {
	require := require.New(t)
	numBlocks := 3 * bundle.MaxBlockRangeLength

	store, err := NewMemStore(t)
	require.NoError(err)

	// While progressing through the blocks, all execution plans must be retained
	// until their maximum block range has expired.
	for currentBlockNumber := range numBlocks {

		// Check that the store covers all plans of the past that are allowed
		// to be included in the current block (before adding it). Plans are
		// retained for DefaultProcessedBundlesRetention blocks, which exceeds the
		// maximum block range, so expired plans may still be present.
		for block := uint64(0); block < currentBlockNumber; block++ {
			blockRange := bundle.MakeMaxRangeStartingAt(block)
			if blockRange.IsInRange(currentBlockNumber) {
				require.True(
					store.HasBundleRecentlyBeenProcessed(uint64ToHash(block)),
					"Current block %d, checking plan with range %v",
					currentBlockNumber, blockRange,
				)
			}
		}

		store.AddProcessedBundles(currentBlockNumber, map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(currentBlockNumber): {},
		})
	}
}

func TestStore_ProcessedBundles_RetainsAllHashesToVerifyContainedExecutionPlans(t *testing.T) {
	// This test makes sure that exactly those hashes are retained in the store
	// that are required to verify the stored execution plan hashes.
	require := require.New(t)
	numBlocks := 3 * bundle.MaxBlockRangeLength

	store, err := NewMemStore(t)
	require.NoError(err)

	for currentBlockNumber := range numBlocks {

		// Verify that for exactly those blocks that are still in the store the
		// hashes required for their verification are also in the store.
		for b := range currentBlockNumber {

			_, hashPresent := store.GetProcessedBundleHistoryHash(b)

			// A hash should be present if:
			//  - the plans for the associated block are still in the store
			//  - the plans for the succeeding block are still in the store
			thisInStore := store.HasBundleRecentlyBeenProcessed(uint64ToHash(b))
			nextInStore := store.HasBundleRecentlyBeenProcessed(uint64ToHash(b + 1))

			wantHash := thisInStore || nextInStore
			require.Equal(
				wantHash, hashPresent,
				"Current block %d, checking presence of hash for block %d, plans for block %d in store: %t, plans for block %d in store: %t",
				currentBlockNumber, b, b, thisInStore, b+1, nextInStore,
			)
		}

		store.AddProcessedBundles(currentBlockNumber, map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(currentBlockNumber): {},
		})
	}
}

func TestStore_SetProcessedBundlesHistoryHash_WritesBundlesHistoryHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum := uint64(10)
	hash := common.Hash{1, 2, 3}

	store.SetProcessedBundlesHistoryHash(blockNum, hash)

	resBlockNum, resHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(blockNum, resBlockNum)
	require.Equal(hash, resHash)
}

func TestStore_SetProcessedBundlesHistoryHash_LogsOnPutError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("put error")
	table.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErr)

	expectCrit(log, "failed to set hash of processed bundles", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to set hash of processed bundles: %v", []any{"error", injectedErr}),
		func() { store.SetProcessedBundlesHistoryHash(1, common.Hash{1, 2, 3}) })
}

func TestStore_EnumerateProcessedBundles_ReturnsEmptySliceWhenNoEntries(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	exportedEntries := store.EnumerateProcessedBundles()
	require.NotNil(exportedEntries)
	require.Empty(exportedEntries, "expected no exported entries when store is empty")
}

func TestStore_EnumerateProcessedBundles_ReturnsAllAddedEntries(t *testing.T) {

	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	expected := map[common.Hash]bundle.ExecutionInfo{}
	// fill the store with the maximum number of block
	for i := range DefaultProcessedBundlesRetention + 1 {
		hash := common.BytesToHash(bigendian.Uint32ToBytes(uint32(i)))
		position := bundle.PositionInBlock{Offset: uint32(i), Count: 1}
		executedBundles := map[common.Hash]bundle.PositionInBlock{
			hash: position,
		}
		store.AddProcessedBundles(uint64(i), executedBundles)
		if i > 1 {
			expected[hash] = bundle.ExecutionInfo{
				ExecutionPlanHash: hash,
				BlockNumber:       uint64(i),
				Position:          position,
			}
		}
	}
	block, historyHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(uint64(DefaultProcessedBundlesRetention), block)
	require.NotNil(historyHash)
	require.NotZero(historyHash)

	entries := store.EnumerateProcessedBundles()
	// DefaultProcessedBundlesRetention-1 entries (oldest was pruned)
	require.Len(entries, int(DefaultProcessedBundlesRetention-1))

	// Verify the returned entries match the expected infos
	got := make(map[common.Hash]bundle.ExecutionInfo, len(entries))
	for _, entry := range entries {
		got[entry.ExecutionPlanHash] = entry
	}
	require.Equal(expected, got)
}

func TestStore_EnumerateProcessedBundles_LogsOnCrit_IteratorError(t *testing.T) {
	store, table, log, _, it := storeTableLogMocks(t)

	injectedErr := errors.New("iterator error")
	gomock.InOrder(
		table.EXPECT().NewIterator([]byte{'e'}, nil).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(injectedErr).Times(2),
		it.EXPECT().Release(),
	)
	expectCrit(log, "failed to export processed bundles", "error", injectedErr)

	require.PanicsWithValue(t,
		fmt.Sprintf("failed to export processed bundles: %v", []any{"error", injectedErr}),
		func() { store.EnumerateProcessedBundles() })
}

func TestStore_EnumerateProcessedBundles_LogsOnCrit_IteratorWrongSize(t *testing.T) {

	tests := map[string]struct {
		key   []byte
		value []byte
	}{
		"short key": {
			key:   []byte{1, 2}, // should be 33 bytes for 'e' + hash
			value: make([]byte, 16),
		},
		"short value": {
			key:   append([]byte{'e'}, common.Hash{1, 2, 3}.Bytes()...),
			value: make([]byte, 15), // should be 16 bytes
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			store, table, log, _, it := storeTableLogMocks(t)

			gomock.InOrder(
				table.EXPECT().NewIterator([]byte{'e'}, nil).Return(it),
				it.EXPECT().Next().Return(true),
				it.EXPECT().Key().Return(tc.key),
				it.EXPECT().Value().Return(tc.value),
				it.EXPECT().Release(),
			)
			expectCrit(
				log,
				"invalid key or value length for processed bundle entry during export",
				"keyLength", len(tc.key),
				"valueLength", len(tc.value))

			require.PanicsWithValue(t,
				fmt.Sprintf("invalid key or value length for processed bundle entry during export: %v",
					[]any{"keyLength", fmt.Sprintf("%d", len(tc.key)),
						"valueLength", fmt.Sprintf("%d", len(tc.value))}),
				func() { store.EnumerateProcessedBundles() })

		})
	}
}

func TestStore_ProcessedBundles_HistoryHashConverges_ForStoresStartingAtDifferentBlocks(t *testing.T) {
	// This test verifies that two stores processing the same bundles converge
	// to the same history hash regardless of where each store begins.
	require := require.New(t)

	blockHistory := map[uint64]map[common.Hash]bundle.PositionInBlock{
		0: nil,
		1: nil,
		2: nil,
		3: nil,
		4: nil,
		5: nil,
		6: nil,
		7: nil,
		8: nil,
		9: nil,
		10: {
			common.Hash{1, 2, 3}: {Offset: 0, Count: 1},
		},
		11: nil,
	}

	store1, err := NewMemStore(t)
	require.NoError(err)

	// Step 1: Store 1 processes blocks 0-9 without bundles
	for block := range uint64(10) {
		store1.AddProcessedBundles(uint64(block), blockHistory[block])
		_, hash := store1.GetLatestProcessedBundleHistoryHash()
		// Check 1: History hash remains zero before the first bundle is executed.
		require.Equal(common.Hash{}, hash,
			"store1 history hash should remain zero before first bundle at block %d", block)
	}

	// first bundle is executed at block 10.
	store1.AddProcessedBundles(10, blockHistory[10])
	_, hashAt10 := store1.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(common.Hash{}, hashAt10, "store1 hash should be non-zero after first bundle at block 10")

	store1.AddProcessedBundles(11, blockHistory[11])
	_, hashAt11 := store1.GetLatestProcessedBundleHistoryHash()

	require.NotEqual(hashAt10, hashAt11, "after the first bundle is executed history hash should always vary")

	// Step 2: Store 2 starts at block 5 with history hash zero, then runs the same blocks.
	store2, err := NewMemStore(t)
	require.NoError(err)

	for block := uint64(5); block < 10; block++ {
		store2.AddProcessedBundles(uint64(block), blockHistory[block])
		_, hash := store2.GetLatestProcessedBundleHistoryHash()
		require.Equal(common.Hash{}, hash,
			"store2 history hash should remain zero before first bundle at block %d", block)
	}

	store2.AddProcessedBundles(10, blockHistory[10])
	_, store2HashAt10 := store2.GetLatestProcessedBundleHistoryHash()

	store2.AddProcessedBundles(11, blockHistory[11])
	_, store2HashAt11 := store2.GetLatestProcessedBundleHistoryHash()

	// Check 2: Both stores share the same history hash from block 10 to 11.
	require.Equal(hashAt10, store2HashAt10, "both stores should have the same hash at block 10")
	require.Equal(hashAt11, store2HashAt11, "both stores should have the same hash at block 11")
}

func TestStore_ProcessedBundles_EmptyBundlesChangeHash_AfterFirstBundleExecuted(t *testing.T) {
	// This test verifies that once the history hash becomes non-zero
	// (i.e. after the first bundle is executed), adding an empty bundle list
	// or nil still causes the hash to change.
	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	// Adding nil or empty bundles before the first execution keeps the hash zero.
	store.AddProcessedBundles(1, nil)
	_, hash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(common.Hash{}, hash, "hash should remain zero with nil bundles before first execution")

	store.AddProcessedBundles(2, map[common.Hash]bundle.PositionInBlock{})
	_, hash = store.GetLatestProcessedBundleHistoryHash()
	require.Equal(common.Hash{}, hash, "hash should remain zero with empty map before first execution")

	// Execute the first bundle — hash becomes non-zero.
	store.AddProcessedBundles(3, map[common.Hash]bundle.PositionInBlock{
		{1, 2, 3}: {Offset: 0, Count: 1},
	})
	_, hashAfterFirstBundle := store.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(common.Hash{}, hashAfterFirstBundle, "hash should be non-zero after first bundle")

	// Once the hash is non-zero, nil and empty bundle lists must still advance the hash.
	store.AddProcessedBundles(4, nil)
	_, hashAfterNil := store.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(hashAfterFirstBundle, hashAfterNil,
		"nil bundle list should change the hash once history hash is non-zero")

	store.AddProcessedBundles(5, map[common.Hash]bundle.PositionInBlock{})
	_, hashAfterEmpty := store.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(hashAfterNil, hashAfterEmpty,
		"empty bundle map should change the hash once history hash is non-zero")
}

func TestStore_ProcessedBundles_HistoryHashRemainsZero_WhenNoBundlesAreEverProcessed(t *testing.T) {
	// This test verifies that the history hash stays zero across an arbitrary
	// number of blocks when no bundles are ever executed, including blocks
	// that would trigger the pruning of old entries.

	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	for block := range 3 * bundle.MaxBlockRangeLength {
		store.AddProcessedBundles(uint64(block), nil)
		_, hash := store.GetLatestProcessedBundleHistoryHash()
		require.Equal(common.Hash{}, hash,
			"history hash should remain zero at block %d (nil)", block)

		store.AddProcessedBundles(uint64(block), map[common.Hash]bundle.PositionInBlock{})
		_, hash = store.GetLatestProcessedBundleHistoryHash()
		require.Equal(common.Hash{}, hash,
			"history hash should remain zero at block %d (empty map)", block)
	}
}

func TestStore_ProcessedBundles_DeletingEntries_DoesNotAffectHistoryHash(t *testing.T) {
	// This test verifies that the automatic pruning of old bundle entries
	// (triggered once enough blocks have passed) does not alter the history hash.

	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	// Execute the first bundle at block 0 so the history hash becomes non-zero.
	firstBundleHash := common.Hash{1, 2, 3}
	store.AddProcessedBundles(0, map[common.Hash]bundle.PositionInBlock{
		firstBundleHash: {Offset: 0, Count: 1},
	})
	_, hashAt0 := store.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(common.Hash{}, hashAt0)

	// Compute the expected hash by replaying the same updates manually,
	// independent of any internal pruning.
	expectedHash := hashAt0
	for block := uint64(1); block <= DefaultProcessedBundlesRetention; block++ {
		expectedHash = referenceComputeStateHash(expectedHash, common.Hash{}, block)
	}

	// Advance enough blocks to trigger pruning of the entry at block 0.
	for block := uint64(1); block <= DefaultProcessedBundlesRetention; block++ {
		store.AddProcessedBundles(block, nil)
	}

	// The entry from block 0 must have been pruned.
	require.False(store.HasBundleRecentlyBeenProcessed(firstBundleHash),
		"entry from block 0 should have been pruned after DefaultProcessedBundlesRetention blocks")

	// The history hash must match the manually computed value — pruning must not affect it.
	_, actualHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(expectedHash, actualHash,
		"history hash should not be affected by the deletion of old entries")
}

func TestStore_EarliestBundleRollbackBlock_IsNotLimited_WithoutHistory(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)

	_, limited := store.EarliestBundleRollbackBlock()
	require.False(t, limited)
}

func TestStore_EarliestBundleRollbackBlock_CoversReplayProtectionWindow(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	store.AddProcessedBundles(10, map[common.Hash]bundle.PositionInBlock{{0x01}: {}})
	store.AddProcessedBundles(11, nil)

	earliest, limited := store.EarliestBundleRollbackBlock()
	require.True(t, limited)
	require.Equal(t, 10+bundle.MaxBlockRangeLength, earliest)
}

func TestStore_RollbackProcessedBundles_RestoresStateOfEarlierBlock(t *testing.T) {
	require := require.New(t)
	bundlesOf := func(block uint64) map[common.Hash]bundle.PositionInBlock {
		if block%3 != 0 {
			return nil
		}
		return map[common.Hash]bundle.PositionInBlock{uint64ToHash(block): {}}
	}
	const rollbackBlock, headBlock = uint64(30), uint64(50)

	reference, err := NewMemStore(t)
	require.NoError(err)
	for block := uint64(1); block <= rollbackBlock; block++ {
		reference.AddProcessedBundles(block, bundlesOf(block))
	}

	store, err := NewMemStore(t)
	require.NoError(err)
	for block := uint64(1); block <= headBlock; block++ {
		store.AddProcessedBundles(block, bundlesOf(block))
	}
	require.NoError(store.RollbackProcessedBundles(rollbackBlock))

	// the store looks as if it never processed blocks after the rollback block
	require.Equal(reference.EnumerateProcessedBundles(), store.EnumerateProcessedBundles())
	wantBlock, wantHash := reference.GetLatestProcessedBundleHistoryHash()
	gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(wantBlock, gotBlock)
	require.Equal(wantHash, gotHash)
	for block := rollbackBlock + 1; block <= headBlock; block++ {
		_, found := store.GetProcessedBundleHistoryHash(block)
		require.False(found, "history hash of block %d should be removed", block)
		require.False(store.HasBundleRecentlyBeenProcessed(uint64ToHash(block)))
	}

	// re-processing the rolled back blocks yields the same history
	for block := rollbackBlock + 1; block <= headBlock; block++ {
		reference.AddProcessedBundles(block, bundlesOf(block))
		store.AddProcessedBundles(block, bundlesOf(block))
	}
	_, wantHash = reference.GetLatestProcessedBundleHistoryHash()
	_, gotHash = store.GetLatestProcessedBundleHistoryHash()
	require.Equal(wantHash, gotHash)
}

func TestStore_RollbackProcessedBundles_ReportsError_WithoutWriting_WhenDeletionFails(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)
	ctrl := gomock.NewController(t)
	earliestIt := NewMockdbIterator(ctrl)
	hashesIt := NewMockdbIterator(ctrl)

	const block = uint64(5)
	injectedErr := errors.New("injected iteration error")

	// the history hash of the rollback block is present
	table.EXPECT().NewIterator([]byte{'h'}, nil).Return(earliestIt)
	earliestIt.EXPECT().Next().Return(true)
	earliestIt.EXPECT().Key().Return(getBundleHistoryHashKey(block))
	earliestIt.EXPECT().Value().Return(make([]byte, 32))
	earliestIt.EXPECT().Release()
	table.EXPECT().Get(getBundleHistoryHashKey(block)).Return(make([]byte, 32), nil)

	// deleting the bundles of later blocks fails
	table.EXPECT().NewBatch().Return(batch)
	table.EXPECT().NewIterator([]byte{'i'}, bigendian.Uint64ToBytes(block+1)).Return(it)
	it.EXPECT().Next().Return(false)
	it.EXPECT().Error().Return(injectedErr)
	it.EXPECT().Release()
	table.EXPECT().NewIterator([]byte{'h'}, bigendian.Uint64ToBytes(block+1)).Return(hashesIt)
	hashesIt.EXPECT().Next().Return(false)
	hashesIt.EXPECT().Error().Return(nil)
	hashesIt.EXPECT().Release()
	batch.EXPECT().Put(nil, gomock.Any()).Return(nil)
	// no batch.Write() is expected

	err := store.RollbackProcessedBundles(block)
	require.ErrorContains(t, err, "failed to roll back processed bundles")
	require.ErrorIs(t, err, injectedErr)
}

func TestStore_RollbackProcessedBundles_IsNoop_WithoutHistory(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)

	require.NoError(t, store.RollbackProcessedBundles(5))

	block, hash := store.GetLatestProcessedBundleHistoryHash()
	require.Zero(t, block)
	require.Equal(t, common.Hash{}, hash)
}

func TestStore_AddProcessedBundles_HonorsConfiguredRetention(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)
	const retention = bundle.MaxBlockRangeLength + 5
	store.cfg.ProcessedBundlesRetention = retention

	const head = 2 * retention
	for block := uint64(1); block <= head; block++ {
		store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{uint64ToHash(block): {}})
	}

	for block := uint64(1); block <= head; block++ {
		require.Equal(block > head-retention+1, store.HasBundleRecentlyBeenProcessed(uint64ToHash(block)), "block %d", block)
	}
	earliest, _, found := store.GetEarliestBundleHistoryHash()
	require.True(found)
	require.Equal(head+1-retention, earliest)
}

func TestStore_AddProcessedBundles_RetainsReplayProtectionWindow_EvenIfRetentionIsSmaller(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)
	store.cfg.ProcessedBundlesRetention = 1

	const head = 2 * bundle.MaxBlockRangeLength
	for block := uint64(1); block <= head; block++ {
		store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{uint64ToHash(block): {}})
	}

	for block := uint64(1); block <= head; block++ {
		require.Equal(block > head-bundle.MaxBlockRangeLength+1,
			store.HasBundleRecentlyBeenProcessed(uint64ToHash(block)), "block %d", block)
	}
}

func TestStore_RollbackProcessedBundles_ReportsError_WhenWritingFails(t *testing.T) {
	store, table, _, batch, it := storeTableLogMocks(t)
	ctrl := gomock.NewController(t)
	earliestIt := NewMockdbIterator(ctrl)
	hashesIt := NewMockdbIterator(ctrl)

	const block = uint64(5)
	injectedErr := errors.New("batch write error")
	expectHistoryHashOf(table, earliestIt, block)
	table.EXPECT().NewBatch().Return(batch)
	for prefix, iterator := range map[byte]*MockdbIterator{'i': it, 'h': hashesIt} {
		table.EXPECT().NewIterator([]byte{prefix}, bigendian.Uint64ToBytes(block+1)).Return(iterator)
		iterator.EXPECT().Next().Return(false)
		iterator.EXPECT().Error().Return(nil)
		iterator.EXPECT().Release()
	}
	batch.EXPECT().Put(nil, gomock.Any()).Return(nil)
	batch.EXPECT().Write().Return(injectedErr)

	err := store.RollbackProcessedBundles(block)
	require.ErrorIs(t, err, injectedErr)
}

func TestStore_RollbackProcessedBundles_Fails_WhenHistoryHashOfBlockIsNotRetained(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	store.AddProcessedBundles(10, map[common.Hash]bundle.PositionInBlock{{0x01}: {}})
	store.AddProcessedBundles(11, nil)

	err = store.RollbackProcessedBundles(5)
	require.ErrorContains(t, err, "no processed bundles history hash retained for block 5")

	// the store is left untouched
	block, _ := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(t, uint64(11), block)
}

// --- helper functions ---

// referenceComputeStateHash is a reference implementation of the hash
// computation for the processed bundles state. To be used by tests.
func referenceComputeStateHash(
	oldHash, addedHash common.Hash,
	blockNum uint64,
) common.Hash {

	var data []byte
	data = append(data, oldHash.Bytes()...)
	data = append(data, addedHash.Bytes()...)
	data = binary.BigEndian.AppendUint64(data, blockNum)
	return common.Hash(crypto.Keccak256(data))
}

// storeTableLogMocks initializes a store with mocked table as ProcessedBundles,
// and logger.
// Returns the mocks so expectations can be added on them.
func storeTableLogMocks(t *testing.T) (
	*Store,
	*MockstoreTable,
	*logger.MockLogger,
	*MockstoreBatch,
	*MockdbIterator,
) {
	ctrl := gomock.NewController(t)
	store := &Store{}
	store.cfg.ProcessedBundlesRetention = DefaultProcessedBundlesRetention
	table := NewMockstoreTable(ctrl)
	store.table.ProcessedBundles = table

	log := logger.NewMockLogger(ctrl)
	store.Log = log

	batch := NewMockstoreBatch(ctrl)
	it := NewMockdbIterator(ctrl)

	return store, table, log, batch, it
}

// expectHistoryHashOf sets up the given mock table to report the given block as
// the earliest one with a retained history hash.
func expectHistoryHashOf(table *MockstoreTable, it *MockdbIterator, block uint64) {
	table.EXPECT().NewIterator([]byte{'h'}, nil).Return(it)
	it.EXPECT().Next().Return(true)
	it.EXPECT().Key().Return(getBundleHistoryHashKey(block))
	it.EXPECT().Value().Return(make([]byte, 32))
	it.EXPECT().Release()
	table.EXPECT().Get(getBundleHistoryHashKey(block)).Return(make([]byte, 32), nil)
}

// expectCrit sets up the given mock logger to expect a Crit call with the given
// message and error, and to panic with message containing both when that call happens.
// In production, a Crit log call causes the logger to exit the process.
// To prevent the test from exiting, the mock logger is configured to panic instead.
func expectCrit(log *logger.MockLogger, msg string, args ...any) {
	log.EXPECT().Crit(msg, args).
		Do(func(msg string, ctx ...any) {
			panic(fmt.Sprintf("%v: %v", msg, ctx))
		})
}

// uint64ToHash returns unique hashes for input integers.
// It can be used in tests to streamline the creation if unique and deterministic
// hashes without having to hardcode them.
func uint64ToHash(i uint64) common.Hash {
	var b [32]byte
	binary.BigEndian.PutUint64(b[24:], i)
	return common.Hash(b)
}

// BlockHashTableValueMatcher is a gomock.Matcher that matches byte slices of length 40
// where the first 8 bytes encode a block number equal to the one specified in the matcher.
//
// Use it to set expectations when writing the block number
type BlockHashTableValueMatcher struct {
	blockNum uint64
}

func (m BlockHashTableValueMatcher) Matches(v any) bool {
	b, ok := v.([]byte)
	if !ok {
		return false
	}
	if len(b) != 8+32 {
		return false
	}
	encodedBlockNum := b[:8]
	gotBlockNum := binary.BigEndian.Uint64(encodedBlockNum)
	return gotBlockNum == m.blockNum
}

func (m BlockHashTableValueMatcher) String() string {
	return fmt.Sprintf("is a byte slice of length 40, with block number %d encoded in the first 8 bytes", m.blockNum)
}

type BundleExecutionInfoMatcher struct {
	expected bundle.ExecutionInfo
}

func (m BundleExecutionInfoMatcher) Matches(v any) bool {
	b, ok := v.([]byte)
	if !ok {
		return false
	}
	if len(b) != 8+4+4 {
		return false
	}
	blockNum := binary.BigEndian.Uint64(b[:8])
	offset := binary.BigEndian.Uint32(b[8:12])
	count := binary.BigEndian.Uint32(b[12:16])
	return blockNum == m.expected.BlockNumber &&
		offset == m.expected.Position.Offset &&
		count == m.expected.Position.Count
}

func (m BundleExecutionInfoMatcher) String() string {
	return fmt.Sprintf("is a byte slice encoding bundle.ExecutionInfo with block number %d, offset %d and count %d",
		m.expected.BlockNumber, m.expected.Position.Offset, m.expected.Position.Count)
}
