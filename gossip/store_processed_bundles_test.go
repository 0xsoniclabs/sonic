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
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/logger"
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
	for i, blockNum := range []uint64{0, 1, bundle.MaxBlockRange / 2, bundle.MaxBlockRange - 2, bundle.MaxBlockRange - 1} {
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
		store.addNewBundles(blockNum, infos, batch)
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
		bundle.MaxBlockRange - 1,
		bundle.MaxBlockRange,
		bundle.MaxBlockRange + 1,
	} {
		t.Run(fmt.Sprintf("BlockNumber=%d", block), func(t *testing.T) {
			store, table, _, _, _ := storeTableLogMocks(t)

			batch := NewMockstoreBatch(gomock.NewController(t))
			table.EXPECT().NewBatch().Return(batch)
			table.EXPECT().Get(gomock.Any())

			batch.EXPECT().Put(nil, BlockHashTableValueMatcher{blockNum: block})
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
			if block >= bundle.MaxBlockRange-1 {
				toDelete := block - bundle.MaxBlockRange + 1

				it := NewMockdbIterator(gomock.NewController(t))
				table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it)
				next := it.EXPECT().Next().Return(true)
				it.EXPECT().Next().Return(false).After(next).AnyTimes()
				it.EXPECT().Key().Return(getIndexKey(toDelete, uint64ToHash(toDelete)))
				table.EXPECT().Get(getEntryKey(uint64ToHash(toDelete))).Return(nil, nil).AnyTimes()

				hash := uint64ToHash(toDelete)
				batch.EXPECT().Delete(getEntryKey(hash)).Return(nil)
				batch.EXPECT().Delete(getIndexKey(toDelete, hash)).Return(nil)
			}

			store.AddProcessedBundles(block, map[common.Hash]bundle.PositionInBlock{
				info1.ExecutionPlanHash: {},
			})
		})
	}
}

func TestStore_AddProcessedBundles_LogsOnBatchPutNewEntryError(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)

	injectedErr := errors.New("new entry put error")
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErr)

	table.EXPECT().NewBatch().Return(batch)
	table.EXPECT().Get(gomock.Any()).Return(nil, nil)

	expectCrit(log, "failed to update hash of processed bundles", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to update hash of processed bundles: %v", []any{"error", injectedErr}),
		func() { store.AddProcessedBundles(1, nil) })
}

func TestStore_AddProcessedBundles_LogsOnBatchWriteError(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)

	injectedErr := errors.New("batch write error")
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
	batch.EXPECT().Write().Return(injectedErr)

	table.EXPECT().NewBatch().Return(batch)
	table.EXPECT().Get(gomock.Any()).Return(nil, nil)

	expectCrit(log, "failed to write batch for updating processed bundles", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to write batch for updating processed bundles: %v", []any{"error", injectedErr}),
		func() { store.AddProcessedBundles(1, nil) })
}

func TestStore_GetProcessedBundleHistoryHash_InitiallyZero(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum, hash := store.GetProcessedBundleHistoryHash()
	require.Zero(blockNum)
	require.Zero(hash)
}

func TestStore_GetProcessedBundleHistoryHash_CorrectlyParsesHash(t *testing.T) {
	store, table, _, _, _ := storeTableLogMocks(t)

	for i := range 2 * bundle.MaxBlockRange {
		block := uint64(i)
		hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("hash for block %d", block)))

		encoded := append(
			binary.BigEndian.AppendUint64(nil, block),
			hash.Bytes()...,
		)

		table.EXPECT().Get(nil).Return(encoded, nil)
		gotBlock, gotHash := store.GetProcessedBundleHistoryHash()

		require.Equal(t, block, gotBlock)
		require.Equal(t, hash, gotHash)
	}
}

func TestStore_GetProcessedBundleHistoryHash_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to get hash of processed bundles", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to get hash of processed bundles: %v", []any{"error", injectedErr}),
		func() { store.GetProcessedBundleHistoryHash() })
}

func TestStore_GetProcessedBundleHistoryHash_LogsOnInvalidStateLength(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	table.EXPECT().Get(gomock.Any()).Return([]byte{1, 2, 3}, nil)
	store.table.ProcessedBundles = table

	expectCrit(log, "invalid state length for processed bundles", "length", 3)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("invalid state length for processed bundles: %v", []any{"length", 3}),
		func() { store.GetProcessedBundleHistoryHash() })
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
		store.addNewBundles(uint64(blockNum), infoMap, batch)
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
			addedHash := store.addNewBundles(1, c.executedBundles, batch)

			expectedHash := common.Hash{}
			for hash := range c.executedBundles {
				expectedHash = xorHash(expectedHash, hash)
			}
			require.Equal(expectedHash, addedHash)
		})
	}
}

