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
	lru "github.com/hashicorp/golang-lru"
)

// CheckFunc is the core type of the checker cache.
// It represents a function that takes an argument of type A and returns a result of type R.
// The Cache will then store the returned result, repeated calls to the checker
// for the same input will be cached for a certain duration to avoid expensive repeated checks.
type CheckFunc[A hasheable, R any] func(A) R

// CheckerCache is a cache for storing the results of expensive checks. It uses
// an LRU cache internally to store the results and evict old entries when the cache is full.
//
// Cached checks will have an associated validity duration, which is exponentially
// increased on each check until a maximum duration is reached.
type CheckerCache[R any] struct {
	cache *lru.Cache
}

// NewCheckerCache creates a new CheckerCache with the given size in bytes.
func NewCheckerCache[R any](size int) *CheckerCache[R] {
	if size <= 0 {
		size = 10 * 1024 * 1024 // 10 MiB
	}

	entrySize := reflect.TypeFor[checkerEntry[R]]().Size()
	capacity := max(size/(int(entrySize)), 1)
	cache, _ := lru.New(capacity) // only fails if capacity <= 0
	return &CheckerCache[R]{cache: cache}
}

func (c *CheckerCache[R]) get(txHash common.Hash) (checkerEntry[R], bool) {
	if entry, ok := c.cache.Get(txHash); ok {
		return entry.(checkerEntry[R]), true
	}
	return checkerEntry[R]{}, false
}

func (c *CheckerCache[R]) put(txHash common.Hash, entry checkerEntry[R]) {
	c.cache.Add(txHash, entry)
}

// WrapCheck wraps an expensive function with caching functionality. The returned checker will use
// the cache to store and retrieve results of checks.
func WrapCheck[A hasheable, R any](cache *CheckerCache[R], predicate CheckFunc[A, R]) CheckFunc[A, R] {
	cw := checkerWrapper[A, R]{
		predicate: predicate,
		cache:     cache,
	}
	return cw.check
}

type checkerWrapper[A hasheable, R any] struct {
	predicate CheckFunc[A, R]
	cache     *CheckerCache[R]
}

// Check executes the check for the given transaction, using the cache to avoid repeated expensive checks.
func (cw *checkerWrapper[A, R]) check(argument A) R {
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
type checkerEntry[T any] struct {
	validUntil       time.Time
	validityDuration time.Duration
	value            T
}

// hasheable is the interface required for the keys used in the CheckerCache.
// This helps to identify already cached results for the same input.
type hasheable interface {
	Hash() common.Hash
}
