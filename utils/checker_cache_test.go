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

package utils

import (
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestNewCachedChecker_CapacityIsEnforced(t *testing.T) {
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
			cache := NewCheckerCache(tc.input)

			// To check the full size, we add entries until one is evicted.
			i := 0
			for ; ; i++ {
				if cache.cache.Add(i, struct{}{}) {
					break
				}
			}

			capacity := max(tc.size/int(unsafe.Sizeof(checkerEntry{})), 1)
			require.Equal(t, capacity, i)
		})
	}
}

func TestCachedChecker_get_MissingEntryReturnsNotFound(t *testing.T) {
	cache := NewCheckerCache(10)
	_, found := cache.get(common.Hash{})
	require.False(t, found)
}

func TestCachedChecker_get_ExistingEntriesAreReturned(t *testing.T) {
	cache := NewCheckerCache(1024)

	entryA := checkerEntry{value: true}
	entryB := checkerEntry{value: false}

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

func TestCachedChecker_ReturnsCachedValueWithoutCallingCheckerFunction(t *testing.T) {
	cache := NewCheckerCache(1024)
	timePoint := time.Now()

	one := types.NewTx(&types.LegacyTx{Nonce: 1})
	two := types.NewTx(&types.LegacyTx{Nonce: 2})
	cache.put(one.Hash(), checkerEntry{
		validUntil: timePoint.Add(time.Minute),
		value:      true,
	})
	cache.put(two.Hash(), checkerEntry{
		validUntil: timePoint.Add(time.Minute),
		value:      false,
	})

	expectNeverCalled := func(tx *types.Transaction) bool {
		t.Fatal("unexpected call to the underlying checker")
		return false
	}
	check := WrapCheck(cache, expectNeverCalled)

	require.True(t, check(one))
	require.False(t, check(two))
}

func TestCachedChecker_NonCachedValue_FetchesNewValue(t *testing.T) {
	cache := NewCheckerCache(1024)

	i := types.NewTx(&types.LegacyTx{Nonce: 1})
	_, found := cache.get(i.Hash())

	require.False(t, found)

	synctest.Test(t, func(t *testing.T) {

		callCount := 0
		checker := func(tx *types.Transaction) bool {
			callCount++
			return true
		}

		// Fetch the value the first time, should call the underlying checker.
		cachedChecker := WrapCheck(cache, checker)
		require.True(t, cachedChecker(i))
		require.Equal(t, 1, callCount)
	})
}

func TestCachedChecker_OutdatedEntry_FetchesNewValue(t *testing.T) {

	cache := NewCheckerCache(1024)
	i := types.NewTx(&types.LegacyTx{Nonce: 1})

	synctest.Test(t, func(t *testing.T) {
		startTime := time.Now()
		validInterval := 200 * time.Millisecond
		cache.put(i.Hash(), checkerEntry{
			validUntil:       startTime.Add(validInterval),
			validityDuration: validInterval,
			value:            true,
		})

		callCount := 0
		checker := func(tx *types.Transaction) bool {
			callCount++
			return false
		}

		time.Sleep(validInterval)
		queryTime := time.Now()

		// The entry is now outdated, so the underlying checker should be called.
		cachedChecker := WrapCheck(cache, checker)
		require.False(t, cachedChecker(i))
		require.Equal(t, 1, callCount)

		// The validity duration should have been increased (exponential backoff).
		entry, found := cache.get(i.Hash())
		require.True(t, found)
		require.False(t, entry.value)
		require.Equal(t, entry.validUntil, queryTime.Add(entry.validityDuration))
		require.Equal(t, entry.validityDuration, 400*time.Millisecond) // 200ms * 2
	})
}

func TestCachedChecker_ValidityDurationIsCapped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		cache := NewCheckerCache(1024)
		i := types.NewTx(&types.LegacyTx{Nonce: 1})

		startTime := time.Now()
		validInterval := 10 * time.Second
		cache.put(i.Hash(), checkerEntry{
			validUntil:       startTime.Add(validInterval),
			validityDuration: validInterval,
			value:            true,
		})

		time.Sleep(validInterval + time.Millisecond)
		queryTime := time.Now()

		callCount := 0
		checker := func(tx *types.Transaction) bool {
			callCount++
			return true
		}

		// The entry is now outdated, so the underlying checker should be called.
		cachedChecker := WrapCheck(cache, checker)
		require.True(t, cachedChecker(i))
		require.Equal(t, 1, callCount)

		// The validity duration should be capped to the maximum (15s).
		entry, found := cache.get(i.Hash())
		require.True(t, found)
		require.True(t, entry.value)
		require.Equal(t, entry.validUntil, queryTime.Add(entry.validityDuration))
		require.Equal(t, entry.validityDuration, 15*time.Second)
	})
}

func TestCachedChecker_DynamicBackoffAdaptation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCheckerCache(1024)
		tx := types.NewTx(&types.LegacyTx{Nonce: 42})

		validity := 200 * time.Millisecond
		cache.put(tx.Hash(), checkerEntry{
			validUntil:       time.Now().Add(-time.Second), // expired
			validityDuration: validity,
			value:            false,
		})

		callCount := 0
		checker := func(tx *types.Transaction) bool {
			callCount++
			return callCount%2 == 0 // alternate result
		}
		cachedChecker := WrapCheck(cache, checker)

		// Simulate repeated accesses, each time after expiry, to test backoff
		for i, expected := range []time.Duration{400 * time.Millisecond, 800 * time.Millisecond, 1600 * time.Millisecond} {
			time.Sleep(validity)
			_ = cachedChecker(tx)
			entry, found := cache.get(tx.Hash())
			require.True(t, found)
			require.Equal(t, expected, entry.validityDuration, "iteration %d", i)
			validity = expected
		}
	})
}

func TestCachedChecker_MinimumValidityDurationIsEnforced(t *testing.T) {
	cache := NewCheckerCache(1024)
	tx := types.NewTx(&types.LegacyTx{Nonce: 99})

	// Insert entry with a very small validity duration
	cache.put(tx.Hash(), checkerEntry{
		validUntil:       time.Now().Add(-time.Second),
		validityDuration: 1 * time.Millisecond,
		value:            true,
	})

	checker := func(tx *types.Transaction) bool { return false }
	cachedChecker := WrapCheck(cache, checker)
	_ = cachedChecker(tx)
	entry, found := cache.get(tx.Hash())
	require.True(t, found)
	require.GreaterOrEqual(t, entry.validityDuration, 200*time.Millisecond)
}

func TestCachedChecker_CacheEvictionCausesRecomputation(t *testing.T) {
	cache := NewCheckerCache(1) // Only one entry allowed
	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})

	callCount := 0
	checker := func(tx *types.Transaction) bool {
		callCount++
		return true
	}
	cachedChecker := WrapCheck(cache, checker)

	// First access for tx1, should call checker
	require.True(t, cachedChecker(tx1))
	require.Equal(t, 1, callCount)

	// Access for tx2, should evict tx1
	require.True(t, cachedChecker(tx2))
	require.Equal(t, 2, callCount)

	// Access tx1 again, should call checker again due to eviction
	require.True(t, cachedChecker(tx1))
	require.Equal(t, 3, callCount)
}
