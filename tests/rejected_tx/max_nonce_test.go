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

package rejectedtx

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestAccount_RejectTransactions(t *testing.T) {
	nonceAccount := tests.NewAccount()
	codeAccount := tests.NewAccount()
	nonceAndCodeAccount := tests.NewAccount()

	accounts := map[string]tests.Account{
		"nonce":        *nonceAccount,
		"code":         *codeAccount,
		"nonceAndCode": *nonceAndCodeAccount,
	}

	genesisAccounts := []makefakegenesis.Account{
		{
			Address: nonceAccount.Address(),
			Balance: big.NewInt(1e18),
			Nonce:   math.MaxUint64,
		},
		{
			Address: codeAccount.Address(),
			Balance: big.NewInt(1e18),
			Code:    []byte{0x01},
		},
		{
			Address: nonceAndCodeAccount.Address(),
			Balance: big.NewInt(1e18),
			Nonce:   math.MaxUint64,
			Code:    []byte{0x01},
		},
	}

	for upgradeName, upgrades := range map[string]opera.Upgrades{
		"Sonic":   opera.GetSonicUpgrades(),
		"Allegro": opera.GetAllegroUpgrades(),
		"Brio":    opera.GetBrioUpgrades(),
	} {
		t.Run(upgradeName, func(t *testing.T) {
			t.Parallel()

			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: tests.AsPointer(upgrades),
				Accounts: genesisAccounts,
				NumNodes: 3,
			})

			for testName, account := range accounts {
				t.Run(testName, func(t *testing.T) {
					require := require.New(t)
					client, err := net.GetClient()
					require.NoError(err)
					defer client.Close()

					address := account.Address()
					nonce, err := client.PendingNonceAt(t.Context(), address)
					require.NoError(err)

					signer := types.LatestSignerForChainID(net.GetChainId())
					tx := types.MustSignNewTx(account.PrivateKey, signer, &types.LegacyTx{
						To:       &address,
						Value:    big.NewInt(1),
						Nonce:    nonce,
						Gas:      21000,
						GasPrice: big.NewInt(1e12),
					})

					_, err = net.Run(tx)
					require.Error(err, "transaction should be rejected")
					require.NotContains(err.Error(), "wait timeout", "transaction rejected for wrong reason")
				})
			}
		})
	}
}
