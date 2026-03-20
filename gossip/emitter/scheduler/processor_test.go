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

package scheduler

import (
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
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
	chain.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{}).AnyTimes()
	chain.EXPECT().GetEvmChainConfig(gomock.Any()).Return(&params.ChainConfig{})

	info := BlockInfo{}
	factory := &evmProcessorFactory{chain: chain}
	result := factory.beginBlock(info.toEvmBlock(), nil)
	require.NotNil(t, result)
}

func TestEvmProcessor_Run_IfExecutionSucceeds_ReportsSuccessAndGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(0, tx).Return(
		evmcore.ExecutionSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{
				{Transaction: tx, Receipt: &types.Receipt{GasUsed: 10}},
			},
		})

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(tx)
	require.True(t, success)
	require.Equal(t, uint64(10), gasUsed)
}

func TestEvmProcessor_Run_IfExecutionProducesMultipleProcessedTransactions_SkipsTransactionsWithoutReceipt(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(0, tx).Return(evmcore.ExecutionSummary{})

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(tx)
	require.False(t, success)
	require.Zero(t, gasUsed)
}

func TestEvmProcessor_Run_IfExecutionProducesMultipleProcessedTransactions_SumsUpGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(0, tx).Return(
		evmcore.ExecutionSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{
				{Transaction: tx, Receipt: &types.Receipt{GasUsed: 10}},
			{Receipt: nil}, // skipped transaction
				{Receipt: &types.Receipt{GasUsed: 20}},
			}})

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(tx)
	require.True(t, success)
	require.Equal(t, uint64(30), gasUsed)
}

func TestEvmProcessor_Run_IfRequestedTransactionIsNotExecuted_AFailedExecutionIsReported(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(0, tx).Return(
		evmcore.ExecutionSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{{
				Transaction: &types.Transaction{}, // different transaction
				Receipt:     &types.Receipt{GasUsed: 10},
			}}})

	processor := &evmProcessor{processor: runner}
	success, _ := processor.run(tx)
	require.False(t, success)
}

func TestEvmProcessor_Run_IfExecutionFailed_ReportsAFailedExecution(t *testing.T) {
	t.Run("not processed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		tx := types.NewTx(&types.LegacyTx{})
		runner.EXPECT().Run(0, tx)
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(tx)
		require.False(t, success)
	})

	t.Run("no receipt", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		tx := types.NewTx(&types.LegacyTx{})
		runner.EXPECT().Run(0, gomock.Any()).Return(evmcore.ExecutionSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: tx, Receipt: nil},
		}})
		processor := &evmProcessor{processor: runner}
		success, _ := processor.run(tx)
		require.False(t, success)
	})

	t.Run("different transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMockevmProcessorRunner(ctrl)
		txA := types.NewTx(&types.LegacyTx{})
		txB := types.NewTx(&types.LegacyTx{})
		runner.EXPECT().Run(0, gomock.Any()).Return(evmcore.ExecutionSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: txB, Receipt: &types.Receipt{GasUsed: 10}},
		}})
		processor := &evmProcessor{processor: runner}
		success, gasUsed := processor.run(txA)
		require.False(t, success)
		require.Equal(t, uint64(10), gasUsed)
	})
}

func TestEvmProcessor_Release_ReleasesStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	processor := &evmProcessor{stateDb: stateDb}
	stateDb.EXPECT().Release()
	processor.release()
}

func TestEvmProcessor_Run_IfBundleExecutionSucceeds_ReportsSuccessAndGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	bundleTracker := NewMockBundleTracker(ctrl)
	bundleTracker.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false)

	tx := bundle.NewBuilder().Build()
	runner.EXPECT().Run(0, tx).Return(
		evmcore.ExecutionSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{{
				Receipt: &types.Receipt{GasUsed: 10},
			}},
			ProcessedBundles: []evmcore.ProcessedBundle{{}},
		})

	processor := &evmProcessor{processor: runner, bundleTracker: bundleTracker}
	success, gasUsed := processor.run(tx)
	require.True(t, success)
	require.Equal(t, uint64(10), gasUsed)
}

func TestEvmProcessor_Run_IfExecutionProducesMultipleProcessedTransactions_FromABundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	bundleTracker := NewMockBundleTracker(ctrl)
	bundleTracker.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false)

	tx := bundle.NewBuilder().Build()
	singleTransactionGasUsed := uint64(10)
	expectedGasUsed := singleTransactionGasUsed * 2

	runner.EXPECT().Run(0, tx).Return(
		evmcore.ExecutionSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{
				{Receipt: &types.Receipt{GasUsed: singleTransactionGasUsed}},
				{Receipt: &types.Receipt{GasUsed: singleTransactionGasUsed}},
			},
			ProcessedBundles: []evmcore.ProcessedBundle{{}}})

	processor := &evmProcessor{processor: runner, bundleTracker: bundleTracker}
	success, gasUsed := processor.run(tx)
	require.True(t, success)
	require.Equal(t, expectedGasUsed, gasUsed)
}

func TestEvmProcessor_Run_IfBundleExecutionFailed_RejectsWhenFailedToGetBundlePlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{
		To: &bundle.BundleProcessor,
	})

	processor := &evmProcessor{processor: runner}
	success, gasUsed := processor.run(tx)
	require.False(t, success)
	require.Zero(t, gasUsed)
}

func TestEvmProcessor_Run_IfBundleExecutionFailed_RejectsWhenBundleHasBeenRecentlyProcessed(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	bundleTracker := NewMockBundleTracker(ctrl)
	bundleTracker.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(true)

	tx := bundle.NewBuilder().Build()

	processor := &evmProcessor{processor: runner, bundleTracker: bundleTracker}
	success, gasUsed := processor.run(tx)
	require.False(t, success)
	require.Zero(t, gasUsed)
}
