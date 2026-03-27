package rpctest

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var _ ethapi.Backend = &backend{}

func Test_ChainID(t *testing.T) {

	be := NewBackendBuilder().Build()
	require.EqualValues(t, 1, be.ChainID().Uint64())

	for _, v := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder().WithChainId(v).Build()
		require.EqualValues(t, v, be.ChainID().Uint64())
	}
}

func Test_BlockHistory(t *testing.T) {
	be := NewBackendBuilder().Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())

	for _, v := range []uint64{1, 2, 3} {

		blocks := make([]TestBlock, v)
		for i := uint64(0); i < v; i++ {
			blocks[i] = TestBlock{Number: i + 1}
		}

		be := NewBackendBuilder().WithBlockHistory(blocks).Build()
		require.EqualValues(t, v, be.CurrentBlock().NumberU64())
	}
}

func Test_WithPool(t *testing.T) {

	be := NewBackendBuilder().Build()
	err := be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.ErrorContains(t, err, "tx pool not initialized")

	ctrl := gomock.NewController(t)
	mockPool := NewMocktxPool(ctrl)

	be = NewBackendBuilder().WithPool(mockPool).Build()
	mockPool.EXPECT().AddLocal(gomock.Any()).Return(nil).Times(1)

	err = be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.NoError(t, err)
}

func Test_WithAccount(t *testing.T) {

	var (
		addr1 = common.HexToAddress("0x01")
		addr2 = common.HexToAddress("0x02")
	)

	be := NewBackendBuilder().Build()
	latest := rpc.BlockNumber(1)
	state, block, err := be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	zero := state.GetBalance(addr1)
	require.Zero(t, zero.Sign(), "expected zero balance")

	be = NewBackendBuilder().
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
