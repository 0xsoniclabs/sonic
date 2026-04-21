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
	"crypto/sha256"
	"encoding/json"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
)

// sharedNetEntry is a reference-counted wrapper around an IntegrationTestNet.
// It tracks the number of active users so that the network is only stopped
// when the last reference is released.
type sharedNetEntry struct {
	net      *IntegrationTestNet
	refCount int
}

// networkPool manages a set of shared integration test networks keyed by their
// configuration hash. It provides reference-counted access so that multiple
// tests with identical configuration can reuse the same network, and the
// network is stopped only when the last reference is released.
type networkPool struct {
	mu      sync.Mutex
	entries map[common.Hash]*sharedNetEntry
}

// globalNetworkPool is the process-wide pool of shared integration test
// networks. It is used by StartIntegrationTestNet.
var globalNetworkPool = &networkPool{
	entries: make(map[common.Hash]*sharedNetEntry),
}

// acquire returns an existing network for the given key, incrementing its
// reference count, or nil if no such network exists.
func (p *networkPool) acquire(key common.Hash) *IntegrationTestNet {
	p.mu.Lock()
	defer p.mu.Unlock()
	if entry, ok := p.entries[key]; ok {
		entry.refCount++
		return entry.net
	}
	return nil
}

// register adds a newly created network to the pool with a reference count of 1.
func (p *networkPool) register(key common.Hash, net *IntegrationTestNet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries[key] = &sharedNetEntry{net: net, refCount: 1}
}

// release decrements the reference count for the given key. If the count
// reaches zero the network is stopped, removed from the pool, and true is
// returned. Otherwise false is returned and the network remains running.
func (p *networkPool) release(key common.Hash) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[key]
	if !ok {
		return false
	}
	entry.refCount--
	if entry.refCount <= 0 {
		delete(p.entries, key)
		entry.net.Stop()
		entry.net.removeDirectory()
		return true
	}
	return false
}

// stopAll stops every network in the pool regardless of reference counts
// and empties the pool. This is intended for use in TestMain teardown.
func (p *networkPool) stopAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, entry := range p.entries {
		entry.net.Stop()
		entry.net.removeDirectory()
		delete(p.entries, key)
	}
}

// hashOptions computes a deterministic hash over the shareable parts of
// IntegrationTestNetOptions. Options that contain non-comparable fields
// (ModifyConfig, Accounts) make the configuration non-shareable and are
// handled by the callers.
func hashOptions(opts IntegrationTestNetOptions) common.Hash {
	h := sha256.New()
	// Upgrades
	if opts.Upgrades != nil {
		data, _ := json.Marshal(*opts.Upgrades)
		h.Write(data)
	}
	// NumNodes and ValidatorsStake
	meta, _ := json.Marshal(struct {
		NumNodes        int
		ValidatorsStake []uint64
		ExtraArgs       []string
	}{
		NumNodes:        opts.NumNodes,
		ValidatorsStake: opts.ValidatorsStake,
		ExtraArgs:       opts.ClientExtraArguments,
	})
	h.Write(meta)
	return common.BytesToHash(h.Sum(nil))
}

// isShareable returns true if the given options describe a network that can
// safely be shared across tests. Networks must opt in to sharing via
// Shareable: true, and must not use ModifyConfig or custom Accounts.
func isShareable(opts IntegrationTestNetOptions) bool {
	if !opts.Shareable {
		return false
	}
	if opts.ModifyConfig != nil {
		return false
	}
	if len(opts.Accounts) > 0 {
		return false
	}
	return true
}

// getIntegrationTestNetSession creates a new session for network running on the
// given Upgrade. If there is no network running with this Upgrade, a new one
// will be initialized.
// If a tests can run in parallel, the call to t.Parallel() should be done
// after calling this function.
//
// A typical use case would look as follows:
//
//	t.Run("test_case", func(t *testing.T) {
//		session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
//		t.Parallel()
//		< use session instead of net of the rest of the test >
//	})
func getIntegrationTestNetSession(t *testing.T, upgrades opera.Upgrades) IntegrationTestNetSession {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Shareable: true,
		Upgrades:  AsPointer(upgrades),
	})
	return net.SpawnSession(t)
}

// // hashUpgrades computes a hash over the given upgrades for use as a map key.
// func hashUpgrades(upgrades opera.Upgrades) common.Hash {
// 	hash := sha256.New()
// 	jsonData, _ := json.Marshal(upgrades)
// 	_, _ = hash.Write(jsonData)
// 	return common.BytesToHash(hash.Sum(nil))
// }
