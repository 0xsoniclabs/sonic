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
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=store_processed_bundles_test.go -destination=store_processed_bundles_test_mock.go -package=gossip

// storeTable is an interface needed to generate a mock for a kvdb.Store.
type storeTable interface {
	kvdb.Store
}

var _ storeTable // to avoid storeTable unused warning.

// storeBatch is an interface needed to generate a mock for a kvdb.Batch.
type storeBatch interface {
	kvdb.Batch
}

var _ storeBatch // to avoid storeBatch unused warning.

// dbIterator is an interface needed to generate a mock for a ethdb.Iterator.
type dbIterator interface {
	ethdb.Iterator
}

var _ dbIterator // to avoid dbIterator unused warning.

func TestStore_HasBundleRecentlyBeenProcessed_TracksAddedBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	isRecentlyProcessed := func(hash common.Hash) bool {
		return store.HasBundleRecentlyBeenProcessed(hash)
	}
	// initially there are no recently processed bundles
	require.False(isRecentlyProcessed(hash1))
	require.False(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})
	require.True(isRecentlyProcessed(hash1))
	require.False(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))

	store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2), wrapInfo(hash3)})
	require.True(isRecentlyProcessed(hash1))
	require.True(isRecentlyProcessed(hash2))
	require.True(isRecentlyProcessed(hash3))
}

func TestStore_HasBundleRecentlyBeenProcessed_CleansUpOldBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}
	hash4 := common.Hash{10, 11, 12}
	hash5 := common.Hash{13, 14, 15}

	isRecentlyProcessed := func(hash common.Hash) bool {
		return store.HasBundleRecentlyBeenProcessed(hash)
	}

	// initially there are no recently processed bundles
	require.False(isRecentlyProcessed(hash1))
	require.False(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))
	require.False(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))

	// add hash1 in block 1.
	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})

	require.True(isRecentlyProcessed(hash1))
	require.False(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))
	require.False(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))

	// add hash2 in block 2.
	store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2)})

	require.True(isRecentlyProcessed(hash1))
	require.True(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))
	require.False(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))

	// add hash3 in block 1 + bundle.MaxBlockRange/2,
	// so hash1 is still recent but will be cleaned up in the next step.
	store.AddProcessedBundles(1+bundle.MaxBlockRange/2,
		[]bundle.ExecutionInfo{wrapInfo(hash3)})

	require.True(isRecentlyProcessed(hash1))
	require.True(isRecentlyProcessed(hash2))
	require.True(isRecentlyProcessed(hash3))
	require.False(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))

	// add hash4 in block bundle.MaxBlockRange,
	// just before hash1 is considered too old
	store.AddProcessedBundles(bundle.MaxBlockRange,
		[]bundle.ExecutionInfo{wrapInfo(hash4)})

	require.True(isRecentlyProcessed(hash1))
	require.True(isRecentlyProcessed(hash2))
	require.True(isRecentlyProcessed(hash3))
	require.True(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))

	// add hash5 in block 1 + bundle.MaxBlockRange + 1,
	// so hash1 is now too old and should be cleaned up,
	store.AddProcessedBundles(1+bundle.MaxBlockRange,
		[]bundle.ExecutionInfo{wrapInfo(hash5)})

	require.False(isRecentlyProcessed(hash1))
	require.True(isRecentlyProcessed(hash2))
	require.True(isRecentlyProcessed(hash3))
	require.True(isRecentlyProcessed(hash4))
	require.True(isRecentlyProcessed(hash5))

	// add an no execution plan in block 1 + 2*bundle.MaxBlockRange,
	// which should clean up all remaining recent bundles
	store.AddProcessedBundles(1+2*bundle.MaxBlockRange,
		[]bundle.ExecutionInfo{})

	require.False(isRecentlyProcessed(hash1))
	require.False(isRecentlyProcessed(hash2))
	require.False(isRecentlyProcessed(hash3))
	require.False(isRecentlyProcessed(hash4))
	require.False(isRecentlyProcessed(hash5))
}

func TestStore_HasBundleRecentlyBeenProcessed_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to check processed bundle", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("%v: %v", "failed to check processed bundle", injectedErr),
		func() { store.HasBundleRecentlyBeenProcessed(common.Hash{1, 2, 3}) })
}

