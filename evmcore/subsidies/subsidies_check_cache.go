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

package coresubsidies

import (
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
)

// SubsidiesCheckerCache is a cache for subsidiesChecker results. Fetch results
// are cached internally, using an exponential backoff strategy to reduce load
// on the underlying checker.
type SubsidiesCheckerCache struct {
	cache *lru.Cache
}

// NewSubsidiesCheckerCache creates a new subsidiesCheckerCache with roughly
// the given size in bytes. If size is less than or equal to zero, a default
// size is of 10MiB is used.
func NewSubsidiesCheckerCache(size int) *SubsidiesCheckerCache {
	if size <= 0 {
		size = 10 * 1024 * 1024 // 10 MiB
	}
	capacity := max(size/getSizeOfSubsidiesCheckerCacheEntry(), 1)
	cache, _ := lru.New(capacity) // only fails if capacity <= 0
	return &SubsidiesCheckerCache{cache: cache}
}

func (c *SubsidiesCheckerCache) get(txHash common.Hash) (subsidiesCheckerCacheEntry, bool) {
	if entry, ok := c.cache.Get(txHash); ok {
		return entry.(subsidiesCheckerCacheEntry), true
	}
	return subsidiesCheckerCacheEntry{}, false
}

func (c *SubsidiesCheckerCache) put(txHash common.Hash, entry subsidiesCheckerCacheEntry) {
	c.cache.Add(txHash, entry)
}

func (c *SubsidiesCheckerCache) Wrap(checker SubsidiesChecker) *cachedSubsidiesChecker {
	return &cachedSubsidiesChecker{
		cache:   c,
		checker: checker,
	}
}

// subsidiesCheckerCacheEntry is a single entry in the subsidiesCheckerCache.
type subsidiesCheckerCacheEntry struct {
	validUntil       time.Time
	validityDuration time.Duration
	covered          bool
}

func getSizeOfSubsidiesCheckerCacheEntry() int {
	var entry subsidiesCheckerCacheEntry
	return int(unsafe.Sizeof(entry))
}

// cachedSubsidiesChecker is a subsidiesChecker that caches results using a
// subsidiesCheckerCache.
type cachedSubsidiesChecker struct {
	cache   *SubsidiesCheckerCache
	checker SubsidiesChecker
}

// isSponsored checks if the given transaction is sponsored, using the cache
// to reduce load on the underlying checker. Cache entries have a validity
// duration that is exponentially increased on each check, up to a maximum.
func (c *cachedSubsidiesChecker) IsSponsored(tx *types.Transaction) bool {
	return c._isSponsored(tx, time.Now())
}

// _isSponsored is the internal implementation of isSponsored, allowing to
// specify the current time (for testing).
func (c *cachedSubsidiesChecker) _isSponsored(
	tx *types.Transaction,
	now time.Time,
) bool {
	const (
		initialValidity = 200 * time.Millisecond
		maxValidity     = 15 * time.Second
		scalingFactor   = 2
	)

	hash := tx.Hash()
	entry, found := c.cache.get(hash)

	// If the last result is still valid, it can be reused.
	if found && entry.validUntil.After(now) {
		// Cache hit, return the cached result
		return entry.covered
	}

	// The coverage should be refreshed.
	entry.covered = c.checker.IsSponsored(tx)

	// Exponential backoff of the next check time.
	entry.validityDuration = max(min(maxValidity, entry.validityDuration*scalingFactor), initialValidity)
	entry.validUntil = now.Add(entry.validityDuration)
	c.cache.put(hash, entry)

	return entry.covered
}
