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

package transaction_properties

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/bundles"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// bundleScenarios are the three regimes a bundle can meet. Which one a network runs decides what an
// envelope even means, so it is a scenario rather than a draw.
var bundleScenarios = []struct {
	name     string
	upgrades func() opera.Upgrades
}{
	{"enabled", func() opera.Upgrades {
		upgrades := opera.GetBrioUpgrades()
		upgrades.TransactionBundles = true
		return upgrades
	}},
	{"disabled", opera.GetBrioUpgrades},
	{"beforeBrio", opera.GetAllegroUpgrades},
}

// TestTransactionProperties_BundlesExecuteWholeOrNotAtAll draws the same batches as the ordinary
// property test and puts one of them inside another: an envelope carrying a batch of its own, with an
// execution plan that tolerates no failure. What it adds over the hand-written tests in tests/bundles
// is the mixture -- a bundle whose contents carry every defect the generator draws, next to ordinary
// transactions in the same event -- and one property those tests cannot state: whatever the bundle was
// asked to do, either all of it reached the block or none of it did, and the accounts moved by exactly
// what the part that did reach it accounts for.
func TestTransactionProperties_BundlesExecuteWholeOrNotAtAll(t *testing.T) {
	for _, scenario := range bundleScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			upgrades := scenario.upgrades()
			require.False(t, upgrades.SingleProposerBlockFormation,
				"forced emission is unsupported in single proposer mode")

			network := core.StartNetwork(t, upgrades)
			session := network.Net.SpawnSession(t)

			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			// The batch draws from the first window of accounts and a bundle's contents from the
			// second, so the two nonce sequences never meet.
			batch := genConfig(network)
			claimed := batch
			claimed.MaxAccountsPerBatch = 2 * batch.MaxAccountsPerBatch

			domain, err := bundles.New(client, network,
				regular.Domain{Cfg: batch}, batch.MaxAccountsPerBatch)
			require.NoError(t, err)

			runner := &core.Runner{
				Session: session,
				Client:  client,
				Network: network,
				Cfg:     claimed,
				Domain:  domain,
			}
			runner.Init(t)

			rapid.Check(t, runner.Run)

			t.Logf("%s", domain)
			runner.Report(t, scenario.name, upgrades)
			runner.VerifyChainReplays(t)
		})
	}
}