func TestStore_GetBundleExecutionInfo_LogsOnGetError(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(gomock.Any()).Return(nil, injectedErr)

	expectCrit(log, "failed to get execution info for bundle", "error", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("%v: %v", "failed to get execution info for bundle", injectedErr),
		func() { store.GetBundleExecutionInfo(common.Hash{1, 2, 3}) })
}

func TestStore_GetBundleExecutionInfo_LogsOnInvalidDataLength(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)

	table.EXPECT().Get(gomock.Any()).Return([]byte{1, 2, 3}, nil)

	expectCrit(log, "invalid data length for execution info", "length", 3)

	require.PanicsWithValue(t,
		fmt.Sprintf("%v: %v", "invalid data length for execution info", 3),
		func() { store.GetBundleExecutionInfo(common.Hash{1, 2, 3}) })
}

func TestStore_GetBundleExecutionInfo_ReturnsInfoForAddedBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}

	info1 := bundle.ExecutionInfo{
		ExecutionPlanHash: hash1,
		BlockNum:          1,
		Position:          0,
		Count:             1,
	}
	info2 := bundle.ExecutionInfo{
		ExecutionPlanHash: hash2,
		BlockNum:          2,
		Position:          1,
		Count:             2,
	}

	// initially there is no info for unknown hashes
	info := store.GetBundleExecutionInfo(hash1)
	require.Nil(info)
	info = store.GetBundleExecutionInfo(hash2)
	require.Nil(info)

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{info1, info2})

	resInfo1 := store.GetBundleExecutionInfo(hash1)
	require.Equal(info1, *resInfo1)

	resInfo2 := store.GetBundleExecutionInfo(hash2)
	require.Equal(info2, *resInfo2)
}

func TestStore_ProcessedBundles_TableIsInitiallyEmpty(t *testing.T) {
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
	store, err := NewMemStore(t)
	require.NoError(err)

	_, initialHash := store.GetProcessedBundleHistoryHash()

	hash1 := common.Hash{1, 2, 3}
	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})

	historyHash1 := updateHistoryHash(1, initialHash, hash1, common.Hash{})

	_, hashAfterFirstAdd := store.GetProcessedBundleHistoryHash()
	require.NotEqual(initialHash, hashAfterFirstAdd)
	require.Equal(historyHash1, hashAfterFirstAdd)

	hash2 := common.Hash{4, 5, 6}
	store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2)})

	historyHash2 := updateHistoryHash(2, historyHash1, hash2, common.Hash{})

	_, hashAfterSecondAdd := store.GetProcessedBundleHistoryHash()
	require.NotEqual(hashAfterFirstAdd, hashAfterSecondAdd)
	require.Equal(historyHash2, hashAfterSecondAdd)
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

	require.NotEqual(hashA, hashB)
}

func TestStore_ProcessedBundles_OldHashAffectsNewHash(t *testing.T) {
	require := require.New(t)
	store1, err := NewMemStore(t)
	require.NoError(err)
	store2, err := NewMemStore(t)
	require.NoError(err)

	_, hashA0 := store1.GetProcessedBundleHistoryHash()
	_, hashB0 := store2.GetProcessedBundleHistoryHash()
	require.Equal(hashA0, hashB0)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	store1.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})
	store2.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash2)})

	_, hashA := store1.GetProcessedBundleHistoryHash()
	_, hashB := store2.GetProcessedBundleHistoryHash()
	require.NotEqual(hashA, hashB)

	store1.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash3)})
	store2.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash3)})

	_, hashA2 := store1.GetProcessedBundleHistoryHash()
	_, hashB2 := store2.GetProcessedBundleHistoryHash()
	require.NotEqual(hashA2, hashB2)
}

