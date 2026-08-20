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

package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestEip7997_DeterministicFactoryIsPresentFromCantoOn(t *testing.T) {
	for name, upgrades := range map[string]opera.Upgrades{
		"brio":  opera.GetBrioUpgrades(),
		"canto": opera.GetCantoUpgrades(),
	} {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, upgrades)
			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			code, err := client.CodeAt(t.Context(), params.DeterministicFactoryAddress, nil)
			require.NoError(t, err)

			if !upgrades.Canto {
				require.Empty(t, code)
				return
			}
			require.Equal(t, params.DeterministicFactoryCode, code)

			nonce, err := client.NonceAt(t.Context(), params.DeterministicFactoryAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(1), nonce)
		})
	}
}

func TestEip7997_DeterministicFactoryDeploysAtTheCreate2Address(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetCantoUpgrades())
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Init code returning a single STOP instruction as the runtime code.
	initCode := []byte{0x60, 0x01, 0x60, 0x00, 0xf3}
	salt := common.Hash{0x79, 0x97}
	want := crypto.CreateAddress2(
		params.DeterministicFactoryAddress,
		salt,
		crypto.Keccak256(initCode),
	)

	before, err := client.CodeAt(t.Context(), want, nil)
	require.NoError(t, err)
	require.Empty(t, before, "the target address must be free before the deployment")

	to := params.DeterministicFactoryAddress
	receipt, err := session.Run(CreateTransaction(t, session, &types.DynamicFeeTx{
		To:   &to,
		Data: append(salt.Bytes(), initCode...),
	}, session.GetSessionSponsor()))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	after, err := client.CodeAt(t.Context(), want, nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0x00}, after)
}

func TestEip7997_DeterministicFactoryIsInsertedWhenCantoIsActivated(t *testing.T) {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetBrioUpgrades()),
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	code, err := client.CodeAt(t.Context(), params.DeterministicFactoryAddress, nil)
	require.NoError(t, err)
	require.Empty(t, code, "the factory must not exist before Canto")

	beforeUpgrade, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	var rules opera.Rules
	require.NoError(t, client.Client().Call(&rules, "eth_getRules", "latest"))
	rules.Upgrades = opera.GetCantoUpgrades()
	UpdateNetworkRules(t, net, rules)
	net.AdvanceEpoch(t, 1)

	// The new rules take effect with the first block of the new epoch, which
	// is where the irregular state transition inserting the factory happens.
	receipt, err := net.EndowAccount(NewAccount().Address(), big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	code, err = client.CodeAt(t.Context(), params.DeterministicFactoryAddress, nil)
	require.NoError(t, err)
	require.Equal(t, params.DeterministicFactoryCode, code)

	// Blocks predating the activation must be unaffected.
	code, err = client.CodeAt(t.Context(), params.DeterministicFactoryAddress,
		new(big.Int).SetUint64(beforeUpgrade))
	require.NoError(t, err)
	require.Empty(t, code)
}
