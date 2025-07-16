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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
)

// activeSessions holds the currently active integration test networks.
// In order to reduce the number of networks that need to be initialized
// for each test, we keep a map of active networks keyed by the hash of their
// Upgrade. If a test does not need to restart the a network, it should try to
// reuse an existing one.
var activeSessions map[common.Hash]*IntegrationTestNet

// TestMain is a functionality offered by the testing package that allows
// us to run some code before and after all tests in the package.
func TestMain(m *testing.M) {

	m.Run()
	fmt.Printf("Finished running tests")

	// Stop all active networks after tests are done
	for _, net := range activeSessions {
		fmt.Printf("Stopping network for upgrade: %v\n", net.options.Upgrades)
		net.Stop()
	}
	activeSessions = nil
}

// getSession creates a new session for network running on the
// given Upgrade. If there is no network running with this Upgrade, a new one
// will be initialized.
// If a tests can run in parallel, the call to t.Parallel() should be done
// after calling this function.
func getSession(t *testing.T, upgrades opera.Upgrades) IntegrationTestNetSession {
	if activeSessions == nil {
		activeSessions = make(map[common.Hash]*IntegrationTestNet)
	}

	net, ok := activeSessions[hashOptions(upgrades)]
	if ok {
		return net.SpawnSession(t)
	}
	t.Logf("Starting network for upgrade: %v", upgrades)
	myNet := StartIntegrationTestNet(t)
	activeSessions[hashOptions(upgrades)] = myNet
	return myNet.SpawnSession(t)
}

func hashOptions(upgrades opera.Upgrades) common.Hash {
	h := sha256.New()

	// serialize the options fields to json
	jsonData, err := json.Marshal(upgrades)
	if err != nil {
		fmt.Printf("failed to serialize options to json; %v\n", err)
		return common.Hash{}
	}
	h.Write(jsonData)
	return common.BytesToHash(h.Sum(nil))
}