func TestStore_ProcessedBundles_StoredHashUsesXorForAddedAndDeletedHashes(t *testing.T) {
	// this test checks that the hash stored in the table for processed bundles
	// is consistent with the expected hash computed. From the documentation:
	//
	// The hash of the processed bundle's history is computed as follows:
	//  - initially, the hash is zero
	//  - for every update, the hash is updated as follows:
	//      addedExecPlanHash = Xor(<hashes of newly added execution plans>)
	//      deletedExecPlanHash = Xor(<hashes of deleted execution plans>)
	//      newHash = Keccak256(oldHash || addedExecPlanHash || deletedExecPlanHash || blockNum)

	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	// get initial hash
	_, initialHash := store.GetProcessedBundleHistoryHash()

	// add two hashes
	store.AddProcessedBundles(1, []bundle.ExecutionInfo{
		wrapInfo(hash1),
		wrapInfo(hash2),
	})

	// get hash after adding two hashes
	blockNum, bundleHistoryHash := store.GetProcessedBundleHistoryHash()
	require.Equal(uint64(1), blockNum)

	addedHash := xorHash(hash1, hash2)
	newHash := updateHistoryHash(blockNum, initialHash, addedHash, common.Hash{})

	require.Equal(newHash, bundleHistoryHash)

	newerBlockNum := blockNum + bundle.MaxBlockRange
	// delete one and verify deletion hash as well
	store.AddProcessedBundles(newerBlockNum, []bundle.ExecutionInfo{
		wrapInfo(hash3),
	})

	addedHahs := xorHash(common.Hash{}, hash3)
	deletedHash := xorHash(hash1, hash2)
	newHash = updateHistoryHash(newerBlockNum, bundleHistoryHash, addedHahs, deletedHash)
	newerEncodedBlockNumber, updatedBundleHistoryHash := store.GetProcessedBundleHistoryHash()
	require.Equal(newerBlockNum, newerEncodedBlockNumber)
	require.Equal(newHash, updatedBundleHistoryHash)
}

func TestStore_GetEntryKey_ReturnsExpectedKey(t *testing.T) {
	require := require.New(t)

	hash := common.Hash{1, 2, 3}
	expectedKey := append([]byte{'e'}, hash.Bytes()...)
	require.Equal(expectedKey, getEntryKey(hash))
}

func TestStore_GetIndexKey_ReturnsExpectedKey(t *testing.T) {
	require := require.New(t)

	blockNum := uint64(123)
	hash := common.Hash{1, 2, 3}
	expectedKey := append([]byte{'i'}, make([]byte, 8)...)
	binary.BigEndian.PutUint64(expectedKey[1:9], blockNum)
	expectedKey = append(expectedKey, hash.Bytes()...)
	require.Equal(expectedKey, getIndexKey(blockNum, hash))
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
		fmt.Sprintf("%v: %v", "failed to update hash of processed bundles", injectedErr),
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
		fmt.Sprintf("%v: %v", "failed to write batch for updating processed bundles", injectedErr),
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
		fmt.Sprintf("%v: %v", "failed to get hash of processed bundles", injectedErr),
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
		fmt.Sprintf("%v: %v", "invalid state length for processed bundles", 3),
		func() { store.GetProcessedBundleHistoryHash() })
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
		"all zeros":           {common.Hash{0, 0, 0}, common.Hash{0, 0, 0}, common.Hash{0, 0, 0}},
		"zero and non-zero":   {common.Hash{0, 0, 0}, common.Hash{7, 8, 9}, common.Hash{7, 8, 9}},
		"non-zero and zero":   {common.Hash{10, 11, 12}, common.Hash{0, 0, 0}, common.Hash{10, 11, 12}},
		"same non-zero":       {common.Hash{1, 1, 1}, common.Hash{1, 1, 1}, common.Hash{0, 0, 0}},
		"operation with 0xff": {common.Hash{0xff, 0xff, 0xff}, common.Hash{0x1, 0x2, 0x3}, common.Hash{0xfe, 0xfd, 0xfc}},
		"32 bytes computed": {
			common.Hash{0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			common.Hash{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			common.Hash{1, 0, 3, 2, 5, 4, 6, 9, 8, 11, 10, 13, 12, 15, 14, 17, 16, 19, 18, 21, 20, 23, 22, 25, 24, 27, 26, 29, 28, 31, 30, 1},
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
			batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).Times(2 * len(c.executedBundles)) // 2 times per hash
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
		fmt.Sprintf("%v: %v", "failed to add processed bundle hash to batch", compoundErr),
		func() {
			store.addNewBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)}, batch)
		})
}