func TestStore_addNewBundles_LogsOnBatchPutError(t *testing.T) {
	store, _, log, batch, _ := storeTableLogMocks(t)

	injectedErrEntry := errors.New("entry put error")
	injectedErrIndex := errors.New("index put error")
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErrEntry)
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErrIndex)

	compoundErr := errors.Join(injectedErrEntry, injectedErrIndex)
	expectCrit(log, "failed to add processed bundle hash to batch", "error", compoundErr)

	hash1 := common.Hash{1, 2, 3}
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to add processed bundle hash to batch: %v", []any{"error", compoundErr}),
		func() {
			store.addNewBundles(1, map[common.Hash]bundle.PositionInBlock{
				hash1: {Offset: 4, Count: 5},
			}, batch)
		})
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
			finishingBlock:          bundle.MaxBlockRange - 2,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          bundle.MaxBlockRange - 2,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          bundle.MaxBlockRange - 1,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange / 2,
			finishingBlock:          bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange - 1,
			finishingBlock:          bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange,
			finishingBlock:          bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		// Following cases are after the warm up phase, when current block
		// number is large enough to have a history to delete,
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          bundle.MaxBlockRange - 1,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 0,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: 1,
			finishingBlock:          bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange / 2,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange - 1,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange + 1,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		// Following cases are recent enough to not be deleted
		{
			storedBundleBlockNumber: bundle.MaxBlockRange + 2,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange * 3 / 2,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2*bundle.MaxBlockRange - 1,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2 * bundle.MaxBlockRange,
			finishingBlock:          2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		// future block numbers should not cause deletion
		{
			storedBundleBlockNumber: 2*bundle.MaxBlockRange + 1,
			finishingBlock:          2 * bundle.MaxBlockRange,
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
			store := &Store{}
			store.table.ProcessedBundles = table

			existingBundleHash := common.Hash{1, 2, 3}

			// The algorithm would not contemplate any history
			// when the block number is short enough to not to require any cleanup
			if c.finishingBlock >= bundle.MaxBlockRange-1 {
				existingBundleKey := getIndexKey(c.storedBundleBlockNumber, existingBundleHash)
				gomock.InOrder(
					table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it),
					it.EXPECT().Next().Return(true),
					it.EXPECT().Key().Return(existingBundleKey),
					// AnyTimes: when entries are not to be deleted, the iteration is stopped
					it.EXPECT().Next().Return(false).AnyTimes(),
				)

				// this expectation is the core of the test:
				// it checks that the delete calls are made if and only if the
				// existing bundle is old enough to be deleted.
				if c.expectDeleted {
					batch.EXPECT().Delete(getIndexKey(c.storedBundleBlockNumber, existingBundleHash))
					batch.EXPECT().Delete(getEntryKey(existingBundleHash))
				}
			}

			hash := store.deleteOutdatedBundles(c.finishingBlock, batch)
			if c.expectDeleted {
				require.Equal(t, existingBundleHash, hash)
			}
		})
	}
}

func TestStore_deleteOutdatedBundles_RemovesMultipleEntries_WhenNotCleanedForTooLong(t *testing.T) {
	ctrl := gomock.NewController(t)
	batch := NewMockstoreBatch(ctrl)
	table := NewMockstoreTable(ctrl)

	store := &Store{}
	store.table.ProcessedBundles = table

	it := NewMockdbIterator(ctrl)
	table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it)

	for i := range 10 {
		it.EXPECT().Next().Return(true)
		it.EXPECT().Key().Return(getIndexKey(uint64(i), uint64ToHash(uint64(i))))
		batch.EXPECT().Delete(gomock.Any())
		batch.EXPECT().Delete(gomock.Any())
	}
	it.EXPECT().Next().Return(false)

	store.deleteOutdatedBundles(bundle.MaxBlockRange+10, batch)
}

