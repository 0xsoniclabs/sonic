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

func wrapInfo(hash common.Hash) bundle.ExecutionInfo {
	return bundle.ExecutionInfo{
		ExecutionPlanHash: hash,
		BlockNum:          1,
		Position:          0,
	}
}
