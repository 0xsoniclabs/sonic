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
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
)

type TransactionCheckFunc func(*types.Transaction) bool

// TransactionCheckCache is a cache for storing the results of expensive transaction checks.
// It uses an LRU cache internally to store the results and evict old entries when the cache is full.
//
// Backoff Scheme:
// Each cached check result is associated with a validity duration, which determines how long
// the cached value can be reused before the check must be recomputed. The validity duration
// starts at an initial value (200ms) and is exponentially increased (doubled) each time the
// check is recomputed, up to a maximum duration (15s). This means:
//   - On the first cache miss, the check is performed and cached with a short validity (200ms).
//   - If the same transaction is checked again after the validity expires, the check is recomputed,
//     and the validity duration is doubled (e.g., 400ms, 800ms, etc.), up to the maximum.
//   - If the transaction is checked again before the validity expires, the cached result is reused.
//   - This exponential backoff reduces the frequency of expensive checks for frequently-seen
//     transactions, while still ensuring that results are periodically refreshed.
//
// This approach balances performance (by avoiding repeated expensive checks) and correctness
// (by ensuring results are eventually refreshed), adapting dynamically to transaction access patterns.
type TransactionCheckCache struct {
	cache *lru.Cache
}

// NewCheckerCache creates a new CheckerCache with the given size in bytes.
// if size is 0 then a default of 10MiB will be used as max size.
func NewCheckerCache(size int) *TransactionCheckCache {
	if size <= 0 {
		size = 10 * 1024 * 1024 // 10 MiB
	}

	entrySize := reflect.TypeOf(checkerEntry{}).Size()
	capacity := max(size/(int(entrySize)), 1)
	cache, _ := lru.New(capacity) // only fails if capacity <= 0
	return &TransactionCheckCache{cache: cache}
}

func (c *TransactionCheckCache) get(txHash common.Hash) (checkerEntry, bool) {
	if entry, ok := c.cache.Get(txHash); ok {
		return entry.(checkerEntry), true
	}
	return checkerEntry{}, false
}

func (c *TransactionCheckCache) put(txHash common.Hash, entry checkerEntry) {
	c.cache.Add(txHash, entry)
}

// WrapCheck wraps an expensive transaction check function with caching functionality.
// The returned checker will use the provided TransactionCheckCache to store and retrieve
// results, automatically applying the exponential backoff scheme for cache validity.
//
// Usage:
//
//	cache := utils.NewCheckerCache(0) // Use default size
//	expensiveCheck := func(tx *types.Transaction) bool {
//	    // ... perform expensive validation ...
//	    return true
//	}
//	cachedCheck := utils.WrapCheck(cache, expensiveCheck)
//	result := cachedCheck(tx)
//
// This will cache the result of expensiveCheck for each transaction, reducing redundant computation.
func WrapCheck(cache *TransactionCheckCache, predicate TransactionCheckFunc) TransactionCheckFunc {
	cw := checkerWrapper{
		predicate: predicate,
		cache:     cache,
	}
	return cw.check
}

type checkerWrapper struct {
	predicate TransactionCheckFunc
	cache     *TransactionCheckCache
}

// Check executes the check for the given transaction, using the cache to avoid repeated expensive checks.
func (cw *checkerWrapper) check(argument *types.Transaction) bool {
	const (
		initialValidity = 200 * time.Millisecond
		maxValidity     = 15 * time.Second
		scalingFactor   = 2
	)
	now := time.Now()

	hash := argument.Hash()
	entry, found := cw.cache.get(hash)

	// If the last result is still valid, it can be reused.
	if found && entry.validUntil.After(now) {
		// Cache hit, return the cached result
		return entry.value
	}

	// The entry should be refreshed.
	entry.value = cw.predicate(argument)

	// Exponential backoff of the next check time.
	entry.validityDuration = max(min(maxValidity, entry.validityDuration*scalingFactor), initialValidity)
	entry.validUntil = now.Add(entry.validityDuration)
	cw.cache.put(hash, entry)

	return entry.value
}

// checkerEntry is a single entry in the CheckerCache.
type checkerEntry struct {
	validUntil       time.Time
	validityDuration time.Duration
	value            bool
}
