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

package priority

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPriority_PriorityCanBeEnabledAndDisabled verifies that the
// TransactionPriorities feature can be toggled on and off via a network-rules
// update.
// Analog to TestGasSubsidies_CanBeEnabledAndDisabled.
func TestPriority_PriorityCanBeEnabledAndDisabled(t *testing.T) {
	require := require.New(t)

	type upgrades struct{ TransactionPriorities bool }
	type rulesDiff struct{ Upgrades upgrades }

	for name, upgrade := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			upgrade.TransactionPriorities = true
			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrade,
			})

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			prio := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
			setPrioritized(t, net, prio.Address(), 1, 1, common.Hash{0xaa})

			// --- TransactionPriorities enabled ---
			rules := tests.GetNetworkRules(t, net)
			require.True(rules.Upgrades.TransactionPriorities,
				"TransactionPriorities should be enabled after the update")

			requirePriorityHasEffect(t, net, prio, true)

			// --- TransactionPriorities disabled ---
			tests.UpdateNetworkRules(t, net, rulesDiff{
				Upgrades: upgrades{TransactionPriorities: false},
			})
			net.AdvanceEpoch(t, 1)

			rules = tests.GetNetworkRules(t, net)
			require.False(rules.Upgrades.TransactionPriorities,
				"TransactionPriorities should be disabled by default")

			requirePriorityHasEffect(t, net, prio, false)

			// --- TransactionPriorities enabled ---
			tests.UpdateNetworkRules(t, net, rulesDiff{
				Upgrades: upgrades{TransactionPriorities: true},
			})
			net.AdvanceEpoch(t, 1)

			rules = tests.GetNetworkRules(t, net)
			require.True(rules.Upgrades.TransactionPriorities,
				"TransactionPriorities should be enabled after the update")

			requirePriorityHasEffect(t, net, prio, true)
		})
	}
}

// TestPriority_CallingRegistryBeforeDeploy_TransactionIsNotPrioritized verifies
// that when the priority registry has not been deployed (i.e. the network is
// started without TransactionPriorities in the genesis Upgrades), the registry
// address holds no code and read calls against it fail. Consequently no
// transaction can be classified as prioritized, regardless of whether the
// TransactionPriorities flag is later toggled on via a rules update.
func TestPriority_CallingRegistryBeforeDeploy_TransactionIsNotPrioritized(t *testing.T) {
	for name, upgrade := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrade,
			})

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			// No code should be deployed at the priority registry address,
			// because TransactionPriorities was not set in the genesis Upgrades.
			code, err := client.CodeAt(t.Context(), registry.GetAddress(), nil)
			require.NoError(err)
			require.Empty(code, "priority registry must not be deployed")

			// Any read against the (non-existent) registry contract must fail.
			reg, err := registry.NewRegistry(registry.GetAddress(), client)
			require.NoError(err)
			_, err = reg.GetPriorityConfig(&bind.CallOpts{})
			require.Error(err, "reading from a non-deployed registry must fail")
		})
	}
}

// requirePriorityHasEffect submits a mixed burst where `prio`'s transactions
// are appended last, and asserts on the resulting ordering:
//   - expectPrioritized == true: within every block that mixes `prio` and
//     other senders, `prio`'s transactions must appear before the others
//     (prefix).
//   - expectPrioritized == false: at least one mixed block must place at
//     least one non-`prio` user transaction before a `prio` transaction
//     (proof that no reordering took place).
//
// `prio`'s current pending nonce is read from the network so the helper can
// be called multiple times back-to-back.
// TODO
func requirePriorityHasEffect(
	t *testing.T,
	net *tests.IntegrationTestNet,
	prio *tests.Account,
	expectPrioritized bool,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	prioNonce, err := client.PendingNonceAt(t.Context(), prio.Address())
	require.NoError(err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	sink := common.Address{0x99}

	// Ordinary background traffic from fresh, unregistered senders.
	burst := types.Transactions(buildOrdinaryTraffic(t, net, signer, 3, 3))

	// Append 3 txs from the priority sender at its current pending nonce.
	for i := uint64(0); i < 3; i++ {
		burst = append(burst, types.MustSignNewTx(prio.PrivateKey, signer, &types.LegacyTx{
			Nonce:    prioNonce + i,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      21000,
			GasPrice: gasPrice,
		}))
	}

	firstBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	hashes, err := net.SendAll(burst)
	require.NoError(err)
	waitForReceipts(t, net, hashes)

	prioAddr := prio.Address()
	requirePriorityAppliedSince(t, net, signer, firstBlock, expectPrioritized,
		func(a common.Address) bool { return a == prioAddr })
}
