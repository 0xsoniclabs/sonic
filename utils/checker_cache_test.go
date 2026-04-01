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
	"strconv"
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
			cache := NewCheckerCache[bool](tc.input)

			// To check the full size, we add entries until one is evicted.
			i := 0
			for ; ; i++ {
				if cache.cache.Add(i, struct{}{}) {
					break
				}
			}

			capacity := max(tc.size/int(unsafe.Sizeof(checkerEntry[bool]{})), 1)
			require.Equal(t, capacity, i)
		})
	}
}

func TestCachedChecker_get_MissingEntryReturnsNotFound(t *testing.T) {
	cache := NewCheckerCache[bool](10)
	_, found := cache.get(common.Hash{})
	require.False(t, found)
}

func TestCachedChecker_get_ExistingEntriesAreReturned(t *testing.T) {
	cache := NewCheckerCache[bool](1024)

	entryA := checkerEntry[bool]{value: true}
	entryB := checkerEntry[bool]{value: false}

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
	cache := NewCheckerCache[bool](1024)
	timePoint := time.Now()

	one := hasheableInteger(1)
	two := hasheableInteger(2)
	cache.put(one.Hash(), checkerEntry[bool]{
		validUntil: timePoint.Add(time.Minute),
		value:      true,
	})
	cache.put(two.Hash(), checkerEntry[bool]{
		validUntil: timePoint.Add(time.Minute),
		value:      false,
	})

	expectNeverCalled := func(hasheableInteger) bool {
		t.Fatal("unexpected call to the underlying checker")
		return false
	}
	check := WrapCheck(cache, expectNeverCalled)

	require.True(t, check(hasheableInteger(1)))
	require.False(t, check(hasheableInteger(2)))
}

func TestCachedChecker_CheckerIsGeneric(t *testing.T) {
	cache := NewCheckerCache[bool](1024)
	callCount := 0

	require.True(t,
		WrapCheck(cache, func(hasheableInteger) bool {
			callCount++
			return true
		})(hasheableInteger(42)))
	require.Equal(t, 1, callCount)

	require.True(t,
		WrapCheck(cache, func(hasheableFloat) bool {
			callCount++
			return true
		})(hasheableFloat(1.33)))
	require.Equal(t, 2, callCount)

	require.True(t,
		WrapCheck(cache, func(*types.Transaction) bool {
			callCount++
			return true
		})(types.NewTx(&types.LegacyTx{})))
	require.Equal(t, 3, callCount)
}

func TestCachedChecker_NonCachedValue_FetchesNewValue(t *testing.T) {
	cache := NewCheckerCache[bool](1024)

	i := hasheableInteger(1)
	_, found := cache.get(i.Hash())

	require.False(t, found)

	synctest.Test(t, func(t *testing.T) {

		timePoint := time.Now()

		callCount := 0
		checker := func(tx hasheableInteger) bool {
			callCount++
			return true
		}

		// Fetch the value the first time, should call the underlying checker.
		cachedChecker := WrapCheck(cache, checker)
		require.True(t, cachedChecker(i))
		require.Equal(t, 1, callCount)

		// The result should be cached now.
		entry, found := cache.get(i.Hash())
		require.True(t, found)
		require.True(t, entry.value)
		require.True(t, entry.validUntil.After(timePoint))
		require.Equal(t, entry.validityDuration, 200*time.Millisecond)

		time.Sleep(entry.validityDuration - 1)

		// Second call should use the cache, so no call to the underlying checker.
		require.True(t, cachedChecker(i))
		require.Equal(t, 1, callCount)
	})
}

func TestCachedChecker_OutdatedEntry_FetchesNewValue(t *testing.T) {

	cache := NewCheckerCache[bool](1024)
	i := hasheableInteger(1)

	synctest.Test(t, func(t *testing.T) {
		startTime := time.Now()
		validInterval := 200 * time.Millisecond
		cache.put(i.Hash(), checkerEntry[bool]{
			validUntil:       startTime.Add(validInterval),
			validityDuration: validInterval,
			value:            true,
		})

		callCount := 0
		checker := func(tx hasheableInteger) bool {
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

		cache := NewCheckerCache[bool](1024)
		i := hasheableInteger(1)

		startTime := time.Now()
		validInterval := 10 * time.Second
		cache.put(i.Hash(), checkerEntry[bool]{
			validUntil:       startTime.Add(validInterval),
			validityDuration: validInterval,
			value:            true,
		})

		time.Sleep(validInterval + time.Millisecond)
		queryTime := time.Now()

		callCount := 0
		checker := func(tx hasheableInteger) bool {
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

type hasheableInteger int

func (h hasheableInteger) Hash() common.Hash {
	var hash common.Hash
	copy(hash[:], []byte(strconv.Itoa(int(h))))
	return hash
}

type hasheableFloat float32

func (h hasheableFloat) Hash() common.Hash {
	var hash common.Hash
	copy(hash[:], []byte(strconv.FormatFloat(float64(h), 'f', -1, 32)))
	return hash
}
