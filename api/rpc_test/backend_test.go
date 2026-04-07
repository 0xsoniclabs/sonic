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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_NewBackendBuilder_CanSetChainID(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, opera.FakeNetworkID, be.ChainID().Uint64())

	for _, v := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder(t).WithChainID(v).Build()
		require.EqualValues(t, v, be.ChainID().Uint64())
	}
}

func Test_NewBackendBuilder_CanSetBlockHistory(t *testing.T) {
	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())
	be = NewBackendBuilder(t).WithBlockHistory(nil).Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())

	for _, v := range []uint64{1, 2, 3} {

		blocks := make([]Block, v)
		for i := uint64(0); i < v; i++ {
			blocks[i] = Block{Number: i + 1}
		}

		be := NewBackendBuilder(t).WithBlockHistory(blocks).Build()
		require.EqualValues(t, v, be.CurrentBlock().NumberU64())
	}
}

func Test_NewBackendBuilder_CanSetTxPool(t *testing.T) {
	be := NewBackendBuilder(t).Build()
	require.Panics(t, func() {
		_ = be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	})

	ctrl := gomock.NewController(t)
	mockPool := NewMockTxPool(ctrl)

	be = NewBackendBuilder(t).WithPool(mockPool).Build()
	mockPool.EXPECT().AddLocal(gomock.Any()).Return(nil).Times(1)

	err := be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.NoError(t, err)
}

func Test_NewBackendBuilder_CanSetInitialState(t *testing.T) {

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
		WithAccount(addr1, AccountState{Balance: big.NewInt(42)}).
		WithAccount(addr2, AccountState{Balance: big.NewInt(43)}).
		Build()
	state, block, err = be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	require.EqualValues(t, state.GetBalance(addr1).Uint64(), 42)
	require.EqualValues(t, state.GetBalance(addr2).Uint64(), 43)
}

func Test_NewBackendBuilder_CanSetUpgrade(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	require.True(t, be.rules.Upgrades.Brio)

	be = NewBackendBuilder(t).WithUpgrade(opera.GetSonicUpgrades()).Build()
	require.True(t, be.rules.Upgrades.Sonic)
	require.False(t, be.rules.Upgrades.Brio)
}

func Test_FakeBackend_ProducesCompatibleSigners(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	for _, chainId := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder(t).WithChainID(chainId).Build()

		signer := be.GetSigner()

		tx, err := types.SignTx(
			types.NewTransaction(1, common.Address{0x42}, big.NewInt(0), 21000, big.NewInt(1), nil),
			signer,
			key,
		)
		require.NoError(t, err)

		referenceSigner := types.LatestSignerForChainID(big.NewInt(int64(chainId)))
		recovered, err := referenceSigner.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), recovered)
	}
}

func Test_DefaultBlockHistory(t *testing.T) {
	blockHistory := defaultBlockHistory()
	require.EqualValues(t, 1, len(blockHistory))
	require.EqualValues(t, 1, blockHistory[0].Number)
	require.Equal(t, common.HexToHash("0x1"), blockHistory[0].Hash)
}
