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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCachedPriority_Expired_HoldsUntilTheTimeToLiveElapsed(t *testing.T) {
	now := time.Unix(1_000, 0)
	tests := map[string]struct {
		age  time.Duration
		want bool
	}{
		"just computed":       {0, false},
		"just before expiry":  {time.Minute - 1, false},
		"time to live passed": {time.Minute, true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			entry := cachedPriority{computedAt: now.Add(-test.age)}
			require.Equal(t, test.want, entry.expired(now, time.Minute))
		})
	}
}

func TestPriorityCache_Priority_ReportsCachedPriorityUntilItExpires(t *testing.T) {
	tx := newCacheTx(1)
	cache := newPriorityCache(time.Minute, nil)

	cache.entries[tx.Hash()] = cachedPriority{priority: prioritized(1), computedAt: time.Now()}
	require.Equal(t, prioritized(1), cache.Priority(tx.Hash()))

	cache.entries[tx.Hash()] = cachedPriority{priority: prioritized(1), computedAt: time.Now().Add(-time.Minute)}
	require.Equal(t, priorities.Priority{}, cache.Priority(tx.Hash()))
}

func TestPriorityCache_Priority_UnclassifiedTransactionIsNotPrioritized(t *testing.T) {
	cache := newPriorityCache(time.Minute, nil)
	require.Equal(t, priorities.Priority{}, cache.Priority(common.Hash{1}))
}

func TestPriorityCache_GetConfig_ReportsConfigOfTheLastClassification(t *testing.T) {
	cache := newPriorityCache(time.Minute, nil)
	require.Equal(t, priorities.Config{}, cache.getConfig())

	config := priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 3}
	cache.store(nil, config)
	require.Equal(t, config, cache.getConfig())
}

func TestPriorityCache_GetRevision_ChangesWithEveryStoredClassification(t *testing.T) {
	cache := newPriorityCache(time.Minute, nil)
	before := cache.getRevision()

	cache.store(nil, priorities.Config{})

	require.NotEqual(t, before, cache.getRevision())
}

func TestPriorityCache_Observe_AccumulatesObservationsWithoutDuplicates(t *testing.T) {
	first, second := newCacheTx(1), newCacheTx(2)
	cache := newPriorityCache(time.Minute, nil)

	cache.observe(types.Transactions{first})
	cache.observe(types.Transactions{first, second})

	require.ElementsMatch(t, []*types.Transaction{first, second}, cache.takeUnclassified())
	require.Empty(t, cache.takeUnclassified())
}

func TestEmitter_ObserveNewPoolTxs_HandsPoolArrivalsToTheCache(t *testing.T) {
	tx := newCacheTx(1)
	var feed notify.Feed

	txPool := NewMockTxPool(gomock.NewController(t))
	txPool.EXPECT().SubscribeNewTxsNotify(gomock.Any()).DoAndReturn(
		func(ch chan<- evmcore.NewTxsNotify) notify.Subscription { return feed.Subscribe(ch) })

	em := &Emitter{world: World{TxPool: txPool}}
	em.priorityCache = newPriorityCache(time.Minute, nil)
	done := make(chan struct{})
	defer close(done)
	go em.observeNewPoolTxs(done)

	// The subscription is registered by the started goroutine.
	require.Eventually(t, func() bool {
		return feed.Send(evmcore.NewTxsNotify{Txs: []*types.Transaction{tx}}) > 0
	}, time.Second, time.Millisecond)

	require.Eventually(t, func() bool {
		return len(em.priorityCache.takeUnclassified()) == 1
	}, time.Second, time.Millisecond)
}

func TestPriorityCache_Run_ClassifiesObservedTransactions(t *testing.T) {
	tx := newCacheTx(1)
	context := &priorityContext{
		classifier: fakeClassifier{byHash: map[common.Hash]priorities.Priority{tx.Hash(): prioritized(1)}},
		config:     priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 2},
		release:    func() {},
	}
	cache := newPriorityCache(time.Minute, func(*priorityContext) *priorityContext { return context })
	done := make(chan struct{})
	defer close(done)
	go cache.run(done)

	cache.observe(types.Transactions{tx})

	require.Eventually(t, func() bool {
		return cache.Priority(tx.Hash()) == prioritized(1)
	}, time.Second, time.Millisecond)
	require.Equal(t, context.config, cache.getConfig())
}

