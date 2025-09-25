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

package tests

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_SubsidyIsRejectedInCaseTheContractIsNotDeployed(t *testing.T) {
	subsidies := map[string]bool{
		"with gas subsidies":    true,
		"without gas subsidies": false,
	}

	for name, enabled := range subsidies {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetSonicUpgrades()
			upgrades.GasSubsidies = enabled

			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sponsor := tests.NewAccount()
			sponsee := tests.NewAccount()
			receiver := tests.NewAccount()
			receiverAddress := receiver.Address()

			donation := big.NewInt(1e16)
			registry, err := registry.NewRegistry(registry.GetAddress(), client)
			require.NoError(t, err)

			receipt, err := net.EndowAccount(sponsor.Address(), big.NewInt(1e18))
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
				opts.Value = donation
				return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
			})
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			// check that the sponsorship funds got deposited
			sponsorship, err := registry.UserSponsorships(nil, sponsee.Address(), receiver.Address())

			if enabled {
				require.NoError(t, err)
				require.Equal(t, donation, sponsorship.Funds)
			} else {
				require.Error(t, err)
			}

			// --- submit a sponsored transaction ---

			chainId := net.GetChainId()
			signer := types.LatestSignerForChainID(chainId)
			tx, err := types.SignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
				To:       &receiverAddress,
				Gas:      21000,
				GasPrice: big.NewInt(0),
			})
			require.NoError(t, err)

			err = client.SendTransaction(t.Context(), tx)
			if enabled {
				require.NoError(t, err, "the transaction should be accepted for processing because gas subsidies are enabled")
			} else {
				require.Error(t, err, "the transaction should be rejected because there are no gas subsidies")
				// If the transaction has not been sent, the receipt cannot be checked, the test ends here.
				return
			}

			// Wait for the sponsored transaction to be executed.
			receipt, err = net.GetReceipt(tx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
			require.NoError(t, err)
			require.True(t, slices.ContainsFunc(
				block.Transactions(),
				func(cur *types.Transaction) bool {
					return cur.Hash() == tx.Hash()
				},
			))

			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "the transaction should succeed because the contract is deployed")
		})
	}
}
