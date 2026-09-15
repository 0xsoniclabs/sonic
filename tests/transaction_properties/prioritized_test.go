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
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/priorities"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/subsidies"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// prioritizedScenarios are what is prioritized: ordinary transactions, and sponsored ones. Both are
// worth running, because priorities and subsidies meet in a block: a sponsorship request can be
// hoisted like anything else, and its follow-up transaction then lands inside the hoisted prefix.
var prioritizedScenarios = []struct {
	name      string
	sponsored bool
}{
	{name: "regular"},
	{name: "sponsored", sponsored: true},
}

// prioritizedFork is the one fork the priorities property runs on. Transaction priorities are a
// network rule rather than a fork feature -- the ordering step is gated on the flag alone -- so no
// ordering rule is peculiar to a fork, and the fork-dependent half of what a prioritized
// transaction still has to satisfy is what the two property tests above already search on every
// fork. The latest fork is the one where every transaction type and every rule is in play.
const prioritizedFork = "Brio"

// TestTransactionProperties_PrioritizedTransactionsAreOrderedAsTheRegistrySays draws the same
// batches as the ordinary property test, registers a priority for every nonce slot the batch can
// reach in a registry answering per (sender, nonce), and checks that every block the batch reached
// begins with exactly the transactions the ordering rules hoist, in exactly their order.
//
// What it adds over the hand-written tests in tests/priorities is the mixture: two transactions of
// one sender carrying different priorities in one nonce sequence, entities sharing a gas budget that
// may or may not cover the batch, every defect the generator draws still applied, and -- in the
// second scenario -- a sponsorship request among the hoisted ones.
func TestTransactionProperties_PrioritizedTransactionsAreOrderedAsTheRegistrySays(t *testing.T) {
	for _, scenario := range prioritizedScenarios {
		for name, upgrades := range opera.GetAllHardForksInOrder() {
			if name != prioritizedFork {
				continue
			}

			t.Run(fmt.Sprintf("%s/%s", scenario.name, name), func(t *testing.T) {
				require.False(t, upgrades.SingleProposerBlockFormation,
					"forced emission is unsupported in single proposer mode")

				upgrades.TransactionPriorities = true
				upgrades.GasSubsidies = scenario.sponsored
				network := core.StartNetwork(t, upgrades)
				priorities.InstallRegistry(t, network.Net)

				session := network.Net.SpawnSession(t)
				client, err := session.GetClient()
				require.NoError(t, err)
				defer client.Close()

				cfg := genConfig(network)
				var inner core.Domain = regular.Domain{Cfg: cfg}
				var funds *subsidies.Domain
				if scenario.sponsored {
					funds, err = subsidies.New(
						session, client, network, inner, subsidies.ModeFundBacked)
					require.NoError(t, err)
					inner = funds
				}

				domain, err := priorities.New(session, client, network, inner, cfg)
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

				if funds != nil {
					require.NoError(t, funds.CheckFunds(t.Context()))
					t.Logf("%s", funds)
				}
				t.Logf("%s", domain)
				runner.Report(t, name)
				runner.VerifyChainReplays(t)
				runner.ExportGenesis(t)
			})
		}
	}
}
