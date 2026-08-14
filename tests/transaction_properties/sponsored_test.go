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
	"errors"
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
// The fund-backed mode runs on every fork, because it is the one whose coverage depends on money and
// because the structural rules a sponsored transaction still has to satisfy are fork-dependent. The
// network-sponsored modes run on Brio alone: once a request is covered they behave the same on every
// fork, and the fork-dependent half is already covered by the fund-backed runs.
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

				// Every fund must hold what the run paid it less what the follow-ups charged. The one
				// fork that cannot be asked is a fork that reproduced known defect 2: it leaves state
				// the blocks do not account for, and the archive a call reads through is that state
				// re-derived from blocks, so it stops advancing and no fund can be read.
				switch err := domain.CheckFunds(t.Context()); {
				case errors.Is(err, subsidies.ErrArchiveBehind) && !upgrades.Allegro:
					t.Logf("skipping the fund conservation check: %v", err)
				default:
					require.NoError(t, err)
				}

				t.Logf("%s", domain)
				runner.Report(t, name, upgrades)
				runner.VerifyChainReplays(t)
			})
		}
	}
}