func TestStore_deleteOutdatedBundles_RemovesBundles_WhenOld(t *testing.T) {

	caseTable := []struct {
		storedBundleBlockNumber uint64
		currentBlockNumber      uint64
		expectDeleted           bool
	}{
		// Following cases are the warm up phase of the storage
		// when current block number is not large enough to have a history to delete
		{
			storedBundleBlockNumber: 0,
			currentBlockNumber:      bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 1,
			currentBlockNumber:      bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange / 2,
			currentBlockNumber:      bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange - 1,
			currentBlockNumber:      bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange,
			currentBlockNumber:      bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		// Following cases are after the warm up phase, when current block
		// number is large enough to have a history to delete,
		{
			storedBundleBlockNumber: 0,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange / 2,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange - 1,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           true,
		},
		// Following cases are recent enough to not be deleted
		{
			storedBundleBlockNumber: bundle.MaxBlockRange + 1,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: bundle.MaxBlockRange * 3 / 2,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2*bundle.MaxBlockRange - 1,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		{
			storedBundleBlockNumber: 2 * bundle.MaxBlockRange,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
		// future block numbers should not cause deletion
		{
			storedBundleBlockNumber: 2*bundle.MaxBlockRange + 1,
			currentBlockNumber:      2 * bundle.MaxBlockRange,
			expectDeleted:           false,
		},
	}

	for _, c := range caseTable {
		name := fmt.Sprintf("storedBlock=%d/currentBlock=%d", c.storedBundleBlockNumber, c.currentBlockNumber)
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
			if c.currentBlockNumber > bundle.MaxBlockRange {
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

			hash := store.deleteOutdatedBundles(c.currentBlockNumber, batch)
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
		fmt.Sprintf("%v: %v", "failed to delete old processed bundle hash", compoundErr),
		func() { store.deleteOutdatedBundles(bundle.MaxBlockRange+1, batch) })
}

func TestStore_computeNewBundleStateHash_ReturnsExpectedHash(t *testing.T) {
	require := require.New(t)

	oldHash := common.Hash{1, 2, 3}
	addedHash := common.Hash{4, 5, 6}
	deletedHash := common.Hash{7, 8, 9}
	blockNum := uint64(123)

	update := make([]byte, 3*32+8)
	copy(update[:32], oldHash.Bytes())
	copy(update[32:64], addedHash.Bytes())
	copy(update[64:96], deletedHash.Bytes())
	binary.BigEndian.PutUint64(update[96:], blockNum)
	expectedHash := common.Hash(crypto.Keccak256(update))

	require.Equal(expectedHash, computeNewBundleStateHash(oldHash, addedHash, deletedHash, blockNum))
}

// --- helper functions ---

// return execution info with the given hash, for block number 1 and position 0.
func wrapInfo(hash common.Hash) bundle.ExecutionInfo {
	return bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNum:          1,
		Position:          0,
		Count:             1,
	}
}

func updateHistoryHash(blockNum uint64,
	oldHash, addedHash, deletedHash common.Hash) common.Hash {

	// size of the update buffer is:
	//  - 32 bytes for the previous hash
	//  - 32 bytes for the added hashes
	//  - 32 bytes for the deleted hashes
	//  - 8 bytes for the block number
	update := make([]byte, 3*32+8)
	copy(update[:32], oldHash.Bytes())
	copy(update[32:64], addedHash.Bytes())
	copy(update[64:96], deletedHash.Bytes())
	binary.BigEndian.PutUint64(update[96:], blockNum)
	return common.Hash(crypto.Keccak256(update))
}

// storeTableLogMocks initializes a store with mocked table as ProcessedBundles,
// and logger.
// Returns the mocks so expectations can be added on them.
func storeTableLogMocks(t *testing.T) (*Store, *MockstoreTable, *logger.MockLogger, *MockstoreBatch, *MockdbIterator) {
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
func expectCrit(log *logger.MockLogger, msg, label string, err any) {
	log.EXPECT().Crit(msg, gomock.Any(), err).
		Do(func(msg string, ctx ...any) {
			panic(fmt.Sprintf("%v: %v", msg, ctx[1]))
		})
}
