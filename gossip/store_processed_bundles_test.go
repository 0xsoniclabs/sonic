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

func TestStore_HasBundleRecentlyBeenProcessed_Returns(t *testing.T) {

	cases := map[string]struct {
		hash []byte
	}{
		"bundle found ": {
			hash: []byte{1, 2, 3, 4},
		},
		"bundle not found ": {
			hash: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			store, table, _, _, _ := storeTableLogMocks(t)
			hash := common.Hash{1, 2, 3}
			table.EXPECT().Get(getEntryKey(hash)).Return(c.hash, nil)
			got := store.HasBundleRecentlyBeenProcessed(hash)
			require.Equal(t, c.hash != nil, got)
		})
	}
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

func TestStore_GetBundleExecutionInfo_ReturnsInfoForAddedBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	// Prepare a set of bundles across multiple blocks
	hashes := []common.Hash{
		{1, 2, 3},    // 0: first in history, first in block 1
		{4, 5, 6},    // 1: middle in block 1
		{7, 8, 9},    // 2: last in block 1
		{10, 11, 12}, // 3: first in block 512
		{13, 14, 15}, // 4: middle in block 512
		{16, 17, 18}, // 5: last in block 512
		{19, 20, 21}, // 6: first in last block of history
		{22, 23, 24}, // 7: middle in last block of history
		{25, 26, 27}, // 8: last in last block of history
	}
	infos := []bundle.ExecutionInfo{
		{
			ExecutionPlanHash: hashes[0],
			BlockNum:          1,
			Position:          0,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[1],
			BlockNum:          1,
			Position:          1,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[2],
			BlockNum:          1,
			Position:          2,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[3],
			BlockNum:          512,
			Position:          0,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[4],
			BlockNum:          512,
			Position:          1,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[5],
			BlockNum:          512,
			Position:          2,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[6],
			BlockNum:          1023,
			Position:          0,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[7],
			BlockNum:          1023,
			Position:          1,
			Count:             1,
		},
		{
			ExecutionPlanHash: hashes[8],
			BlockNum:          1023,
			Position:          2,
			Count:             1,
		},
	}
	// Add all bundles to store
	store.AddProcessedBundles(1, infos[:3])
	store.AddProcessedBundles(512, infos[3:6])
	store.AddProcessedBundles(1023, infos[6:])

	type testCase struct {
		hash     common.Hash
		expected *bundle.ExecutionInfo
	}
	tests := map[string]testCase{
		"not found (unknown hash)": {
			hash:     common.Hash{99, 99, 99},
			expected: nil,
		},
		"first in first block": {
			hash:     hashes[0],
			expected: &infos[0],
		},
		"middle in first block ": {
			hash:     hashes[1],
			expected: &infos[1],
		},
		"last in first block": {
			hash:     hashes[2],
			expected: &infos[2],
		},
		"first in middle block": {
			hash:     hashes[3],
			expected: &infos[3],
		},
		"middle in middle block": {
			hash:     hashes[4],
			expected: &infos[4],
		},
		"last in middle block": {
			hash:     hashes[5],
			expected: &infos[5],
		},
		"first in last block": {
			hash:     hashes[6],
			expected: &infos[6],
		},
		"middle in last block": {
			hash:     hashes[7],
			expected: &infos[7],
		},
		"last in last block": {
			hash:     hashes[8],
			expected: &infos[8],
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			info := store.GetBundleExecutionInfo(tc.hash)

			require.Equal(tc.expected, info)

		})
	}
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
			expectedHash := updateHistoryHash(1, initialHash, addedHash, common.Hash{})
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

func TestStore_ProcessedBundles_OldHashAffectsNewHash(t *testing.T) {
	require := require.New(t)
	store1, err := NewMemStore(t)
	require.NoError(err)
	store2, err := NewMemStore(t)
	require.NoError(err)

	getHistoryHashes := func() (common.Hash, common.Hash) {
		_, hashA := store1.GetProcessedBundleHistoryHash()
		_, hashB := store2.GetProcessedBundleHistoryHash()
		return hashA, hashB
	}

	addBundlesInBlock := func(hashA, hashB common.Hash, blockNum uint64) {
		store1.AddProcessedBundles(blockNum, []bundle.ExecutionInfo{wrapInfo(hashA, blockNum)})
		store2.AddProcessedBundles(blockNum, []bundle.ExecutionInfo{wrapInfo(hashB, blockNum)})
	}

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}
	hash4 := common.Hash{10, 11, 12}

	// initially, both stores have the same hash (zero)
	hashA0, hashB0 := getHistoryHashes()
	require.Equal(hashA0, hashB0)
	require.Zero(hashA0)

	// adding two different bundles, changes the hash in both stores
	addBundlesInBlock(hash1, hash2, 1)
	hash1A, hash1B := getHistoryHashes()
	require.NotEqual(hash1A, hash1B)

	// adding the same bundle in both stores must produce different new hashes
	addBundlesInBlock(hash3, hash3, 2)
	hash2A, hash2B := getHistoryHashes()
	require.NotEqual(hash2A, hash2B)

	// adding the same bundle in both stores with a gapped history functions
	// normally since the block number does not affect the generation of the
	// history hash (except for being inside the execution plans)
	addBundlesInBlock(hash4, hash4, 5)
	hash3A, hash3B := getHistoryHashes()
	require.NotEqual(hash3A, hash3B)
}

