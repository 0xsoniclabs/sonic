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

package tests

import (
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNetworkPool_AcquireReturnsNilForUnknownKey(t *testing.T) {
	pool := &networkPool{entries: make(map[common.Hash]*sharedNetEntry)}
	require.Nil(t, pool.acquire(common.Hash{1}))
}

func TestNetworkPool_RegisterAndAcquire(t *testing.T) {
	pool := &networkPool{entries: make(map[common.Hash]*sharedNetEntry)}
	key := common.Hash{1}
	net := &IntegrationTestNet{}

	pool.register(key, net)
	require.Same(t, net, pool.acquire(key))

	pool.mu.Lock()
	require.Equal(t, 2, pool.entries[key].refCount)
	pool.mu.Unlock()
}

func TestNetworkPool_ReleaseDecrementsRefCount(t *testing.T) {
	pool := &networkPool{entries: make(map[common.Hash]*sharedNetEntry)}
	key := common.Hash{2}
	net := &IntegrationTestNet{
		nodes: make([]integrationTestNode, 1),
	}

	pool.register(key, net)
	_ = pool.acquire(key) // refCount = 2

	stopped := pool.release(key) // refCount = 1
	require.False(t, stopped)

	pool.mu.Lock()
	require.Equal(t, 1, pool.entries[key].refCount)
	pool.mu.Unlock()
}

func TestNetworkPool_ReleaseReturnsFalseForUnknownKey(t *testing.T) {
	pool := &networkPool{entries: make(map[common.Hash]*sharedNetEntry)}
	require.False(t, pool.release(common.Hash{99}))
}

func TestNetworkPool_StopAllEmptiesPool(t *testing.T) {
	pool := &networkPool{entries: make(map[common.Hash]*sharedNetEntry)}
	pool.entries[common.Hash{1}] = &sharedNetEntry{
		net:      &IntegrationTestNet{nodes: make([]integrationTestNode, 1)},
		refCount: 3,
	}
	pool.entries[common.Hash{2}] = &sharedNetEntry{
		net:      &IntegrationTestNet{nodes: make([]integrationTestNode, 1)},
		refCount: 1,
	}

	pool.stopAll()
	require.Empty(t, pool.entries)
}

func TestIsShareable(t *testing.T) {
	tests := []struct {
		name     string
		opts     IntegrationTestNetOptions
		expected bool
	}{
		{
			name:     "default options are not shareable",
			opts:     IntegrationTestNetOptions{},
			expected: false,
		},
		{
			name:     "shareable is shareable",
			opts:     IntegrationTestNetOptions{Shareable: true},
			expected: true,
		},
		{
			name: "ModifyConfig is not shareable",
			opts: IntegrationTestNetOptions{
				Shareable:    true,
				ModifyConfig: func(c *config.Config) {},
			},
			expected: false,
		},
		{
			name: "custom Accounts is not shareable",
			opts: IntegrationTestNetOptions{
				Shareable: true,
				Accounts:  []makefakegenesis.Account{{Name: "test"}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isShareable(tt.opts))
		})
	}
}

func TestHashOptions_DifferentConfigsProduceDifferentHashes(t *testing.T) {
	a := IntegrationTestNetOptions{NumNodes: 1}
	b := IntegrationTestNetOptions{NumNodes: 2}
	require.NotEqual(t, hashOptions(a), hashOptions(b))
}

func TestHashOptions_SameConfigsProduceSameHash(t *testing.T) {
	a := IntegrationTestNetOptions{NumNodes: 1, ValidatorsStake: []uint64{100}}
	b := IntegrationTestNetOptions{NumNodes: 1, ValidatorsStake: []uint64{100}}
	require.Equal(t, hashOptions(a), hashOptions(b))
}
