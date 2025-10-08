// Copyright 2025 Sonic Operations Ltd
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

package evmcore

import (
	"testing"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewSubsidiesCheckerCache_CapacityIsEnforced(t *testing.T) {
	const MiB = 1024 * 1024
	tests := map[string]struct {
		input int
		size  int
	}{
		"negative": {input: -10, size: 10 * MiB},
		"zero":     {input: 0, size: 10 * MiB},
		"one":      {input: 1, size: 1},
		"small":    {input: 100, size: 100},
		"large":    {input: 200 * MiB, size: 200 * MiB},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cache := newSubsidiesCheckerCache(tc.input)

			// To check the full size, we add entries until one is evicted.
			i := 0
			for ; ; i++ {
				if cache.cache.Add(i, struct{}{}) {
					break
				}
			}

			capacity := max(tc.size/getSizeOfSubsidiesCheckerCacheEntry(), 1)
			require.Equal(t, capacity, i)
		})
	}
}

func TestSubsidiesCheckerCache_MissingEntry_ReturnsNotFound(t *testing.T) {
	cache := newSubsidiesCheckerCache(10)
	_, found := cache.get(common.Hash{})
	require.False(t, found)
}

func TestSubsidiesCheckerCache_PresentEntries_AreReturned(t *testing.T) {
	cache := newSubsidiesCheckerCache(1024)

	entryA := subsidiesCheckerCacheEntry{covered: true}
	entryB := subsidiesCheckerCacheEntry{covered: false}

	hashA := common.Hash{0x1}
	hashB := common.Hash{0x2}

	_, found := cache.get(hashA)
	require.False(t, found)
	_, found = cache.get(hashB)
	require.False(t, found)

	// -- add first element --
	cache.put(hashA, entryA)
	got, found := cache.get(hashA)
	require.True(t, found)
	require.Equal(t, entryA, got)

	_, found = cache.get(hashB)
	require.False(t, found)

	// -- add second element --
	cache.put(hashB, entryB)
	got, found = cache.get(hashA)
	require.True(t, found)
	require.Equal(t, entryA, got)

	got, found = cache.get(hashB)
	require.True(t, found)
	require.Equal(t, entryB, got)
}

func TestSubsidiesCheckerCache_Wrap_WrapsCache(t *testing.T) {
	cache := newSubsidiesCheckerCache(10)

	checker := &cachedSubsidiesChecker{}
	res := cache.wrap(checker)
	require.Equal(t, res.cache, cache)
	require.Equal(t, res.checker, checker)
}

func TestGetSizeOfSubsidiesCheckerCacheEntry(t *testing.T) {
	require.Equal(t,
		int(unsafe.Sizeof(subsidiesCheckerCacheEntry{})),
		getSizeOfSubsidiesCheckerCacheEntry(),
	)
}

func TestCachedSubsidiesChecker_isSponsored_UsesValueFromCache(t *testing.T) {
	cache := newSubsidiesCheckerCache(1024)

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})
	require.NotEqual(t, tx1.Hash(), tx2.Hash())

	now := time.Now()
	cache.put(tx1.Hash(), subsidiesCheckerCacheEntry{
		validUntil: now.Add(time.Minute),
		covered:    true,
	})
	cache.put(tx2.Hash(), subsidiesCheckerCacheEntry{
		validUntil: now.Add(time.Minute),
		covered:    false,
	})

	checker := cache.wrap(nil)
	require.True(t, checker.isSponsored(tx1))
	require.False(t, checker.isSponsored(tx2))
}

func TestCachedSubsidiesChecker_isSponsored_NonCachedValue_FetchesNewValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	checker := NewMocksubsidiesChecker(ctrl)

	cache := newSubsidiesCheckerCache(1024)

	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	_, found := cache.get(tx.Hash())
	require.False(t, found)

	cachedChecker := cache.wrap(checker)

	// Fetch the value the first time, should call the underlying checker.
	now := time.Now()
	checker.EXPECT().isSponsored(tx).Return(true).Times(1)
	require.True(t, cachedChecker._isSponsored(tx, now))

	// The result should be cached now.
	entry, found := cache.get(tx.Hash())
	require.True(t, found)
	require.True(t, entry.covered)
	require.True(t, entry.validUntil.After(now))
	require.Equal(t, entry.validityDuration, 200*time.Millisecond)

	// Second call should use the cache, so no call to the underlying checker.
	require.True(t, cachedChecker._isSponsored(tx, now.Add(time.Millisecond)))
}

func TestCachedSubsidiesChecker_isSponsored_OutdatedEntry_FetchesNewValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	checker := NewMocksubsidiesChecker(ctrl)

	cache := newSubsidiesCheckerCache(1024)
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})

	now := time.Now()
	validInterval := 200 * time.Millisecond
	cache.put(tx.Hash(), subsidiesCheckerCacheEntry{
		validUntil:       now.Add(validInterval),
		validityDuration: validInterval,
		covered:          true,
	})

	now = now.Add(validInterval + time.Millisecond)

	// The entry is now outdated, so the underlying checker should be called.
	cachedChecker := cache.wrap(checker)
	checker.EXPECT().isSponsored(tx).Return(false)
	require.False(t, cachedChecker._isSponsored(tx, now))

	// The validity duration should have been increased (exponential backoff).
	entry, found := cache.get(tx.Hash())
	require.True(t, found)
	require.False(t, entry.covered)
	require.Equal(t, entry.validUntil, now.Add(entry.validityDuration))
	require.Equal(t, entry.validityDuration, 400*time.Millisecond) // 200ms * 2
}

func TestCacheSubsidiesChecker_isSponsored_ValidityDurationIsCapped(t *testing.T) {
	ctrl := gomock.NewController(t)
	checker := NewMocksubsidiesChecker(ctrl)

	cache := newSubsidiesCheckerCache(1024)
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})

	now := time.Now()
	validInterval := 10 * time.Second
	cache.put(tx.Hash(), subsidiesCheckerCacheEntry{
		validUntil:       now.Add(validInterval),
		validityDuration: validInterval,
		covered:          true,
	})

	now = now.Add(validInterval + time.Millisecond)

	// The entry is now outdated, so the underlying checker should be called.
	cachedChecker := cache.wrap(checker)
	checker.EXPECT().isSponsored(tx).Return(true)
	require.True(t, cachedChecker._isSponsored(tx, now))

	// The validity duration should be capped to the maximum (15s).
	entry, found := cache.get(tx.Hash())
	require.True(t, found)
	require.True(t, entry.covered)
	require.Equal(t, entry.validUntil, now.Add(entry.validityDuration))
	require.Equal(t, entry.validityDuration, 15*time.Second)
}
