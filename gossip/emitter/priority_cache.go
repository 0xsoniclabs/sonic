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

package emitter

import (
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
)

// PriorityCache memoizes the priority classification of transactions. A single
// classification issues an EVM call against the head state, which is too
// expensive to repeat for every candidate whenever an event is built.
//
// The emitter calls Priorities, which classifies whatever is missing, so that a
// transaction is never ordered as ordinary merely because it has not been
// classified before.
//
// Entries do not expire; they are evicted in least-recently-used order once the
// cache is full. Since it holds twice as many entries as the pool admits
// transactions, a classification normally survives for as long as its
// transaction is pending, and a cached priority may thus be older than the
// current head state. That is admissible because it only affects the order in
// which this validator offers transactions to the DAG and never affects
// consensus: the authoritative priority ordering is re-derived during block
// formation.
//
// A nil cache is inert: it classifies nothing.
type PriorityCache struct {
	// entries maps a transaction hash to its priorities.Priority. The
	// underlying cache is safe for concurrent use.
	entries *lru.Cache
}

// NewPriorityCache creates a cache holding twice as many priorities as the given
// configuration admits transactions to the pool.
func NewPriorityCache(poolConfig evmcore.TxPoolConfig) *PriorityCache {
	capacity := 2 * (poolConfig.GlobalSlots + poolConfig.GlobalQueue)
	entries, _ := lru.New(max(int(capacity), 1)) // only fails for a non-positive capacity
	return &PriorityCache{entries: entries}
}

// Priorities returns the priority of each of the given transactions, classifying
// and caching the ones that are missing. Transactions are omitted from the
// result — and thereby treated as non-prioritized — if there is no context,
// meaning that priorities are disabled or the head state is unavailable.
func (c *PriorityCache) Priorities(
	txs types.Transactions,
	context *priorityContext,
) map[common.Hash]priorities.Priority {
	if c == nil || context == nil {
		return nil
	}
	resolved := make(map[common.Hash]priorities.Priority, len(txs))
	for _, tx := range txs {
		hash := tx.Hash()
		entry, found := c.entries.Get(hash)
		if !found {
			entry = classify(context.classifier, tx)
			c.entries.Add(hash, entry)
		}
		resolved[hash] = entry.(priorities.Priority)
	}
	return resolved
}

// classify determines the priority of the given transaction, treating a
// classification error as "not prioritized" like the consensus path does.
func classify(classifier priorities.Classifier, tx *types.Transaction) priorities.Priority {
	priority, err := classifier.Priority(tx)
	if err != nil {
		return priorities.Priority{}
	}
	return priority
}
