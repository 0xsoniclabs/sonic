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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_WorkWithAllTxTypes(t *testing.T) {
	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()
	receiverAddress := receiver.Address()

	transactions := map[string]types.TxData{
		"LegacyTx": &types.LegacyTx{
			Nonce:    0,
			GasPrice: nil,
			Gas:      21000,
			To:       &receiverAddress,
		},
		"AccessListTx": &types.AccessListTx{
			ChainID:    nil,
			Nonce:      0,
			GasPrice:   nil,
			Gas:        21000 + 2400,
			To:         &receiverAddress,
			AccessList: []types.AccessTuple{{}},
		},
		"DynFeeTx": &types.DynamicFeeTx{
			ChainID:   nil,
			Nonce:     0,
			GasFeeCap: nil,
			GasTipCap: nil,
			Gas:       21000,
			To:        &receiverAddress,
		},
		"BlobTx": &types.BlobTx{
			ChainID:    nil,
			Nonce:      0,
			GasFeeCap:  nil,
			GasTipCap:  nil,
			Gas:        21000,
			To:         receiverAddress,
			BlobFeeCap: nil,
		},
		"SetCodeTx": &types.SetCodeTx{
			ChainID:   nil,
			Nonce:     0,
			GasFeeCap: nil,
			GasTipCap: nil,
			Gas:       21000 + 25000,
			To:        receiverAddress,
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for name, tx := range transactions {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetAllegroUpgrades()
			upgrades.GasSubsidies = true

			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			donation := big.NewInt(1e16)
			registry := createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

			// --- send a subsidized transaction ---

			signer := types.LatestSignerForChainID(net.GetChainId())
			signedTx, err := types.SignNewTx(sponsee.PrivateKey, signer, tx)
			require.NoError(t, err)

			receipt := sendSponsoredTransaction(t, client, net, signedTx)

			// check that the sponsorship funds got deducted
			ops := &bind.CallOpts{
				BlockNumber: receipt.BlockNumber,
			}
			sponsorship, err := registry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
			require.NoError(t, err)
			require.Less(t, sponsorship.Funds.Uint64(), donation.Uint64())
		})
	}
}
