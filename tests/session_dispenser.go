// Copyright 2025 Sonic Operations Ltd
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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
)

// activeTestNetInstances holds the currently active integration test networks.
// In order to reduce the number of networks that need to be initialized
// for each test, we keep a map of active networks keyed by the hash of their
// Upgrade.
var activeTestNetInstances map[common.Hash]*IntegrationTestNet

// TestMain is a functionality offered by the testing package that allows
// us to run some code before and after all tests in the package.
func TestMain(m *testing.M) {

	m.Run()

	// Stop all active networks after tests are done
	for _, net := range activeTestNetInstances {
		net.Stop()
		for i := range net.nodes {
			// it is safe to ignore this error since the tests have ended and
			// the directories are not needed anymore.
			_ = os.RemoveAll(net.nodes[i].directory)
		}
	}
	activeTestNetInstances = nil
}

// getSession creates a new session for network running on the
// given Upgrade. If there is no network running with this Upgrade, a new one
// will be initialized.
// If a tests can run in parallel, the call to t.Parallel() should be done
// after calling this function.
//
// A typical use case would look as follows:
//
//	t.Run("test_case", func(t *testing.T) {
//		session := getSession(t, opera.GetSonicUpgrades())
//		t.Parallel()
//		< use session instead of net of the rest of the test >
//	})
func getSession(t *testing.T, upgrades opera.Upgrades) IntegrationTestNetSession {
	if activeTestNetInstances == nil {
		activeTestNetInstances = make(map[common.Hash]*IntegrationTestNet)
	}

	net, ok := activeTestNetInstances[hashUpgrades(upgrades)]
	if ok {
		return net.SpawnSession(t)
	}

	t.Logf("Starting reusable test integration network for upgrade: %v", upgrades)
	myNet := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(upgrades),
		// Networks started by here will survive the test calling it, so they
		// will be stopped after all tests in the package have finished.
		SkipCleanUp: true,
	})
	activeTestNetInstances[hashUpgrades(upgrades)] = myNet
	return myNet.SpawnSession(t)
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
