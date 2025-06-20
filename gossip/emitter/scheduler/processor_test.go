package scheduler

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEvmProcessorFactory_BeginBlock_CreatesProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	chain := NewMockChain(ctrl)

	chain.EXPECT().StateDB().Return(state.NewMockStateDB(ctrl))
	chain.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{})
	chain.EXPECT().GetEvmChainConfig(gomock.Any()).Return(&params.ChainConfig{})

	info := BlockInfo{}
	factory := &evmProcessorFactory{chain: chain}
	result := factory.beginBlock(info.toEvmBlock())
	require.NotNil(t, result)
}

func TestEvmProcessor_Run_IfExecutionSucceeds_ReportsSuccessAndGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	runner.EXPECT().Run(0, nil).Return(&types.Receipt{
		GasUsed: 10,
	}, false, nil)

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(nil)
	require.True(t, success)
	require.Equal(t, uint64(10), gasUsed)
}

func TestEvmProcessor_Run_IfExecutionFailed_ReportsAFailedExecution(t *testing.T) {

	t.Run("skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		runner.EXPECT().Run(0, nil).Return(nil, true, nil)
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(nil)
		require.False(t, success)
	})

	t.Run("failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		runner.EXPECT().Run(0, nil).Return(nil, false, fmt.Errorf("failed"))
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(nil)
		require.False(t, success)
	})

	t.Run("no receipt", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		runner.EXPECT().Run(0, nil).Return(nil, false, nil)
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(nil)
		require.False(t, success)
	})
}

func TestEvmProcessor_Release_ReleasesStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	processor := &evmProcessor{stateDb: stateDb}
	stateDb.EXPECT().Release()
	processor.release()
}
