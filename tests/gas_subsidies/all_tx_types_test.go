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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_SupportAllTxTypes(t *testing.T) {
	transactions := map[string]types.TxData{
		"LegacyTx": &types.LegacyTx{
			Gas: 21000,
			To:  &common.Address{0x42},
		},
		"AccessListTx": &types.AccessListTx{
			Gas: 21000,
			To:  &common.Address{0x42},
		},
		"DynFeeTx": &types.DynamicFeeTx{
			Gas: 21000,
			To:  &common.Address{0x42},
		},
		"BlobTx": &types.BlobTx{
			Gas: 21000,
			To:  common.Address{0x42},
		},
		"SetCodeTx": &types.SetCodeTx{
			Gas:      21000 + 25000,
			To:       common.Address{0x42},
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	// Enable allegro to support setCode transactions.
	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	for name, tx := range transactions {
		t.Run(name, func(t *testing.T) {

			sponsee := tests.NewAccount()

			// The sponsorship donation needs to be high enough to cover the gas
			// fees of the sponsored tx and the fees of the internal payment tx.
			donation := big.NewInt(1e16)
			Fund(t, net, sponsee.Address(), donation)

			signedTx := makeSponsorRequestTransaction(t, tx, net.GetChainId(), sponsee)
			require.NoError(t, client.SendTransaction(t.Context(), signedTx))

			validateSponsoredTxInBlock(t, &net.Session, signedTx.Hash())
		})
	}
}
