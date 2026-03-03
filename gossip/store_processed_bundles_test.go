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

func TestStore_HasRecentlyBeenProcessed_TracksAddedBundleHashes(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}
	hash3 := common.Hash{7, 8, 9}

	recentlyProcessed := func(hash common.Hash) bool {
		res, err := store.HasRecentlyBeenProcessed(hash)
		require.NoError(err)
		return res
	}

	require.False(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})
	require.NoError(err)

	require.True(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2), wrapInfo(hash3)})
	require.NoError(err)

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
		res, err := store.HasRecentlyBeenProcessed(hash)
		require.NoError(err)
		return res
	}

	require.False(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})
	require.NoError(err)

	require.True(recentlyProcessed(hash1))
	require.False(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(1+bundle.MaxBlockRange/2, []bundle.ExecutionInfo{wrapInfo(hash2)})
	require.NoError(err)

	require.True(recentlyProcessed(hash1))
	require.True(recentlyProcessed(hash2))
	require.False(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(1+bundle.MaxBlockRange, []bundle.ExecutionInfo{wrapInfo(hash3)})
	require.NoError(err)

	require.False(recentlyProcessed(hash1))
	require.True(recentlyProcessed(hash2))
	require.True(recentlyProcessed(hash3))

	err = store.AddProcessedBundles(1+2*bundle.MaxBlockRange, []bundle.ExecutionInfo{})
	require.NoError(err)

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
		Hash:     hash1,
		BlockNum: 1,
		Position: 0,
	}
	info2 := bundle.ExecutionInfo{
		Hash:     hash2,
		BlockNum: 2,
		Position: 1,
	}

	info, err := store.GetBundleExecutionInfo(hash1)
	require.NoError(err)
	require.Nil(info)
	info, err = store.GetBundleExecutionInfo(hash2)
	require.NoError(err)
	require.Nil(info)

	err = store.AddProcessedBundles(1, []bundle.ExecutionInfo{info1, info2})
	require.NoError(err)

	resInfo1, err := store.GetBundleExecutionInfo(hash1)
	require.NoError(err)
	require.Equal(info1, *resInfo1)

	resInfo2, err := store.GetBundleExecutionInfo(hash2)
	require.NoError(err)
	require.Equal(info2, *resInfo2)
}

func TestStore_AddProcessedBundles_UpdatesHistoryHash(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	hash1 := common.Hash{1, 2, 3}
	hash2 := common.Hash{4, 5, 6}

	_, initialHash, err := store.GetProcessedBundleHistoryHash()
	require.NoError(err)

	err = store.AddProcessedBundles(1, []bundle.ExecutionInfo{wrapInfo(hash1)})
	require.NoError(err)

	_, hashAfterFirstAdd, err := store.GetProcessedBundleHistoryHash()
	require.NoError(err)
	require.NotEqual(initialHash, hashAfterFirstAdd)

	err = store.AddProcessedBundles(2, []bundle.ExecutionInfo{wrapInfo(hash2)})
	require.NoError(err)

	_, hashAfterSecondAdd, err := store.GetProcessedBundleHistoryHash()
	require.NoError(err)
	require.NotEqual(hashAfterFirstAdd, hashAfterSecondAdd)
}

func TestStore_GetProcessedBundleHistoryHash_InitiallyZero(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	blockNum, hash, err := store.GetProcessedBundleHistoryHash()
	require.NoError(err)
	require.Zero(blockNum)
	require.Zero(hash)
}

func wrapInfo(hash common.Hash) bundle.ExecutionInfo {
	return bundle.ExecutionInfo{
		Hash:     hash,
		BlockNum: 1,
		Position: 0,
	}
}
