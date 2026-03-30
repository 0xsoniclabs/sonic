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

// uint64ToHash returns unique hashes for input integers.
// It can be used in tests to streamline the creation if unique and deterministic
// hashes without having to hardcode them.
func uint64ToHash(i uint64) common.Hash {
	var b [32]byte
	binary.BigEndian.PutUint64(b[24:], i)
	return common.Hash(b)
}
