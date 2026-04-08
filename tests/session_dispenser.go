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
	"os"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
)

// SharedNetwork holds a set of active integration test networks.
// In order to reduce the number of networks that need to be initialized
// for each testing package, we keep a map of active networks keyed by the hash
// of their Upgrade.
type SharedNetwork struct {
	activeTestNetInstances map[common.Hash]*IntegrationTestNet
	lock                   sync.Mutex
}

// NewSharedNetwork initializes a new SharedNetwork instance. It is expected to
// be used as a global variable per package, and its state must be cleaned up
// after the execution of all tests in the package.
//
// Usage:
// In the package where a shared network is needed, add a file named `main_test.go`
// with the following content:
//
//	var sharedNetwork *SharedNetwork
//	func TestMain(m *testing.M) {
//		sharedNetwork = NewSharedNetwork()
//		m.Run()
//		sharedNetwork.CleanUp()
//	}
func NewSharedNetwork() *SharedNetwork {
	return &SharedNetwork{
		activeTestNetInstances: make(map[common.Hash]*IntegrationTestNet),
	}
}

// GetIntegrationTestNetSession creates a new session for network running on the
// given Upgrade. If there is no network running with this Upgrade, a new one
// will be initialized.
//
// A typical use case would look as follows:
//
//	t.Run("test_case", func(t *testing.T) {
//		session := sharedNetwork.GetIntegrationTestNetSession(t, opera.GetSonicUpgrades())
//		< use session instead of net of the rest of the test >
//	})
//
// This function uses a global state that is cleaned up after the execution of
// the tests in `tests` package.
func (s *SharedNetwork) GetIntegrationTestNetSession(t *testing.T, upgrades opera.Upgrades) IntegrationTestNetSession {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.activeTestNetInstances == nil {
		s.activeTestNetInstances = make(map[common.Hash]*IntegrationTestNet)
	}

	net, ok := s.activeTestNetInstances[hashUpgrades(upgrades)]
	if ok {
		return net.SpawnSession(t)
	}

	myNet := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(upgrades),
		// Networks started by here will survive the test calling it, so they
		// will be stopped after all tests in the package have finished.
		SkipCleanUp: true,
	})
	s.activeTestNetInstances[hashUpgrades(upgrades)] = myNet
	return myNet.SpawnSession(t)
}

// CleanUp stops all active test networks and removes their data directories.
func (s *SharedNetwork) CleanUp() {
	for _, net := range s.activeTestNetInstances {
		net.Stop()
		for i := range net.nodes {
			// it is safe to ignore this error since the tests have ended and
			// the directories are not needed anymore.
			_ = os.RemoveAll(net.nodes[i].directory)
		}
	}
	s.activeTestNetInstances = nil
}

func hashUpgrades(upgrades opera.Upgrades) common.Hash {
	hash := sha256.New()

	// in the unlikely case of an error it is safe to ignore it since the
	// test would fail anyway
	jsonData, _ := json.Marshal(upgrades)
	// random write does not return error
	_, _ = hash.Write(jsonData)
	return common.BytesToHash(hash.Sum(nil))
}
