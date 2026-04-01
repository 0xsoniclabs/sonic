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

package rpctest

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_ChainID(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, opera.FakeNetworkID, be.ChainID().Uint64())

	for _, v := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder(t).WithChainId(v).Build()
		require.EqualValues(t, v, be.ChainID().Uint64())
	}
}

func Test_BlockHistory(t *testing.T) {
	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())

	for _, v := range []uint64{1, 2, 3} {

		blocks := make([]TestBlock, v)
		for i := uint64(0); i < v; i++ {
			blocks[i] = TestBlock{Number: i + 1}
		}

		be := NewBackendBuilder(t).WithBlockHistory(blocks).Build()
		require.EqualValues(t, v, be.CurrentBlock().NumberU64())
	}
}

func Test_WithPool(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	err := be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.ErrorContains(t, err, "tx pool not initialized")

	ctrl := gomock.NewController(t)
	mockPool := NewMocktxPool(ctrl)

	be = NewBackendBuilder(t).WithPool(mockPool).Build()
	mockPool.EXPECT().AddLocal(gomock.Any()).Return(nil).Times(1)

	err = be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.NoError(t, err)
}

func Test_WithAccount(t *testing.T) {

	var (
		addr1 = common.HexToAddress("0x01")
		addr2 = common.HexToAddress("0x02")
	)

	be := NewBackendBuilder(t).Build()
	latest := rpc.BlockNumber(1)
	state, block, err := be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	zero := state.GetBalance(addr1)
	require.Zero(t, zero.Sign(), "expected zero balance")

	be = NewBackendBuilder(t).
		WithAccount(addr1, TestAccount{Balance: big.NewInt(42)}).
		WithAccount(addr2, TestAccount{Balance: big.NewInt(43)}).
		Build()
	state, block, err = be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	require.EqualValues(t, state.GetBalance(addr1).Uint64(), 42)
	require.EqualValues(t, state.GetBalance(addr2).Uint64(), 43)
}

func Test_WithUpgrade(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	require.True(t, be.rules.Upgrades.Brio)

	be = NewBackendBuilder(t).WithUpgrade(opera.GetSonicUpgrades()).Build()
	require.True(t, be.rules.Upgrades.Sonic)
	require.False(t, be.rules.Upgrades.Brio)
}
