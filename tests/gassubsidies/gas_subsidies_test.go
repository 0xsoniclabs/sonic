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
	"testing"

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