func TestStore_AddProcessedBundles_ComputesCorrectHashIteratively(t *testing.T) {
	// This test iteratively adds bundles and verifies the history hash after
	// every addition. It covers:
	//  1. Base case: adding a single bundle with no deletions.
	//  2. Adding multiple bundles in a single block.
	//  3. Triggering deletion of multiple old bundles when the block range expires.
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}
	hash4 := common.Hash{10, 11, 12}
	hash5 := common.Hash{13, 14, 15}
	hash6 := common.Hash{16, 17, 18}
	hash7 := common.Hash{19, 20, 21}

	// --- Step 1: base case, add a single bundle (no deletions) ---
	_, historyHash := store.GetProcessedBundleHistoryHash()
	require.Zero(historyHash)

	store.AddProcessedBundles(0, []bundle.ExecutionInfo{wrapInfo(hash1, 0)})

	blockNum, historyHash := store.GetProcessedBundleHistoryHash()
	require.Equal(uint64(0), blockNum)
	expectedHash := updateHistoryHash(0, common.Hash{}, hash1, common.Hash{})
	require.Equal(expectedHash, historyHash)

	// --- Step 2: add multiple bundles in a single block (no deletions) ---
	prevHash := historyHash

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{
		wrapInfo(hash2, 1),
		wrapInfo(hash3, 1),
		wrapInfo(hash4, 1),
	})

	blockNum, historyHash = store.GetProcessedBundleHistoryHash()
	require.Equal(uint64(1), blockNum)
	addedHash := xorHash(xorHash(hash2, hash3), hash4)
	expectedHash = updateHistoryHash(1, prevHash, addedHash, common.Hash{})
	require.Equal(expectedHash, historyHash)

	// --- Step 3: add more bundles in a later block, still no deletions ---
	prevHash = historyHash

	store.AddProcessedBundles(2, []bundle.ExecutionInfo{
		wrapInfo(hash5, 2),
		wrapInfo(hash6, 2),
	})

	blockNum, historyHash = store.GetProcessedBundleHistoryHash()
	require.Equal(uint64(2), blockNum)
	addedHash = xorHash(hash5, hash6)
	expectedHash = updateHistoryHash(2, prevHash, addedHash, common.Hash{})
	require.Equal(expectedHash, historyHash)

	// --- Step 4: advance to MaxBlockRange-1 so that block 0's bundle expires ---
	// Block 0 had hash1, so it should be deleted.
	prevHash = historyHash
	newBlock := uint64(bundle.MaxBlockRange - 1)

	store.AddProcessedBundles(newBlock, []bundle.ExecutionInfo{
		wrapInfo(hash7, newBlock),
	})

	blockNum, historyHash = store.GetProcessedBundleHistoryHash()
	require.Equal(newBlock, blockNum)
	deletedHash := hash1 // only block 0's bundle is old enough to expire
	expectedHash = updateHistoryHash(newBlock, prevHash, hash7, deletedHash)
	require.Equal(expectedHash, historyHash)

	// --- Step 5: advance to MaxBlockRange so that block 1's bundles expire ---
	// Block 1 had hash2, hash3, hash4, so all three should be deleted.
	prevHash = historyHash
	newBlock = uint64(bundle.MaxBlockRange)
	hash8 := common.Hash{22, 23, 24}

	store.AddProcessedBundles(newBlock, []bundle.ExecutionInfo{
		wrapInfo(hash8, newBlock),
	})

	blockNum, historyHash = store.GetProcessedBundleHistoryHash()
	require.Equal(newBlock, blockNum)
	deletedHash = xorHash(xorHash(hash2, hash3), hash4) // all three from block 1
	expectedHash = updateHistoryHash(newBlock, prevHash, hash8, deletedHash)
	require.Equal(expectedHash, historyHash)

	// --- Step 6: advance to MaxBlockRange+1 so that block 2's bundles expire ---
	// Block 2 had hash5, hash6, so both should be deleted.
	prevHash = historyHash
	newBlock = uint64(bundle.MaxBlockRange + 1)
	hash9 := common.Hash{25, 26, 27}
	hash10 := common.Hash{28, 29, 30}

	store.AddProcessedBundles(newBlock, []bundle.ExecutionInfo{
		wrapInfo(hash9, newBlock),
		wrapInfo(hash10, newBlock),
	})

	blockNum, historyHash = store.GetProcessedBundleHistoryHash()
	require.Equal(newBlock, blockNum)
	addedHash = xorHash(hash9, hash10)
	deletedHash = xorHash(hash5, hash6) // both from block 2
	expectedHash = updateHistoryHash(newBlock, prevHash, addedHash, deletedHash)
	require.Equal(expectedHash, historyHash)
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

