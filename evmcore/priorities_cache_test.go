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
	"errors"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewPriorityCache_HoldsTheSanitizedPoolCapacity(t *testing.T) {
	tests := map[string]struct {
		config   TxPoolConfig
		capacity int
	}{
		"configured": {
			config:   TxPoolConfig{GlobalSlots: 3, GlobalQueue: 2},
			capacity: 5,
		},
		// An unworkable capacity falls back to the same defaults the pool uses.
		"sanitized": {
			config:   TxPoolConfig{},
			capacity: int(DefaultTxPoolConfig.GlobalSlots + DefaultTxPoolConfig.GlobalQueue),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cache := NewPriorityCache(test.config)
			for i := range test.capacity + 1 {
				cache.entries.Add(common.Hash{byte(i), byte(i >> 8)}, priorities.Priority{})
			}
			require.Equal(t, test.capacity, cache.entries.Len())
		})
	}
}

func TestPriorityCache_GetOrClassify_PrefersTheCacheAndRetainsClassifications(t *testing.T) {
	tests := map[string]struct {
		cached      *priorities.Priority
		classified  priorities.Priority
		classifyErr error
		expected    priorities.Priority
	}{
		"cached": {
			cached:   new(priorities.Priority{Level: 1}),
			expected: priorities.Priority{Level: 1},
		},
		"not cached": {
			classified: priorities.Priority{Level: 1},
			expected:   priorities.Priority{Level: 1},
		},
		"failed classification": {
			classified:  priorities.Priority{Level: 1},
			classifyErr: errors.New("injected error"),
			expected:    priorities.Priority{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tx := types.NewTx(&types.LegacyTx{})
			classifier := priorities.NewMockClassifier(gomock.NewController(t))
			if tc.cached == nil {
				classifier.EXPECT().Priority(tx).Return(tc.classified, tc.classifyErr)
			}

			cache := NewPriorityCache(DefaultTxPoolConfig)
			if tc.cached != nil {
				cache.entries.Add(tx.Hash(), *tc.cached)
			}

			require.Equal(t, tc.expected, cache.GetOrClassify(tx, classifier))

			entry, found := cache.entries.Get(tx.Hash())
			require.True(t, found)
			require.Equal(t, tc.expected, entry)
		})
	}
}

func TestPriorityCache_NilCache_TreatsTransactionsAsNotPrioritized(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})
	classifier := priorities.NewMockClassifier(gomock.NewController(t))

	var cache *PriorityCache
	require.Equal(t, priorities.Priority{}, cache.GetOrClassify(tx, classifier))
}
