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
	"maps"
	"math"
	"slices"
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

	history := map[uint64]map[string]bundle.ExecutionInfo{}

	// construct a history of bundles in boundary block number and boundary positions
	for i, blockNum := range []uint64{0, 1, bundle.MaxBlockRange / 2, bundle.MaxBlockRange - 2, bundle.MaxBlockRange - 1} {
		history[blockNum] = map[string]bundle.ExecutionInfo{
			"first": {
				ExecutionPlanHash: uint64ToHash(uint64(i * 3)),
			},
			"midle": {
				ExecutionPlanHash: uint64ToHash(uint64(i*3 + 1)),
			},
			"last": {
				ExecutionPlanHash: uint64ToHash(uint64(i*3 + 2)),
			},
		}
	}

	// initialize storage with the provided history
	table := store.table.ProcessedBundles
	batch := table.NewBatch()
	for blockNum, bundles := range history {
		store.addNewBundles(blockNum, slices.Collect(maps.Values(bundles)), batch)
	}
	require.NoError(batch.Write())

	for blockNum, test := range history {
		for name, test := range test {
			// check every element in the history can be retrieved correctly by the store
			t.Run(fmt.Sprintf("BlockNumber=%d/%s", blockNum, name), func(t *testing.T) {
				info := store.GetBundleExecutionInfo(test.ExecutionPlanHash)
				require.Equal(test, *info)
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
		func() { store.AddProcessedBundles(1, []bundle.ExecutionInfo{}) })
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
		func() { store.AddProcessedBundles(1, []bundle.ExecutionInfo{}) })
}

func TestStore_GetProcessedBundleHistoryHash_InitiallyZero(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum, hash := store.GetProcessedBundleHistoryHash()
	require.Zero(blockNum)
	require.Zero(hash)
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

func TestStore_addNewBundles_ReturnsExpectedHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	cases := map[string]struct {
		executedBundles []bundle.ExecutionInfo
	}{
		"empty list": {
			executedBundles: []bundle.ExecutionInfo{},
		},
		"single entry": {
			executedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
			},
		},
		"two entries": {
			executedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
				wrapInfo(common.Hash{4, 5, 6}),
			},
		},
		"more than two entries": {
			executedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
				wrapInfo(common.Hash{4, 5, 6}),
				wrapInfo(common.Hash{7, 8, 9}),
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
			for _, info := range c.executedBundles {
				expectedHash = xorHash(expectedHash, info.ExecutionPlanHash)
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
			store.addNewBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)}, batch)
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
				encodedBlock := make([]byte, 8)
				binary.BigEndian.PutUint64(encodedBlock, c.storedBundleBlockNumber)
				existingBundleKey := append(append([]byte{'i'}, encodedBlock...), existingBundleHash.Bytes()...)
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

func TestStore_deleteOutdatedBundles_ReturnsXorHashOfDeletedEntries(t *testing.T) {

	cases := map[string]struct {
		storedBundles []bundle.ExecutionInfo
	}{
		"empty list": {
			storedBundles: []bundle.ExecutionInfo{},
		},
		"single bundle": {
			storedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
			},
		},
		"two bundles": {
			storedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
				wrapInfo(common.Hash{4, 5, 6}),
			},
		},
		"more than two bundles": {
			storedBundles: []bundle.ExecutionInfo{
				wrapInfo(common.Hash{1, 2, 3}),
				wrapInfo(common.Hash{4, 5, 6}),
				wrapInfo(common.Hash{7, 8, 9}),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			store, table, _, batch, it := storeTableLogMocks(t)
			encodedBlock := make([]byte, 8)
			binary.BigEndian.PutUint64(encodedBlock, 1)
			existingBundleKeys := make([][]byte, len(c.storedBundles))
			for i, info := range c.storedBundles {
				existingBundleKeys[i] = append(append([]byte{'i'}, encodedBlock...), info.ExecutionPlanHash.Bytes()...)
			}

			table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it)

			for i, key := range existingBundleKeys {
				gomock.InOrder(
					it.EXPECT().Next().Return(true),
					it.EXPECT().Key().Return(key),
					batch.EXPECT().Delete(getIndexKey(1, c.storedBundles[i].ExecutionPlanHash)).Return(nil),
					batch.EXPECT().Delete(getEntryKey(c.storedBundles[i].ExecutionPlanHash)).Return(nil),
				)
			}

			it.EXPECT().Next().Return(false)

			deletedHash := store.deleteOutdatedBundles(bundle.MaxBlockRange+1, batch)

			expectedDeletedHash := common.Hash{}
			for _, info := range c.storedBundles {
				expectedDeletedHash = xorHash(expectedDeletedHash, info.ExecutionPlanHash)
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
		bundles []bundle.ExecutionInfo
	}{
		"empty block": {
			bundles: []bundle.ExecutionInfo{},
		},
		"single new bundle": {
			bundles: []bundle.ExecutionInfo{wrapInfo(hash1)},
		},
		"multiple new bundles": {
			bundles: []bundle.ExecutionInfo{wrapInfo(hash2), wrapInfo(hash3)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(err)
			_, initialHash := store.GetProcessedBundleHistoryHash()

			store.AddProcessedBundles(1, tc.bundles)
			addedHash := common.Hash{}
			for _, info := range tc.bundles {
				addedHash = xorHash(addedHash, info.ExecutionPlanHash)
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

	store1.AddProcessedBundles(1, []bundle.ExecutionInfo{
		wrapInfo(hash1),
		wrapInfo(hash2),
	})

	store2.AddProcessedBundles(1, []bundle.ExecutionInfo{
		wrapInfo(hash2),
		wrapInfo(hash1),
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
		store.AddProcessedBundles(i, []bundle.ExecutionInfo{
			{
				ExecutionPlanHash: uint64ToHash(uint64(i)),
				BlockNum:          i,
			},
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

		store.AddProcessedBundles(currentBlockNumber, []bundle.ExecutionInfo{
			{
				ExecutionPlanHash: uint64ToHash(currentBlockNumber),
				BlockNum:          currentBlockNumber,
			},
		})
	}
}

// --- helper functions ---

// return execution info with the given hash and position 0.
// If a second parameter is given, it is used as the block number,
// otherwise the block number is set to 1. If more than two parameters are given
// the extras are ignored.
func wrapInfo(hash common.Hash, blockNum ...uint64) bundle.ExecutionInfo {
	if len(blockNum) == 0 {
		blockNum = []uint64{1}
	}
	return bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNum:          blockNum[0],
		Position:          0,
		Count:             1,
	}
}

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