func TestStore_deleteOutdatedBundles_ReturnsXorHashOfDeletedEntries(t *testing.T) {

	cases := map[string]struct {
		storedBundles map[common.Hash]bundle.PositionInBlock
	}{
		"empty list": {
			storedBundles: map[common.Hash]bundle.PositionInBlock{},
		},
		"single bundle": {
			storedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {},
			},
		},
		"two bundles": {
			storedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {},
				{4, 5, 6}: {},
			},
		},
		"more than two bundles": {
			storedBundles: map[common.Hash]bundle.PositionInBlock{
				{1, 2, 3}: {},
				{4, 5, 6}: {},
				{7, 8, 9}: {},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			store, table, _, batch, it := storeTableLogMocks(t)
			existingBundleKeys := make(map[common.Hash][]byte, len(c.storedBundles))
			for hash := range c.storedBundles {
				existingBundleKeys[hash] = getIndexKey(1, hash)
			}

			table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it)

			for hash, key := range existingBundleKeys {
				gomock.InOrder(
					it.EXPECT().Next().Return(true),
					it.EXPECT().Key().Return(key),
					batch.EXPECT().Delete(getIndexKey(1, hash)).Return(nil),
					batch.EXPECT().Delete(getEntryKey(hash)).Return(nil),
				)
			}

			it.EXPECT().Next().Return(false)

			deletedHash := store.deleteOutdatedBundles(bundle.MaxBlockRange+1, batch)

			expectedDeletedHash := common.Hash{}
			for hash := range c.storedBundles {
				expectedDeletedHash = xorHash(expectedDeletedHash, hash)
			}
			require.Equal(t, expectedDeletedHash, deletedHash)
		})
	}
}

func TestStore_deleteOutdatedBundles_IgnoresKeysOfWrongLength(t *testing.T) {
	// log mock is ignored because no log called should be triggered.
	store, table, _, batch, it := storeTableLogMocks(t)

	gomock.InOrder(
		it.EXPECT().Next().Return(true),
		// This is the key that will be ignored, since it does not have the correct length.
		it.EXPECT().Key().Return([]byte{1, 2}),
		it.EXPECT().Next().Return(false),
	)
	table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it)

	require.Zero(t, store.deleteOutdatedBundles(bundle.MaxBlockRange+1, batch))
}

