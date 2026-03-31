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

//go:generate mockgen -source=checker_cache.go -destination=checker_cache_mock.go -package=utils

// CheckerCache is a cache for transaction checks, such as whether a transaction is subsidized.
// It stores the result of a check along with an expiration time, and prevents
// repeated execution of expensive checks for a transaction within a short time window.
//
// The cache is a lru cache that evicts the least recently used entries when it reaches its capacity.
type CheckerCache struct {
	cache *lru.Cache
}

// NewCheckerCache creates a new CheckerCache with the given size in bytes.
func NewCheckerCache(size int) *CheckerCache {
	if size <= 0 {
		size = 10 * 1024 * 1024 // 10 MiB
	}

	var entry checkerEntry
	entrySize := reflect.TypeOf(entry).Size()

	capacity := max(size/(int(entrySize)), 1)
	cache, _ := lru.New(capacity) // only fails if capacity <= 0
	return &CheckerCache{cache: cache}
}

func (c *CheckerCache) get(txHash common.Hash) (checkerEntry, bool) {
	if entry, ok := c.cache.Get(txHash); ok {
		return entry.(checkerEntry), true
	}
	return checkerEntry{}, false
}

func (c *CheckerCache) put(txHash common.Hash, entry checkerEntry) {
	c.cache.Add(txHash, entry)
}

// Wrap wraps a Checker with caching functionality. The returned checker will use
// the cache to store and retrieve results of checks.
func (c *CheckerCache) Wrap(adapter Checker) *checkerWrapper {
	return &checkerWrapper{
		predicate: adapter.Check,
		cache:     c,
	}
}

type Checker interface {
	Check(tx *types.Transaction) bool
}

type checkerWrapper struct {
	predicate CheckerFunc
	cache     *CheckerCache
}

// NewUnchachedChecker creates a new checkerWrapper that does not use caching.
// This method allows to adapt different checkers to the same interface, even
// if they do not support caching.
func NewUnchachedChecker(adapter CheckerFunc) *checkerWrapper {
	return &checkerWrapper{
		predicate: adapter,
		cache:     nil,
	}
}

type CheckerFunc func(tx *types.Transaction) bool

func (cw *checkerWrapper) Check(tx *types.Transaction) bool {
	return cw.check(tx, time.Now())
}

func (cw *checkerWrapper) check(tx *types.Transaction, now time.Time) bool {
	if cw.cache == nil {
		return cw.predicate(tx)
	}
	const (
		initialValidity = 200 * time.Millisecond
		maxValidity     = 15 * time.Second
		scalingFactor   = 2
	)

	hash := tx.Hash()
	entry, found := cw.cache.get(hash)

	// If the last result is still valid, it can be reused.
	if found && entry.validUntil.After(now) {
		// Cache hit, return the cached result
		return entry.value
	}

	// The coverage should be refreshed.
	entry.value = cw.predicate(tx)

	// Exponential backoff of the next check time.
	entry.validityDuration = max(min(maxValidity, entry.validityDuration*scalingFactor), initialValidity)
	entry.validUntil = now.Add(entry.validityDuration)
	cw.cache.put(hash, entry)

	return entry.value
}

// checkerEntry is a single entry in the checkerCache.
type checkerEntry struct {
	validUntil       time.Time
	validityDuration time.Duration
	value            bool
}
