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

package gas_subsidies

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t *testing.T) {
	require := require.New(t)
	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		// Brio is commented out until the gas cap is properly handled for internal transactions.
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
	}
	singleProposerOption := map[string]bool{
		"singleProposer": true,
		"distributed":    false,
	}

	for _, test := range upgrades {
		for mode, enabled := range singleProposerOption {
			t.Run(fmt.Sprintf("%s/%v", test.name, mode), func(t *testing.T) {

				test.upgrade.GasSubsidies = true
				test.upgrade.SingleProposerBlockFormation = enabled
				net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
					Upgrades: &test.upgrade,
				})

				sponsee := tests.NewAccount()
				receiverAddress := tests.NewAccount().Address()

				// Before the sponsorship is set up, a transaction from the sponsee
				// to the receiver should fail due to lack of funds.

				tx := &types.LegacyTx{
					To:  &receiverAddress,
					Gas: 21000,
				}
				sponsoredTx := makeSponsorRequestTransaction(t, tx, net.GetChainId(), sponsee)

				client, err := net.GetClient()
				require.NoError(err)
				defer client.Close()

				require.Error(client.SendTransaction(t.Context(), sponsoredTx),
					"should be rejected due to lack of funds and no sponsorship",
				)

				// --- deposit sponsorship funds ---

				donation := big.NewInt(1e16)
				registry := Fund(t, net, sponsee.Address(), donation)

				burnedBefore, err := client.BalanceAt(t.Context(), common.Address{}, nil)
				require.NoError(err)

				// --- submit a sponsored transaction ---

				receipt, err := net.Run(sponsoredTx)
				require.NoError(err)
				validateSponsoredTxInBlock(t, net, receipt.TxHash)

				// --- check that the sponsorship funds got deducted ---
				ok, fundId, err := registry.AccountSponsorshipFundId(nil, sponsee.Address())
				require.NoError(err)
				require.True(ok, "registry should have a fund ID")

				sponsorship, err := registry.Sponsorships(nil, fundId)
				require.NoError(err)
				require.Less(sponsorship.Funds.Uint64(), donation.Uint64())

				// the difference in the sponsorship funds should have been burned
				burnedAfter, err := client.BalanceAt(t.Context(), common.Address{}, nil)
				require.NoError(err)
				require.Greater(burnedAfter.Uint64(), burnedBefore.Uint64())

				// the sponsorship difference and the increase in burned funds should be equal
				diff := new(big.Int).Sub(burnedAfter, burnedBefore)
				reduced := new(big.Int).Sub(donation, sponsorship.Funds)
				require.Equal(diff.Uint64(), reduced.Uint64(),
					"the burned amount should equal the reduction of the sponsorship funds",
				)

			})
		}
	}
}