func TestStore_deleteOutdatedBundles_LogsOnBatchDeleteError(t *testing.T) {
	store, table, log, batch, it := storeTableLogMocks(t)

	injectedErrDeleteEntry := errors.New("entry delete error")
	injectedErrDeleteIndex := errors.New("index delete error")
	batch.EXPECT().Delete(gomock.Any()).Return(injectedErrDeleteEntry)
	batch.EXPECT().Delete(gomock.Any()).Return(injectedErrDeleteIndex)

	gomock.InOrder(
		it.EXPECT().Next().Return(true),
		it.EXPECT().Key().Return(getIndexKey(1, common.Hash{1, 2, 3})),
	)
	table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(it)

	compoundErr := errors.Join(injectedErrDeleteEntry, injectedErrDeleteIndex)
	expectCrit(log, "failed to delete old processed bundle hash", "error", compoundErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("failed to delete old processed bundle hash: %v", []any{"error", compoundErr}),
		func() { store.deleteOutdatedBundles(bundle.MaxBlockRange+1, batch) })
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
	//  - oldHash, addedHash, and deletedHash having specific patterns (e.g., all zeros, all 0xff, etc.)
	//  - combinations of the above

	hashDomain := []common.Hash{
		{},
		{4, 5, 6},
		{0xff, 0xff, 0xff},
		common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	}
	blockNumberDomain := []uint64{
		0, 1, 512,
		bundle.MaxBlockRange - 1,
		bundle.MaxBlockRange,
		bundle.MaxBlockRange + 1,
		math.MaxUint64}

	type testCase struct {
		oldHash     common.Hash
		addedHash   common.Hash
		deletedHash common.Hash
		blockNum    uint64
	}
	testCases := map[string]testCase{}
	for _, oldHash := range hashDomain {
		for _, addedHash := range hashDomain {
			for _, deletedHash := range hashDomain {
				for _, blockNum := range blockNumberDomain {
					name := fmt.Sprintf("oldHash=%s/addedHash=%s/deletedHash=%s/blockNum=%d",
						oldHash.Hex(), addedHash.Hex(), deletedHash.Hex(), blockNum)
					testCases[name] = testCase{
						oldHash:     oldHash,
						addedHash:   addedHash,
						deletedHash: deletedHash,
						blockNum:    blockNum,
					}
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := computeNewBundleStateHash(tc.oldHash, tc.addedHash, tc.deletedHash, tc.blockNum)
			ref := referenceComputeStateHash(tc.blockNum, tc.oldHash, tc.addedHash, tc.deletedHash)
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

	iter = store.table.ProcessedBundles.NewIterator([]byte{'i'}, nil)
	require.False(iter.Next())
	require.NoError(iter.Error())

	iter = store.table.ProcessedBundles.NewIterator([]byte{'e'}, nil)
	require.False(iter.Next())
	require.NoError(iter.Error())
}

func TestStore_ProcessedBundles_UpdatesHistoryHash(t *testing.T) {
	require := require.New(t)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	cases := map[string]struct {
		bundles map[common.Hash]bundle.PositionInBlock
	}{
		"empty block": {
			bundles: map[common.Hash]bundle.PositionInBlock{},
		},
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
			_, initialHash := store.GetProcessedBundleHistoryHash()

			store.AddProcessedBundles(1, tc.bundles)
			addedHash := common.Hash{}
			for hash := range tc.bundles {
				addedHash = xorHash(addedHash, hash)
			}
			expectedHash := referenceComputeStateHash(1, initialHash, addedHash, common.Hash{})
			_, gotHash := store.GetProcessedBundleHistoryHash()
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

	_, hashA := store1.GetProcessedBundleHistoryHash()
	_, hashB := store2.GetProcessedBundleHistoryHash()
	require.Equal(hashA, hashB)
}

func TestStore_ProcessedBundles_HashIsUpdatedWithNewBlocks(t *testing.T) {
	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	// this test relies on the incremental nature uint64ToHash to
	// generate distinct hashes which also yield different xor values
	seenHashes := make(map[common.Hash]struct{})
	for i := range 4 * bundle.MaxBlockRange {
		store.AddProcessedBundles(i, map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(uint64(i)): {Offset: 0, Count: 1},
		})

		block, got := store.GetProcessedBundleHistoryHash()
		require.Equal(uint64(i), block)
		require.NotContains(seenHashes, got)
		seenHashes[got] = struct{}{}
	}
}

func TestStore_ProcessedBundles_RetainsAllBundlesRequiredToCoverTheMaximumBlockRange(t *testing.T) {
	require := require.New(t)
	numBlocks := 3 * bundle.MaxBlockRange

	store, err := NewMemStore(t)
	require.NoError(err)

	// While progressing through the blocks, all execution plans must be retained
	// until their maximum block range has expired.
	for currentBlockNumber := range numBlocks {

		// Check that the store covers exactly the plans of the past that are
		// allowed to be included in the current block (before adding it).
		for block := uint64(0); block < currentBlockNumber; block++ {
			blockRange := bundle.MakeMaxRangeStartingAt(block)
			want := blockRange.IsInRange(currentBlockNumber)
			require.Equal(
				want, store.HasBundleRecentlyBeenProcessed(uint64ToHash(block)),
				"Current block %d, checking plan with range [%d,%d]",
				currentBlockNumber, blockRange.Earliest, blockRange.Latest,
			)
		}

		store.AddProcessedBundles(currentBlockNumber, map[common.Hash]bundle.PositionInBlock{
			uint64ToHash(currentBlockNumber): {},
		})
	}
}

func TestStore_SetRawProcessedBundle_ReturnsErrorForInvalidHashLength(t *testing.T) {
	require := require.New(t)

	tests := map[string]struct {
		key, value []byte
		errorMsg   string
	}{
		"history hash invalid value": {
			key:      nil,
			value:    []byte{0, 1, 2}, // should be 40 bytes for blockNum + hash
			errorMsg: "invalid value length for bundle history hash",
		},
		"entry invalid key length": {
			key:      []byte{0, 1, 2},  // should be 33 bytes for 'e' + hash
			value:    make([]byte, 16), // correct length for value
			errorMsg: "invalid key or value for processed bundle entry",
		},
		"entry invalid value length": {
			key:      append([]byte{'e'}, make([]byte, 32)...), // valid length key
			value:    []byte{0, 1, 2},                          // short value
			errorMsg: "invalid key or value for processed bundle entry",
		},
		"empty execution plan": {
			key:      append([]byte{'e'}, make([]byte, 32)...), // valid length key
			value:    make([]byte, 16),                         // valid length but empty value
			errorMsg: "invalid execution plan hash",            // we require the hash to be non-zero to avoid confusion with empty entries, but the error message is the same as for invalid lengths since we check length first
		},
		"entry does not start with 'e'": {
			key:      append([]byte{'x'}, make([]byte, 32)...), // valid length but wrong prefix
			value:    make([]byte, 16),                         // correct length for value
			errorMsg: "invalid key prefix for processed bundle entry: expected 'e'",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(err)

			err = store.SetRawProcessedBundle(BundleKV{Key: tc.key, Value: tc.value})
			require.Error(err)
			require.Contains(err.Error(), tc.errorMsg)
		})
	}
}

func TestStore_SetRawProcessedBundle_RecognizesBundleHistoryHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum := uint64(10)
	hash := common.Hash{1, 2, 3}
	// build the value for the history hash entry
	value := make([]byte, 8+32)
	binary.BigEndian.PutUint64(value[:8], blockNum)
	copy(value[8:], hash[:])

	// nil key for history hash
	err = store.SetRawProcessedBundle(BundleKV{Key: nil, Value: value})
	require.NoError(err)

	resBlockNum, resHash := store.GetProcessedBundleHistoryHash()
	require.Equal(blockNum, resBlockNum)
	require.Equal(hash, resHash)
}

func TestStore_SetRawProcessedBundle_AddsEntryToStore(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	// build execution info
	hash := common.Hash{1, 2, 3}
	blockNum := uint64(10)
	info := bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNumber:       blockNum,
		Position:          bundle.PositionInBlock{Offset: 0, Count: 1},
	}

	// encode entry value.
	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[:8], info.BlockNumber)
	binary.BigEndian.PutUint32(data[8:12], info.Position.Offset)
	binary.BigEndian.PutUint32(data[12:], info.Position.Count)

	entry := BundleKV{
		Key:   append([]byte{'e'}, hash.Bytes()...),
		Value: data,
	}

	err = store.SetRawProcessedBundle(entry)
	require.NoError(err)

	resInfo := store.GetBundleExecutionInfo(hash)
	require.NotNil(resInfo)
	require.Equal(info.ExecutionPlanHash, resInfo.ExecutionPlanHash)
	require.Equal(info.BlockNumber, resInfo.BlockNumber)
	require.Equal(info.Position, resInfo.Position)
}

