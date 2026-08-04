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

package evmcore

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru/v2"
)

// PriorityCache memoizes the priority classification of transactions.
//
// Entries do not expire; they are evicted in least-recently-used order once the
// cache is full. Stale classifications are acceptable: a transaction that loses
// its priority keeps it here, which only works in its favor, and one that gains
// priority later is old by then, so promoting it no longer helps. Emission order
// is best-effort regardless and does not affect correctness.
//
// A nil cache treats everything as non-prioritized.
type PriorityCache struct {
	// entries maps a transaction hash to its priorities.Priority. The
	// underlying cache is safe for concurrent use.
	entries *lru.Cache[common.Hash, priorities.Priority]
}

// NewPriorityCache creates a cache holding as many priorities as the given
// configuration admits transactions to the pool. The configuration is sanitized
// like the pool sanitizes it, so both derive their capacity from the same values.
func NewPriorityCache(poolConfig TxPoolConfig) *PriorityCache {
	config := poolConfig.sanitize()
	capacity := config.GlobalSlots + config.GlobalQueue
	entries, _ := lru.New[common.Hash, priorities.Priority](int(capacity)) // sanitizing guarantees a positive capacity
	return &PriorityCache{entries: entries}
}

// GetOrClassify returns the cached priority of the given transaction, or
// classifies and caches it on a miss.
func (c *PriorityCache) GetOrClassify(
	tx *types.Transaction,
	classifier priorities.Classifier,
) priorities.Priority {
	if c == nil {
		return priorities.Priority{}
	}
	hash := tx.Hash()
	if entry, found := c.entries.Get(hash); found {
		return entry
	}
	priority, err := classifier.Priority(tx)
	if err != nil {
		priority = priorities.Priority{}
	}
	c.entries.Add(hash, priority)
	return priority
}
