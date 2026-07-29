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
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestNewPriorityCache_HoldsTwiceThePoolCapacity(t *testing.T) {
	cache := NewPriorityCache(evmcore.TxPoolConfig{GlobalSlots: 3, GlobalQueue: 2})
	for i := range 11 {
		cache.entries.Add(common.Hash{byte(i)}, priorities.Priority{})
	}
	require.Equal(t, 10, cache.entries.Len())
}

func TestPriorityCache_IsPrioritized_FollowsTheCachedPriority(t *testing.T) {
	tests := map[string]struct {
		priority   priorities.Priority
		classified bool
		want       bool
	}{
		"not classified": {},
		"ordinary":       {classified: true},
		"prioritized":    {priority: prioritized(1), classified: true, want: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			hash := common.Hash{1}
			cache := newTestPriorityCache()
			if test.classified {
				cache.entries.Add(hash, test.priority)
			}
			require.Equal(t, test.want, cache.IsPrioritized(hash))
		})
	}
}

func TestPriorityCache_Priorities_ClassifiesWhatIsNotCached(t *testing.T) {
	cached, classified := newCacheTx(1), newCacheTx(2)
	context := &priorityContext{classifier: fakeClassifier{
		byHash: map[common.Hash]priorities.Priority{classified.Hash(): prioritized(2)},
	}}

	cache := newTestPriorityCache()
	cache.entries.Add(cached.Hash(), prioritized(1))

	require.Equal(t, map[common.Hash]priorities.Priority{
		cached.Hash():     prioritized(1),
		classified.Hash(): prioritized(2),
	}, cache.Priorities(types.Transactions{cached, classified}, context))
}

func TestPriorityCache_Priorities_CachesTheClassification(t *testing.T) {
	tx := newCacheTx(1)
	context := &priorityContext{classifier: fakeClassifier{
		byHash: map[common.Hash]priorities.Priority{tx.Hash(): prioritized(1)},
	}}

	cache := newTestPriorityCache()
	cache.Priorities(types.Transactions{tx}, context)

	require.True(t, cache.IsPrioritized(tx.Hash()))
}

func TestPriorityCache_Priorities_WithoutAContextNothingIsPrioritized(t *testing.T) {
	tx := newCacheTx(1)
	cache := newTestPriorityCache()
	cache.entries.Add(tx.Hash(), prioritized(1))

	require.Empty(t, cache.Priorities(types.Transactions{tx}, nil))
}

func TestPriorityCache_NilCacheIsInert(t *testing.T) {
	var cache *PriorityCache
	require.False(t, cache.IsPrioritized(common.Hash{1}))
	require.Empty(t, cache.Priorities(types.Transactions{newCacheTx(1)}, &priorityContext{}))
}

func TestClassify_TreatsAClassificationErrorAsNotPrioritized(t *testing.T) {
	tx := newCacheTx(1)
	classifier := fakeClassifier{
		byHash:  map[common.Hash]priorities.Priority{tx.Hash(): prioritized(1)},
		failing: map[common.Hash]bool{tx.Hash(): true},
	}

	require.Equal(t, priorities.Priority{}, classify(classifier, tx))
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

// newTestPriorityCache builds a cache large enough to never evict in a test.
func newTestPriorityCache() *PriorityCache {
	return NewPriorityCache(evmcore.DefaultTxPoolConfig)
}

// newCacheTx builds a transaction distinguishable by its nonce.
func newCacheTx(nonce uint64) *types.Transaction {
	return types.NewTx(&types.LegacyTx{Nonce: nonce})
}