func TestStore_AddProcessedBundles_LogsOnInvalidBlockNum(t *testing.T) {
	store, _, log, _, _ := storeTableLogMocks(t)

	current := uint64(math.MaxUint64)
	expectCrit(log, "invalid block number in execution info", "expected", current, "got", uint64(1))

	execInfo := bundle.ExecutionInfo{
		ExecutionPlanHash: common.Hash{1, 2, 3},
		BlockNum:          1, // use a different block number than the current
		Position:          0,
		Count:             1,
	}
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("invalid block number in execution info: %v", []any{"expected", current, "got", uint64(1)}),
		func() {
			store.AddProcessedBundles(current, []bundle.ExecutionInfo{execInfo})
		})
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

func TestStore_computeNewBundleStateHash_ReturnsExpectedHash(t *testing.T) {

	testCases := []struct {
		name        string
		oldHash     common.Hash
		addedHash   common.Hash
		deletedHash common.Hash
		blockNum    uint64
		expected    common.Hash
	}{
		{
			name:        "all zeros",
			oldHash:     common.Hash{},
			addedHash:   common.Hash{},
			deletedHash: common.Hash{},
			blockNum:    0,
			expected:    common.HexToHash("c24cd7564e291016870aca25c634ca9ab560c07c935b6c0fe3b559cbd3de7501"),
		},
		{
			name:        "simple nonzero",
			oldHash:     common.Hash{1, 2, 3},
			addedHash:   common.Hash{4, 5, 6},
			deletedHash: common.Hash{7, 8, 9},
			blockNum:    123,
			expected:    common.HexToHash("21f799a0c47f7c86bfe025aaa725d5a855e05340f257d200eae6fa3a3f5d1319"),
		},
		{
			name:        "max blockNum with partial hashes",
			oldHash:     common.Hash{0xff, 0xff, 0xff},
			addedHash:   common.Hash{0xaa, 0xbb, 0xcc},
			deletedHash: common.Hash{0x11, 0x22, 0x33},
			blockNum:    math.MaxUint64,
			expected:    common.HexToHash("c442b47e6caf1856c00f46452c45b3f669b9c45f47da9c7bf54cc32e408e9442"),
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			got := computeNewBundleStateHash(v.oldHash, v.addedHash, v.deletedHash, v.blockNum)
			ref := alternativeComputeImpl(t, v.oldHash, v.addedHash, v.deletedHash, v.blockNum)
			require.Equal(t, v.expected, got, "computed hash should match expected value")
			require.Equal(t, ref, got, "actual implementation should match alternative implementation")
		})
	}
}

func alternativeComputeImpl(
	st *testing.T,
	oldHash, addedHash, deletedHash common.Hash,
	blockNum uint64,
) common.Hash {
	h := crypto.NewKeccakState()
	h.Write(oldHash.Bytes())
	h.Write(addedHash.Bytes())
	h.Write(deletedHash.Bytes())
	require.NoError(st, binary.Write(h, binary.BigEndian, blockNum))
	var out common.Hash
	_, err := h.Read(out[:])
	require.NoError(st, err)
	return out
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
			ref := alternativeComputeImpl(t, tc.oldHash, tc.addedHash, tc.deletedHash, tc.blockNum)
			require.Equal(t, ref, got, "actual implementation should match alternative implementation")
		})
	}
}

func TestStore_RetainsAllBundlesRequiredToCoverTheMaximumBlockRange(t *testing.T) {
	require := require.New(t)
	numBlocks := 3 * bundle.MaxBlockRange

	store, err := NewMemStore(t)
	require.NoError(err)

	// Create a list of execution plan hashes indexed by their block numbers.
	hashes := []common.Hash{}
	for i := range numBlocks {
		hashes = append(hashes, common.Hash{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
	}

	// While progressing through the blocks, all execution plans must be retained
	// until their maximum block range has expired.
	for currentBlockNumber := range numBlocks {

		// Check that the store covers exactly the plans of the past that are
		// allowed to be included in the current block (before adding it).
		for block := uint64(0); block < currentBlockNumber; block++ {
			blockRange := bundle.MakeMaxRangeStartingAt(block)
			want := blockRange.IsInRange(currentBlockNumber)
			require.Equal(
				want, store.HasBundleRecentlyBeenProcessed(hashes[block]),
				"Current block %d, checking plan with range [%d,%d]",
				currentBlockNumber, blockRange.Earliest, blockRange.Latest,
			)
		}

		store.AddProcessedBundles(currentBlockNumber, []bundle.ExecutionInfo{
			{
				ExecutionPlanHash: hashes[currentBlockNumber],
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
