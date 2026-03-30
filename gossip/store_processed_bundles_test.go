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
