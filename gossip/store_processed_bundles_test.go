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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStore_HasBundleRecentlyBeenProcessed_TracksAddedBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	recentlyProcessed := func(hash common.Hash) bool {
		return store.HasBundleRecentlyBeenProcessed(hash)
	}

	require.False(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})

	require.True(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2), wrapInfo(hash3)})

	require.True(recentlyProcessed(hash1))
	require.True(recentlyProcessed(hash2))
	require.True(recentlyProcessed(hash3))
}

func TestStore_HasRecentlyBeenProcessed_CleansUpOldBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	recentlyProcessed := func(hash common.Hash) bool {
		return store.HasBundleRecentlyBeenProcessed(hash)
	}

	require.False(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})

	require.True(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	store.AddProcessedBundles(1+bundle.MaxBlockRange/2, []bundle.ExecutionInfo{wrapInfo(hash2)})

	require.True(recentlyProcessed(hash1))
	require.True(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	store.AddProcessedBundles(1+bundle.MaxBlockRange, []bundle.ExecutionInfo{wrapInfo(hash3)})

	require.False(recentlyProcessed(hash1))
	require.True(recentlyProcessed(hash2))
	require.True(recentlyProcessed(hash3))

	store.AddProcessedBundles(1+2*bundle.MaxBlockRange, []bundle.ExecutionInfo{})

	require.False(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))
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
		Count:             3,
	}

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

func TestStore_AddProcessedBundles_UpdatesHistoryHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}

	_, initialHash := store.GetProcessedBundleHistoryHash()

	store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})

	_, hashAfterFirstAdd := store.GetProcessedBundleHistoryHash()
	require.NotEqual(initialHash, hashAfterFirstAdd)

	store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2)})

	_, hashAfterSecondAdd := store.GetProcessedBundleHistoryHash()
	require.NotEqual(hashAfterFirstAdd, hashAfterSecondAdd)
}

func TestStore_GetProcessedBundleHistoryHash_InitiallyZero(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum, hash := store.GetProcessedBundleHistoryHash()
	require.Zero(blockNum)
	require.Zero(hash)
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
	value := make([]byte, 8+32)
	binary.BigEndian.PutUint64(value[:8], blockNum)
	copy(value[8:], hash[:])

	err = store.SetRawProcessedBundle(BundleKV{Key: nil, Value: value}) // nil key for history hash
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
	info := wrapInfo(hash, blockNum)

	// encode entry value.
	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[:8], info.BlockNum)
	binary.BigEndian.PutUint32(data[8:12], info.Position)
	binary.BigEndian.PutUint32(data[12:], info.Count)

	entry := BundleKV{
		Key:   append([]byte{'e'}, hash.Bytes()...),
		Value: data,
	}

	err = store.SetRawProcessedBundle(entry)
	require.NoError(err)

	resInfo := store.GetBundleExecutionInfo(hash)
	require.NotNil(resInfo)
	require.Equal(info.ExecutionPlanHash, resInfo.ExecutionPlanHash)
	require.Equal(info.BlockNum, resInfo.BlockNum)
	require.Equal(info.Position, resInfo.Position)
	require.Equal(info.Count, resInfo.Count)
}

func TestStore_SetRawProcessedBundle_AddsIndexEntry(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash := common.Hash{1, 2, 3}
	blockNum := uint64(10)
	info := wrapInfo(hash, blockNum)

	// encode entry value.
	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[:8], info.BlockNum)
	binary.BigEndian.PutUint32(data[8:12], info.Position)
	binary.BigEndian.PutUint32(data[12:], info.Count)

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

func TestStore_DumpProcessedBundles_ReturnsAllAddedEntries(t *testing.T) {

	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	entries := make([]bundle.ExecutionInfo, bundle.MaxBlockRange+1)
	for i := range bundle.MaxBlockRange + 1 {
		hash := Uint32ToBytes(uint32(i))
		entries[i] = wrapInfo(common.BytesToHash(hash), uint64(i))
		store.AddProcessedBundles(uint64(i), []bundle.ExecutionInfo{entries[i]})
	}

	dumpedEntries := store.DumpProcessedBundles()
	expectedSize := int(
		(8 + 32) + // bundle history hash
			(bundle.MaxBlockRange-1)*
				// key size: 'e' prefix + hash
				// value size: blockNum (8 bytes) + position (4 bytes) + count (4 bytes)
				((1+32)+16) +
			8*bundle.MaxBlockRange) // key/value size

	// first entry is the history hash, then we have one entry per execution info
	actualSize := 0
	for _, entry := range dumpedEntries {
		actualSize += len(entry)
	}

	require.Equal(expectedSize, actualSize,
		fmt.Sprintf("expected %d dumped entries, got %d",
			expectedSize, actualSize))
}

func TestStore_DumpProcessedBundles_ReturnsEncodedEntries(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash := common.Hash{1, 2, 3}
	blockNum := uint64(10)
	info := wrapInfo(hash, blockNum)

	store.AddProcessedBundles(blockNum, []bundle.ExecutionInfo{info})

	dumpedEntries := store.DumpProcessedBundles()
	require.Len(dumpedEntries, 2) // history hash + 1 entry

	// check that the dumped entry matches the expected encoding of the added entry
	expectedEntry := BundleKV{
		Key:   append([]byte{'e'}, hash.Bytes()...),
		Value: make([]byte, 16),
	}
	binary.BigEndian.PutUint64(expectedEntry.Value[:8], info.BlockNum)
	binary.BigEndian.PutUint32(expectedEntry.Value[8:12], info.Position)
	binary.BigEndian.PutUint32(expectedEntry.Value[12:], info.Count)

	require.Contains(dumpedEntries, expectedEntry.Encode())
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