func TestStore_SetRawProcessedBundle_AddsIndexEntry(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash := common.Hash{1, 2, 3}
	blockNum := uint64(10)
	info := bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNumber:       blockNum,
		Position:          bundle.PositionInBlock{Offset: 0, Count: 1},
	}

	// encode entry value.
	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[:8], info.BlockNumber)
	binary.BigEndian.PutUint32(data[8:12], info.Position.Offset)
	binary.BigEndian.PutUint32(data[12:], info.Position.Count)

	entry := BundleKV{
		Key:   append([]byte{'e'}, hash.Bytes()...),
		Value: data,
	}
	err = store.SetRawProcessedBundle(entry)
	require.NoError(err)

	// check that the index entry was added (the value doesn't matter, just that it exists)
	indexKey := getIndexKey(blockNum, hash)
	hasIndexEntry, err := store.table.ProcessedBundles.Has(indexKey)
	require.NoError(err)
	require.True(hasIndexEntry, "expected index entry for processed bundle was not found")
}

func TestStore_DumpProcessedBundles_ReturnsEmptySliceWhenNoEntries(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	dumpedEntries := store.DumpProcessedBundles()
	require.NotNil(dumpedEntries)
	require.Empty(dumpedEntries, "expected no dumped entries when store is empty")
}

func TestStore_DumpProcessedBundles_ReturnsAllAddedEntries(t *testing.T) {

	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	// fill the store with the maximum number of bundles
	for i := range bundle.MaxBlockRange + 1 {
		hash := uint32ToBytes(uint32(i))
		executedBundles := map[common.Hash]bundle.PositionInBlock{
			common.BytesToHash(hash): {Offset: 0, Count: 1},
		}
		store.AddProcessedBundles(uint64(i), executedBundles)
	}
	block, historyHash := store.GetProcessedBundleHistoryHash()
	require.Equal(uint64(bundle.MaxBlockRange), block)
	require.NotNil(historyHash)
	require.NotZero(historyHash)

	// blockNum (8 bytes) + hash (32 bytes)
	bundleHistoryHashSize := 8 + 32
	// key size: 'e' prefix + hash
	// value size: blockNum (8 bytes) + position (4 bytes) + count (4 bytes)
	entrySize := 1 + 32 + 16

	expectedSize :=
		bundleHistoryHashSize +
			int(bundle.MaxBlockRange-1)*entrySize +
			8*int(bundle.MaxBlockRange) // key/value size per entry.

	dumpedEntries := store.DumpProcessedBundles()
	// 1 history hash + MaxBlockRange-1 entries
	require.Len(dumpedEntries, int(bundle.MaxBlockRange),
		fmt.Sprintf("expected %d dumped entries, got %d",
			bundle.MaxBlockRange, len(dumpedEntries)))

	actualSize := 0
	for _, entry := range dumpedEntries {
		actualSize += len(entry)
	}

	require.Equal(expectedSize, actualSize,
		fmt.Sprintf("expected %d dumped entries, got %d",
			expectedSize, actualSize))
}

