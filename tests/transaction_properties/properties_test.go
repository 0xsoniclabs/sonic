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
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/bundles"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/priorities"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/subsidies"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestTransactionProperties draws transactions, injects them past the pool and checks what the
// network did with them -- every workload this package has, on one chain.
//
// One chain rather than one per scenario is what makes the exported history worth keeping: it carries
// every fork and every feature in the order they were switched on, so anything else that needs a
// chain with all of them in it can be started from it. It also costs less than the fourteen networks
// it replaces.
//
// Each fork is a subtest, and each workload within it another. A workload names the rules it needs
// rather than merely switching them on, because the chain is shared and a flag left behind would
// change what the next workload's transactions mean.
func TestTransactionProperties(t *testing.T) {
	_, earliest := firstFork()
	network := core.StartNetwork(t, genesisUpgrades(earliest))

	// The chain is exported once everything has run, which is also what stops the network.
	defer network.ExportGenesis(t)

	for fork, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(fork, func(t *testing.T) {
			require.False(t, upgrades.SingleProposerBlockFormation,
				"forced emission is unsupported in single proposer mode")

			for _, load := range workloads {
				if !load.runsOn(fork) {
					continue
				}
				t.Run(load.name, func(t *testing.T) {
					network.Upgrade(t, load.rules(upgrades))
					load.run(t, network, fork)
				})
			}
		})
	}
}

// workload is one domain run against the chain: which forks it is worth running on, the rules it
// needs while it runs, and what it does.
type workload struct {
	name string

	// forks names the forks this workload runs on, or is empty to run on every one.
	forks []string

	// rules are what the network must be running for this workload to mean what it claims, built
	// from the fork's own upgrades.
	rules func(opera.Upgrades) opera.Upgrades

	run func(t *testing.T, network *core.Network, fork string)
}

func (w workload) runsOn(fork string) bool {
	if len(w.forks) == 0 {
		return true
	}
	for _, name := range w.forks {
		if name == fork {
			return true
		}
	}
	return false
}

// workloads are what the chain is put through, in this order at every fork that admits them: the
// features in the order they build on one another -- ordinary transactions, then sponsorship, then
// bundles, then priorities -- and each of them preceded by the inert case, where the transactions of
// the workload are drawn but the feature carrying them is switched off.
//
// The two network-sponsored workloads are the exception, and sit at the end rather than with the
// sponsorship they belong to. Installing one of their registries replaces the fund-backed one behind
// the registry proxy and nothing puts it back, so anything still needing a fund -- sponsored and
// prioritizedSponsored -- has to have run already. It costs nothing to run them last: a registry
// covering every request answers the same on every fork, and the fork-dependent half of sponsorship
// is what the fund-backed workload searches on each of them.
var workloads = []workload{
	{
		name:  "regular",
		rules: plain,
		run: func(t *testing.T, network *core.Network, fork string) {
			session, client := connect(t, network)
			cfg := genConfig(network)
			check(t, network, session, client, cfg, regular.Domain{Cfg: cfg}, fork)
		},
	},
	{
		// Sponsorship with the feature switched off: a batch mostly of requests, none of which
		// anything will cover, so every one of them has to die of the price it was drawn with. No
		// fund is paid into, and Sonic is not excluded the way the sponsored workload is -- what
		// stalls the archive there is a request the block formation prices at zero, and with the
		// feature off there is no such thing.
		name:  "sponsoredInert",
		rules: plain,
		run:   runSponsored(subsidies.ModeFundBacked),
	},
	{
		// Sonic is left out: a sponsored blob transaction carrying blob hashes produces a block the
		// archive can never ingest there -- block formation prices the request at zero gas, the
		// archive stops at that block, and every state read afterwards answers for the wrong one.
		// Allegro onwards refuses such transactions.
		name:  "sponsored",
		forks: []string{"Allegro", "Brio", "Canto"},
		rules: with(func(u *opera.Upgrades) { u.GasSubsidies = true }),
		run:   runSponsored(subsidies.ModeFundBacked),
	},
	{
		name:  "bundlesInert",
		forks: []string{"Brio", "Canto"},
		rules: plain,
		run:   runBundles,
	},
	{
		name:  "bundles",
		forks: []string{"Brio", "Canto"},
		rules: with(func(u *opera.Upgrades) { u.TransactionBundles = true }),
		run:   runBundles,
	},
	{
		name:  "prioritized",
		forks: []string{"Canto"},
		rules: with(func(u *opera.Upgrades) { u.TransactionPriorities = true }),
		run:   runPrioritized(false),
	},
	{
		name:  "prioritizedSponsored",
		forks: []string{"Canto"},
		rules: with(func(u *opera.Upgrades) {
			u.TransactionPriorities = true
			u.GasSubsidies = true
		}),
		run: runPrioritized(true),
	},
	{
		name:  "sponsoredByNetwork",
		forks: []string{"Canto"},
		rules: with(func(u *opera.Upgrades) { u.GasSubsidies = true }),
		run:   runSponsored(subsidies.ModeNetwork),
	},
	{
		name:  "sponsoredByNetworkTracked",
		forks: []string{"Canto"},
		rules: with(func(u *opera.Upgrades) { u.GasSubsidies = true }),
		run:   runSponsored(subsidies.ModeNetworkTracked),
	},
}

