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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/subsidies"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// sponsoredScenarios are the networks the sponsorship property is checked on. Which registry is
// installed decides who pays, and that is a property of the network rather than of a batch, so it is
// a scenario rather than a draw.
//
// The fund-backed mode runs on every fork from Allegro on, because it is the one whose coverage
// depends on money and because the structural rules a sponsored transaction still has to satisfy are
// fork-dependent. The network-sponsored modes run on Brio alone: once a request is covered they
// behave the same on every fork, and the fork-dependent half is already covered by the fund-backed
// runs. Why Sonic is left out is at the fork filter in the test itself.
var sponsoredScenarios = []struct {
	mode subsidies.Mode

	// onlyFork names the one fork this scenario runs on, or is empty to run on every fork.
	onlyFork string
}{
	{mode: subsidies.ModeFundBacked},
	{mode: subsidies.ModeNetwork, onlyFork: "Brio"},
	{mode: subsidies.ModeNetworkTracked, onlyFork: "Brio"},
}

// TestTransactionProperties_SponsoredTransactionsPreserveConsensusInvariants draws the same batches as
// the ordinary property test, turns most of them into sponsorship requests by taking their gas price
// away, and funds the sponsorships they ask for. What it adds over the hand-written tests in
// tests/gas_subsidies is the mixture: a sponsorship request next to an ordinary transaction from the
// same sender, sharing one nonce sequence, with every defect the generator draws still applied to it,
// against a fund that may or may not cover the batch.
func TestTransactionProperties_SponsoredTransactionsPreserveConsensusInvariants(t *testing.T) {
	for _, scenario := range sponsoredScenarios {
		for name, upgrades := range opera.GetAllHardForksInOrder() {
			if scenario.onlyFork != "" && scenario.onlyFork != name {
				continue
			}

			// Sonic is left out because a sponsored blob transaction carrying blob hashes produces a
			// block the archive can never ingest: the fork accepts blob transactions with hashes
			// (Allegro onwards refuses them), block formation prices the request at a gas price of
			// zero, and the archive -- which is state re-derived from blocks -- stops at that block
			// and never advances again while the chain runs on. Every state read after it then
			// answers for the last block the archive holds instead of the head, so a transaction
			// with a receipt looks like one that never touched its sender, and the accounting
			// oracle reports a nonce that did not follow its transactions. Subsidies are a network
			// rule rather than a fork feature, so no sponsorship rule is peculiar to Sonic; what
			// stopping here gives up is the pre-Allegro half of the structural rules a sponsored
			// transaction still has to satisfy, and those the ordinary property test above still
			// searches on Sonic.
			if !upgrades.Allegro {
				continue
			}

			t.Run(fmt.Sprintf("%v/%s", scenario.mode, name), func(t *testing.T) {
				require.False(t, upgrades.SingleProposerBlockFormation,
					"forced emission is unsupported in single proposer mode")

				upgrades.GasSubsidies = true
				network := core.StartNetwork(t, upgrades)
				subsidies.InstallRegistry(t, network.Net, scenario.mode)

				session := network.Net.SpawnSession(t)
				client, err := session.GetClient()
				require.NoError(t, err)
				defer client.Close()

				cfg := genConfig(network)
				domain, err := subsidies.New(session, client, network, regular.Domain{Cfg: cfg}, scenario.mode)
				require.NoError(t, err)

				runner := &core.Runner{
					Session: session,
					Client:  client,
					Network: network,
					Cfg:     cfg,
					Domain:  domain,
				}
				runner.Init(t)

				rapid.Check(t, runner.Run)

				// Every fund must hold what the run paid it less what the follow-ups charged. This
				// is asked unconditionally: the forks that could leave state the blocks do not
				// account for, and so stall the archive a fund is read through, are the pre-Allegro
				// ones this test no longer runs on.
				require.NoError(t, domain.CheckFunds(t.Context()))

				t.Logf("%s", domain)
				runner.Report(t, name)
				runner.VerifyChainReplays(t)
			})
		}
	}
}
