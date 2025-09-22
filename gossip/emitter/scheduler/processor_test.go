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

package scheduler

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
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

	runner.EXPECT().Run(0, nil).Return([]evmcore.ProcessedTransaction{{
		Receipt: &types.Receipt{
			GasUsed: 10,
		},
	}}, nil)

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(nil)
	require.True(t, success)
	require.Equal(t, uint64(10), gasUsed)
}

func TestEvmProcessor_Run_IfExecutionProducesMultipleProcessedTransactions_SumsUpGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	runner.EXPECT().Run(0, nil).Return([]evmcore.ProcessedTransaction{
		{Receipt: &types.Receipt{GasUsed: 10}},
		{Receipt: &types.Receipt{GasUsed: 20}},
	}, nil)

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(nil)
	require.True(t, success)
	require.Equal(t, uint64(30), gasUsed)
}

func TestEvmProcessor_Run_IfRequestedTransactionIsNotExecuted_AFailedExecutionIsReported(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := &types.Transaction{}
	runner.EXPECT().Run(0, tx).Return([]evmcore.ProcessedTransaction{{
		Transaction: &types.Transaction{}, // different transaction
		Receipt:     &types.Receipt{GasUsed: 10},
	}}, nil)

	processor := &evmProcessor{processor: runner}
	success, _ := processor.run(tx)
	require.False(t, success)
}

func TestEvmProcessor_Run_IfExecutionFailed_ReportsAFailedExecution(t *testing.T) {

	t.Run("skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		runner.EXPECT().Run(0, nil).Return(nil, nil)
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(nil)
		require.False(t, success)
	})

	t.Run("failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		runner.EXPECT().Run(0, nil).Return(nil, fmt.Errorf("failed"))
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