func TestStore_DumpProcessedBundles_ReturnsEncodedEntry(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	// make an execution info
	hash := common.Hash{1, 2, 3}
	blockNum := uint64(10)
	info := bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNumber:       blockNum,
		Position:          bundle.PositionInBlock{Offset: 0, Count: 1},
	}
	executedBundles := map[common.Hash]bundle.PositionInBlock{
		hash: {Offset: info.Position.Offset, Count: info.Position.Count},
	}

	// add execution info to store.
	store.AddProcessedBundles(blockNum, executedBundles)

	// get the dumped entries
	dumpedEntries := store.DumpProcessedBundles()
	require.Len(dumpedEntries, 2) // history hash + 1 entry

	// check that the dumped entry matches the expected encoding of the added entry
	expectedEntry := BundleKV{
		Key:   append([]byte{'e'}, hash.Bytes()...),
		Value: make([]byte, 16),
	}
	binary.BigEndian.PutUint64(expectedEntry.Value[:8], info.BlockNumber)
	binary.BigEndian.PutUint32(expectedEntry.Value[8:12], info.Position.Offset)
	binary.BigEndian.PutUint32(expectedEntry.Value[12:], info.Position.Count)

	require.Contains(dumpedEntries, expectedEntry.Encode())
}

func TestStore_DumpProcessedBundles_LogsOnCrit(t *testing.T) {
	store, table, log, _, it := storeTableLogMocks(t)

	injectedErr := errors.New("iterator error")
	gomock.InOrder(
		table.EXPECT().Get(nil).Return(make([]byte, 40), nil),
		table.EXPECT().NewIterator([]byte{'e'}, nil).Return(it),
		it.EXPECT().Next().Return(false),
		it.EXPECT().Error().Return(injectedErr).AnyTimes(),
	)
	expectCrit(log, "failed to dump processed bundles", "error", injectedErr)

	require.PanicsWithValue(t,
		fmt.Sprintf("failed to dump processed bundles: %v", []any{"error", injectedErr}),
		func() { store.DumpProcessedBundles() })
}

func TestStore_BundleKV_Encode_FollowsExpectedFormat(t *testing.T) {
	tests := map[string]struct {
		key   []byte
		value []byte
	}{
		"entry key": {
			key:   append([]byte{'e'}, common.Hash{1, 2, 3}.Bytes()...),
			value: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		"nil key (history hash)": {
			key:   []byte{},
			value: []byte{1: 40},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			entry := BundleKV{Key: tc.key, Value: tc.value}
			encoded := entry.Encode()

			// expected total: 4 (key len) + key + 4 (value len) + value
			expectedLen := 4 + len(tc.key) + 4 + len(tc.value)
			require.Len(t, encoded, expectedLen)

			// verify lengths are big-endian encoded at the right offsets
			gotKeyLen := binary.BigEndian.Uint32(encoded[0:4])
			require.Equal(t, uint32(len(tc.key)), gotKeyLen)

			gotValLen := binary.BigEndian.Uint32(encoded[4+len(tc.key) : 4+len(tc.key)+4])
			require.Equal(t, uint32(len(tc.value)), gotValLen)

			// verify key and value payloads
			require.Equal(t, tc.key, encoded[4:4+len(tc.key)])
			require.Equal(t, tc.value, encoded[4+len(tc.key)+4:])
		})
	}
}

// --- helper functions ---

// referenceComputeStateHash is a reference implementation of the hash
// computation for the processed bundles state. To be used by tests.
func referenceComputeStateHash(
	blockNum uint64,
	oldHash, addedHash, deletedHash common.Hash,
) common.Hash {

	var data []byte
	data = append(data, oldHash.Bytes()...)
	data = append(data, addedHash.Bytes()...)
	data = append(data, deletedHash.Bytes()...)
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
	table := NewMockstoreTable(ctrl)
	store.table.ProcessedBundles = table

	log := logger.NewMockLogger(ctrl)
	store.Log = log

	batch := NewMockstoreBatch(ctrl)
	it := NewMockdbIterator(ctrl)

	return store, table, log, batch, it
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
