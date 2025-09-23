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

package gassubsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_CanBeEnabledAndDisabled(
	t *testing.T,
) {
	require := require.New(t)

	// The network is initially started using the distributed protocol.
	net := tests.StartIntegrationTestNet(t)
	// a sliced is used here to ensure the forks get updated in an acceptable order.
	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		// Brio is commented out until the gas cap is properly handled for internal transactions.
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
	}
	for _, test := range upgrades {
		t.Run(test.name, func(t *testing.T) {
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			// enforce the current upgrade
			testRules := tests.GetNetworkRules(t, net)
			testRules.Upgrades = test.upgrade
			tests.UpdateNetworkRules(t, net, testRules)
			// Advance the epoch by one to apply the change.
			tests.AdvanceEpochAndWaitForBlocks(t, net)

			// check original state
			type upgrades struct {
				GasSubsidies bool
			}
			type rulesType struct {
				Upgrades upgrades
			}

			var originalRules rulesType
			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(false, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be disabled initially")

			// Enable gas subsidies.
			rulesDiff := rulesType{
				Upgrades: upgrades{GasSubsidies: true},
			}
			tests.UpdateNetworkRules(t, net, rulesDiff)

			// Advance the epoch by one to apply the change.
			net.AdvanceEpoch(t, 1)

			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(true, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be enabled after the update")

			// Disable gas subsidies.
			rulesDiff = rulesType{
				Upgrades: upgrades{GasSubsidies: false},
			}
			tests.UpdateNetworkRules(t, net, rulesDiff)

			// Advance the epoch by one to apply the change.
			net.AdvanceEpoch(t, 1)

			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(false, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be disabled after the update")
		})
	}
}

func TestGasSubsidies_SubsidizedTransactionDeductsSubsidyFunds(t *testing.T) {
	require := require.New(t)

	// --- setup ---

	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			ModifyConfig: func(config *config.Config) {
				// The transaction to deploy the subsidies registry contract has
				// chain id 0, and is thus not replay protected. To be able to
				// submit it, we need to allow unprotected transactions in the
				// transaction pool.
				config.Opera.AllowUnprotectedTxs = true
			},
			Upgrades: &upgrades,
		})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	donation := big.NewInt(1e16)

	// set up sponsorship
	createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

	interanlNonce, err := client.PendingNonceAt(t.Context(), common.Address{})
	require.NoError(err)

	receipt := sendSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 0)

	txIndex, block := getTransactionIndexInBlock(t, client, receipt)
	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	// check that the payment transaction has the same nonce as the internal transaction
	payment := block.Transactions()[txIndex+1]
	require.False(payment.Protected()) // should be a nonce-signed transaction
	require.Equal(interanlNonce, payment.Nonce(),
		"the payment transaction should have the same nonce as the internal transaction",
	)

	receipt = sendSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 1)

	txIndex, block = getTransactionIndexInBlock(t, client, receipt)

	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	payment = block.Transactions()[txIndex+1]
	require.False(payment.Protected()) // should be a nonce-signed transaction
	require.Equal(interanlNonce+1, payment.Nonce(),
		"the payment transaction should have nonce incremented by 1",
	)
}

func TestGasSubsidies_InternalTransactionHasConsecutiveNonce(t *testing.T) {
	require := require.New(t)

	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()
	donation := big.NewInt(1e16)

	// set up sponsorship
	createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

	interanlNonce, err := client.PendingNonceAt(t.Context(), common.Address{})
	require.NoError(err)

	receipt := sendSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 0)

	txIndex, block := getTransactionIndexInBlock(t, client, receipt)
	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	// check that the payment transaction has the same nonce as the internal transaction
	payment := block.Transactions()[txIndex+1]
	require.False(payment.Protected()) // should be a nonce-signed transaction
	require.Equal(interanlNonce, payment.Nonce(),
		"the payment transaction should have the same nonce as the internal transaction",
	)

	receipt = sendSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 1)

	txIndex, block = getTransactionIndexInBlock(t, client, receipt)

	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	payment = block.Transactions()[txIndex+1]
	require.False(payment.Protected()) // should be a nonce-signed transaction
	require.Equal(interanlNonce+1, payment.Nonce(),
		"the payment transaction should have nonce incremented by 1",
	)
}