func TestPriorityCache_Run_WithoutAPriorityContextNothingIsClassified(t *testing.T) {
	tx := newCacheTx(1)
	refreshed := make(chan struct{})
	cache := newPriorityCache(time.Minute, func(*priorityContext) *priorityContext {
		close(refreshed)
		return nil
	})
	done := make(chan struct{})
	defer close(done)
	go cache.run(done)

	cache.observe(types.Transactions{tx})

	<-refreshed
	require.Equal(t, priorities.Priority{}, cache.Priority(tx.Hash()))
}

func TestPriorityCache_Run_ReleasesThePriorityContextWhenStopped(t *testing.T) {
	tx := newCacheTx(1)
	var released atomic.Int32
	context := &priorityContext{
		classifier: fakeClassifier{},
		release:    func() { released.Add(1) },
	}
	acquired := make(chan struct{})
	cache := newPriorityCache(time.Minute, func(*priorityContext) *priorityContext {
		close(acquired)
		return context
	})
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		cache.run(done)
	}()

	cache.observe(types.Transactions{tx})

	<-acquired
	close(done)
	<-stopped
	require.Equal(t, int32(1), released.Load())
}

func TestPriorityCache_TakeUnclassified_SkipsTransactionsWithAValidEntry(t *testing.T) {
	classified, expired, unknown := newCacheTx(1), newCacheTx(2), newCacheTx(3)
	cache := newPriorityCache(time.Minute, nil)
	cache.entries[classified.Hash()] = cachedPriority{computedAt: time.Now()}
	cache.entries[expired.Hash()] = cachedPriority{computedAt: time.Now().Add(-time.Minute)}

	cache.observe(types.Transactions{classified, expired, unknown})

	require.Equal(t, []*types.Transaction{expired, unknown}, cache.takeUnclassified())
}

func TestPriorityCache_Store_DropsExpiredEntries(t *testing.T) {
	fresh, stale, added := newCacheTx(1), newCacheTx(2), newCacheTx(3)
	cache := newPriorityCache(time.Minute, nil)
	cache.entries[fresh.Hash()] = cachedPriority{computedAt: time.Now()}
	cache.entries[stale.Hash()] = cachedPriority{computedAt: time.Now().Add(-time.Minute)}

	cache.store(map[common.Hash]priorities.Priority{added.Hash(): prioritized(1)}, priorities.Config{})

	require.Contains(t, cache.entries, fresh.Hash())
	require.Contains(t, cache.entries, added.Hash())
	require.NotContains(t, cache.entries, stale.Hash())
}

func TestPriorityCache_NilCacheIsInert(t *testing.T) {
	var cache *priorityCache
	require.Equal(t, priorities.Priority{}, cache.Priority(common.Hash{1}))
	require.Equal(t, priorities.Config{}, cache.getConfig())
	cache.observe(types.Transactions{newCacheTx(1)})

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		cache.run(nil)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("the worker of a nil cache did not stop")
	}
}

func TestClassify_TreatsAClassificationErrorAsNotPrioritized(t *testing.T) {
	classified, failing := newCacheTx(1), newCacheTx(2)
	classifier := fakeClassifier{
		byHash: map[common.Hash]priorities.Priority{
			classified.Hash(): prioritized(1),
			failing.Hash():    prioritized(2),
		},
		failing: map[common.Hash]bool{failing.Hash(): true},
	}

	require.Equal(t, map[common.Hash]priorities.Priority{
		classified.Hash(): prioritized(1),
		failing.Hash():    {},
	}, classify(classifier, []*types.Transaction{classified, failing}))
}

// fakeClassifier classifies transactions by hash, reporting an error for the
// transactions marked as failing.
type fakeClassifier struct {
	byHash  map[common.Hash]priorities.Priority
	failing map[common.Hash]bool
}

func (c fakeClassifier) Priority(tx *types.Transaction) (priorities.Priority, error) {
	if c.failing[tx.Hash()] {
		return priorities.Priority{}, fmt.Errorf("injected classification error")
	}
	return c.byHash[tx.Hash()], nil
}

// newCacheTx builds a transaction distinguishable by its nonce.
func newCacheTx(nonce uint64) *types.Transaction {
	return types.NewTx(&types.LegacyTx{Nonce: nonce})
}