// plain is a fork's own rules and nothing else: every feature a workload can ask for switched off,
// including whatever the workload before it switched on.
func plain(upgrades opera.Upgrades) opera.Upgrades {
	upgrades.GasSubsidies = false
	upgrades.TransactionPriorities = false
	upgrades.TransactionBundles = false
	return upgrades
}

// with is plain plus the features one workload needs.
func with(enable func(*opera.Upgrades)) func(opera.Upgrades) opera.Upgrades {
	return func(upgrades opera.Upgrades) opera.Upgrades {
		upgrades = plain(upgrades)
		enable(&upgrades)
		return upgrades
	}
}

// genesisUpgrades are what the chain is started on: the earliest fork, with the two features whose
// registries reach the genesis only when they are enabled there. Both are switched off again by the
// first workload -- a flag can be moved either way later, a contract in the genesis cannot.
func genesisUpgrades(earliest opera.Upgrades) opera.Upgrades {
	earliest.GasSubsidies = true
	earliest.TransactionPriorities = true
	return earliest
}

// firstFork is the fork the chain starts on, taken from the same ordering the subtests follow so the
// two cannot disagree.
func firstFork() (string, opera.Upgrades) {
	for name, upgrades := range opera.GetAllHardForksInOrder() {
		return name, upgrades
	}
	panic("there are no hard forks to test")
}

// runBundles puts one of each batch's transactions inside an envelope carrying a batch of its own.
func runBundles(t *testing.T, network *core.Network, fork string) {
	session, client := connect(t, network)

	// The batch draws from the first window of accounts and a bundle's contents from the second, so
	// the two nonce sequences never meet.
	batch := genConfig(network)
	claimed := batch
	claimed.MaxAccountsPerBatch = 2 * batch.MaxAccountsPerBatch

	domain, err := bundles.New(client, network, regular.Domain{Cfg: batch}, batch.MaxAccountsPerBatch)
	require.NoError(t, err)

	check(t, network, session, client, claimed, domain, fork)
	t.Logf("%s", domain)
}

// runSponsored takes the gas price off part of each batch and has the given registry answer for who
// pays.
func runSponsored(mode subsidies.Mode) func(*testing.T, *core.Network, string) {
	return func(t *testing.T, network *core.Network, fork string) {
		subsidies.InstallRegistry(t, network.Net, mode)
		session, client := connect(t, network)

		cfg := genConfig(network)
		domain, err := subsidies.New(session, client, network, regular.Domain{Cfg: cfg}, mode)
		require.NoError(t, err)

		check(t, network, session, client, cfg, domain, fork)

		// Every fund must hold what this workload paid it, on top of whatever it found there, less
		// what the follow-ups charged.
		require.NoError(t, domain.CheckFunds(t.Context()))
		t.Logf("%s", domain)
	}
}

// runPrioritized registers a priority for every nonce slot each batch can reach, over ordinary
// transactions or over sponsored ones.
func runPrioritized(sponsored bool) func(*testing.T, *core.Network, string) {
	return func(t *testing.T, network *core.Network, fork string) {
		priorities.InstallRegistry(t, network.Net)
		session, client := connect(t, network)

		cfg := genConfig(network)
		var inner core.Domain = regular.Domain{Cfg: cfg}

		var funds *subsidies.Domain
		if sponsored {
			var err error
			funds, err = subsidies.New(session, client, network, inner, subsidies.ModeFundBacked)
			require.NoError(t, err)
			inner = funds
		}

		domain, err := priorities.New(session, client, network, inner, cfg)
		require.NoError(t, err)

		check(t, network, session, client, cfg, domain, fork)

		if funds != nil {
			require.NoError(t, funds.CheckFunds(t.Context()))
			t.Logf("%s", funds)
		}
		t.Logf("%s", domain)
	}
}

// check runs one domain against the chain for as many batches as rapid asks for, and reports what
// the run observed along with every known defect it reproduced or stepped around.
func check(
	t *testing.T,
	network *core.Network,
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	cfg core.GenConfig,
	domain core.Domain,
	fork string,
) {
	runner := &core.Runner{
		Session: session,
		Client:  client,
		Network: network,
		Cfg:     cfg,
		Domain:  domain,
	}
	runner.Init(t)

	rapid.Check(t, runner.Run)

	runner.Report(t, fork)
}

// connect spawns a session of its own for one workload, so what it sends never shares a nonce
// sequence with another's.
func connect(t *testing.T, network *core.Network) (
	tests.IntegrationTestNetSession, *tests.PooledEhtClient,
) {
	session := network.Net.SpawnSession(t)
	client, err := session.GetClient()
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return session, client
}
