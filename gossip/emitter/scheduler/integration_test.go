package scheduler

import (
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This file contains a range of integration tests combining the scheduler
// implementation with the real-world implementations of the evaluator. While
// specifics are tested in individual unit tests, the focus of these tests it to
// verify the overall integration of the components.

func TestIntegration_NoTransactions_ProducesAnEmptySchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	chain := NewMockChain(ctrl)
	state := state.NewMockStateDB(ctrl)

	chain.EXPECT().GetEvmChainConfig().Return(&params.ChainConfig{})
	chain.EXPECT().StateDB().Return(state)
	state.EXPECT().Release()

	scheduler := NewScheduler(chain)
	require.Empty(t, scheduler.Schedule(
		t.Context(),
		&BlockInfo{},
		&fakeTxCollection{},
		Limits{
			Gas:  100_000_000,
			Size: 100_000,
		},
	))
}

func TestIntegration_OneTransactions_ProducesScheduleWithOneTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	chain := NewMockChain(ctrl)
	state := state.NewMockStateDB(ctrl)

	chain.EXPECT().GetEvmChainConfig().Return(&params.ChainConfig{})
	chain.EXPECT().StateDB().Return(state)

	// The scheduler configured for production is running transactions on the
	// actual EVM state processor. Thus, various StateDB interactions are
	// expected, yet the specific details are not important for this test. The
	// main objective is to make the one transaction to be scheduled pass the
	// execution to make it eligible to be included in the resulting schedule.
	any := gomock.Any()
	state.EXPECT().SetTxContext(any, any)
	state.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()
	state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	state.EXPECT().GetNonce(any).AnyTimes()
	state.EXPECT().SetNonce(any, any, any).AnyTimes()
	state.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	state.EXPECT().GetCode(any).Return(nil).AnyTimes()
	state.EXPECT().Snapshot().AnyTimes()
	state.EXPECT().Exist(any).Return(true).AnyTimes()
	state.EXPECT().GetRefund().AnyTimes()
	state.EXPECT().GetLogs(any, any).AnyTimes()
	state.EXPECT().EndTransaction().AnyTimes()
	state.EXPECT().TxIndex().AnyTimes()
	state.EXPECT().Release()

	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{
			To:  &common.Address{},
			Gas: 21_000,
		}),
	}

	schedule := NewScheduler(chain).Schedule(
		t.Context(),
		&BlockInfo{
			GasLimit: 100_000_000,
		},
		&fakeTxCollection{transactions: txs},
		Limits{
			Gas:  100_000_000,
			Size: 100_000,
		},
	)

	require.Equal(t, txs, schedule)
}
