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

package bundles

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundles_BundlesCanBeEnabledAndDisabledStartingFromBrio(t *testing.T) {
	// Start with Brio but bundles initially disabled.
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
		},
	)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Helper to build a fresh bundle envelope using new funded accounts.
	buildEnvelope := func(t *testing.T) *types.Transaction {
		t.Helper()
		senderA := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		senderB := tests.NewAccount()
		addressB := senderB.Address()
		block, err := client.BlockNumber(t.Context())
		require.NoError(t, err)
		envelope, _, _ := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(block).
			AllOf(
				Step(t, net, senderA, &types.AccessListTx{
					To:    &addressB,
					Value: big.NewInt(1),
				}),
			).
			BuildEnvelopeBundleAndPlan()
		return envelope
	}

	// Verify that bundles are initially disabled.
	rules := tests.GetNetworkRules(t, net)
	require.True(t, rules.Upgrades.Brio, "Brio should be enabled")
	require.False(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be disabled by default with Brio")

	envelope := buildEnvelope(t)
	_, err = net.Send(envelope)
	require.ErrorContains(t, err, "bundled transactions are disabled")

	// Enable bundles via network rules update.
	type rulesType struct {
		Upgrades struct{ TransactionBundles bool }
	}
	tests.UpdateNetworkRules(t, net, rulesType{
		Upgrades: struct{ TransactionBundles bool }{TransactionBundles: true},
	})
	net.AdvanceEpoch(t, 1)

	rules = tests.GetNetworkRules(t, net)
	require.True(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be enabled after rules update")

	envelope = buildEnvelope(t)
	_, err = net.Send(envelope)
	require.NoError(t, err, "bundle envelopes should be accepted when bundles are enabled")

	// Disable bundles again via network rules update.
	tests.UpdateNetworkRules(t, net, rulesType{
		Upgrades: struct{ TransactionBundles bool }{TransactionBundles: false},
	})
	net.AdvanceEpoch(t, 1)

	rules = tests.GetNetworkRules(t, net)
	require.False(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be disabled after rules update")

	envelope = buildEnvelope(t)
	_, err = net.Send(envelope)
	require.ErrorContains(t, err, "bundled transactions are disabled")
}
