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
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// newTxsChanSize is the size of the channel receiving the notifications about
// the transactions added to the pool.
const newTxsChanSize = 4096

// cachedPriority is a classified priority together with the time it was
// computed at, which bounds its validity.
type cachedPriority struct {
	priority   priorities.Priority
	computedAt time.Time
}

// expired reports whether the entry has outlived the given time-to-live.
func (e cachedPriority) expired(now time.Time, ttl time.Duration) bool {
	return now.Sub(e.computedAt) >= ttl
}

// priorityCache provides the priorities of the pending transactions to the
// emitter without classifying them on the emission path: each classification
// issues an EVM call against the head state, which is too expensive to perform
// for every candidate while an event is being built.
//
// Transactions are handed to observe — as they enter the pool, and again while
// they are pending — and classified by a background worker (see run) which owns
// the head state it reads. A priority becomes available to Priority once
// computed and stays valid for the configured time-to-live, after which it is
// re-computed against a more recent head state.
//
// Like the classification it replaces, a cached priority is a hint only and
// never affects consensus (see priorityHinter): reporting a not-yet-classified
// transaction as non-prioritized merely postpones its eager inclusion to a
// later event. A nil cache is inert — it classifies nothing and reports every
// transaction as non-prioritized.
type priorityCache struct {
	ttl     time.Duration
	refresh func(*priorityContext) *priorityContext // acquires the head state to classify against

	notify chan struct{} // signals the worker that transactions were observed

	mutex    sync.Mutex
	entries  map[common.Hash]cachedPriority
	pending  map[common.Hash]*types.Transaction // observed, awaiting classification
	config   priorities.Config                  // rate limits read along with the last classification
	revision uint64                             // number of classifications stored so far
}

// newPriorityCache creates a cache whose entries are valid for the given
// time-to-live and which classifies transactions against the context returned
// by refresh.
func newPriorityCache(ttl time.Duration, refresh func(*priorityContext) *priorityContext) *priorityCache {
	return &priorityCache{
		ttl:     ttl,
		refresh: refresh,
		notify:  make(chan struct{}, 1),
		entries: map[common.Hash]cachedPriority{},
		pending: map[common.Hash]*types.Transaction{},
	}
}

// Priority implements priorityLookup. It returns a zero-valued
// (non-prioritized) priority for a transaction that has not been classified
// yet, or whose entry has expired.
func (c *priorityCache) Priority(hash common.Hash) priorities.Priority {
	if c == nil {
		return priorities.Priority{}
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	entry, found := c.entries[hash]
	if !found || entry.expired(time.Now(), c.ttl) {
		return priorities.Priority{}
	}
	return entry.priority
}

// getConfig returns the rate-limit configuration read from the registry along
// with the most recent classification. It is zero-valued — prioritizing nothing
// — while no classification has been performed yet.
func (c *priorityCache) getConfig() priorities.Config {
	if c == nil {
		return priorities.Config{}
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.config
}

// getRevision returns a counter that changes whenever priorities have been
// added to the cache. A consumer that derived something from the cache can use
// it to tell whether its result is still up to date.
func (c *priorityCache) getRevision() uint64 {
	if c == nil {
		return 0
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.revision
}

// observe hands the given transactions to the cache and wakes the worker; the
// ones without a valid entry are classified asynchronously. Re-observing a
// transaction that already has one costs a lookup, so the pending pool can be
// re-observed at any time to keep its priorities from expiring.
func (c *priorityCache) observe(txs types.Transactions) {
	if c == nil || len(txs) == 0 {
		return
	}
	c.mutex.Lock()
	for _, tx := range txs {
		c.pending[tx.Hash()] = tx
	}
	c.mutex.Unlock()

	select {
	case c.notify <- struct{}{}:
	default: // the worker has not consumed the previous signal yet
	}
}

// run classifies the observed transactions until done is closed. It owns the
// priority context it classifies against, and thereby the head state, so no
// classification interferes with the event emission. The head state is only
// held while there are transactions to classify.
func (c *priorityCache) run(done <-chan struct{}) {
	if c == nil {
		return
	}
	var context *priorityContext
	defer func() { releasePriorityContext(context) }()
	for {
		select {
		case <-c.notify:
		case <-done:
			return
		}
		unclassified := c.takeUnclassified()
		if len(unclassified) == 0 {
			context = releasePriorityContext(context)
			continue
		}
		context = c.refresh(context)
		if context == nil {
			// Priorities are disabled or the head state is unavailable; the
			// transactions are classified once they are observed again.
			continue
		}
		c.store(classify(context.classifier, unclassified), context.config)
	}
}

// observeNewPoolTxs hands the transactions entering the pool to the priority
// cache until done is closed, so that they are classified before an event is
// built from them.
func (em *Emitter) observeNewPoolTxs(done <-chan struct{}) {
	newTxs := make(chan evmcore.NewTxsNotify, newTxsChanSize)
	subscription := em.world.TxPool.SubscribeNewTxsNotify(newTxs)
	defer subscription.Unsubscribe()
	for {
		select {
		case notification := <-newTxs:
			em.priorityCache.observe(notification.Txs)
		case <-subscription.Err():
			return
		case <-done:
			return
		}
	}
}

// releasePriorityContext releases the given context, if any, and returns nil.
func releasePriorityContext(context *priorityContext) *priorityContext {
	if context != nil {
		context.release()
	}
	return nil
}

// takeUnclassified consumes the observed transactions and returns the ones that
// have no valid entry.
func (c *priorityCache) takeUnclassified() []*types.Transaction {
	now := time.Now()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	observed := c.pending
	c.pending = map[common.Hash]*types.Transaction{}

	var unclassified []*types.Transaction
	for hash, tx := range observed {
		entry, found := c.entries[hash]
		if !found || entry.expired(now, c.ttl) {
			unclassified = append(unclassified, tx)
		}
	}
	return unclassified
}

// store records the given priorities and the configuration they were computed
// with, dropping the entries that have outlived the time-to-live.
func (c *priorityCache) store(classified map[common.Hash]priorities.Priority, config priorities.Config) {
	now := time.Now()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.config = config
	c.revision++
	for hash, entry := range c.entries {
		if entry.expired(now, c.ttl) {
			delete(c.entries, hash)
		}
	}
	for hash, priority := range classified {
		c.entries[hash] = cachedPriority{priority: priority, computedAt: now}
	}
}

// classify determines the priority of each transaction, treating a
// classification error as "not prioritized" like the consensus path does.
func classify(classifier priorities.Classifier, txs []*types.Transaction) map[common.Hash]priorities.Priority {
	classified := make(map[common.Hash]priorities.Priority, len(txs))
	for _, tx := range txs {
		priority, err := classifier.Priority(tx)
		if err != nil {
			priority = priorities.Priority{}
		}
		classified[tx.Hash()] = priority
	}
	return classified
}
