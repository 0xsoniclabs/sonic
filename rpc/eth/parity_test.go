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

package eth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
)

// The module holds copies of a few small functions of the node, to stay free of
// its packages. The tests keep each copy equal to its original.

func TestIsSponsorshipRequest_MatchesTheNodesFunction(t *testing.T) {
	to := common.Address{1}
	for name, tx := range map[string]*types.Transaction{
		"nil":          nil,
		"free":         types.NewTx(&types.LegacyTx{To: &to, GasPrice: big.NewInt(0)}),
		"paid":         types.NewTx(&types.LegacyTx{To: &to, GasPrice: big.NewInt(1)}),
		"free create":  types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(0)}),
		"dynamic free": types.NewTx(&types.DynamicFeeTx{To: &to, GasFeeCap: big.NewInt(0)}),
		"dynamic paid": types.NewTx(&types.DynamicFeeTx{To: &to, GasFeeCap: big.NewInt(5)}),
	} {
		require.Equal(t, subsidies.IsSponsorshipRequest(tx), isSponsorshipRequest(tx), name)
	}
}

func TestMakeForkId_MatchesTheNodesFunction(t *testing.T) {
	genesis := common.HexToHash("0x1234")
	for _, u := range []opera.UpgradeHeight{
		opera.MakeUpgradeHeight(opera.GetSonicUpgrades(), 0),
		opera.MakeUpgradeHeight(opera.GetAllegroUpgrades(), 15),
		opera.MakeUpgradeHeight(opera.GetBrioUpgrades(), 1<<40),
	} {
		want, err := ethapi.MakeForkId(u, genesis)
		require.NoError(t, err)
		got, err := MakeForkId(u, genesis)
		require.NoError(t, err)
		require.Equal(t, [4]byte(want), [4]byte(got))
	}
}
