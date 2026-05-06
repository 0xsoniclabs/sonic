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
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
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
	chain.EXPECT().GetCurrentChainConfig().Return(&params.ChainConfig{})

	info := BlockInfo{}
	factory := &evmProcessorFactory{chain: chain}
	result := factory.beginBlock(info.toEvmBlock())
	require.NotNil(t, result)
}

func TestEvmProcessor_Run_IfExecutionSucceeds_ReportsSuccessAndGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	stateDb := state.NewMockStateDB(ctrl)

	stateDb.EXPECT().InterTxSnapshot()
	runner.EXPECT().Run(0, nil).Return(evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{{
			Receipt: &types.Receipt{
				GasUsed: 10,
			},
		}},
		ExecutionCost: core_types.ExecutionCost(10),
	})

	processor := &evmProcessor{processor: runner, stateDb: stateDb}
	success, gasUsed := processor.run(nil)
	require.True(t, success)
	require.Equal(t, uint64(10), gasUsed)
}

func TestEvmProcessor_Run_IfExecutionProducesMultipleProcessedTransactions_SumsUpGasUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	stateDb := state.NewMockStateDB(ctrl)

	stateDb.EXPECT().InterTxSnapshot()
	runner.EXPECT().Run(0, nil).Return(evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Receipt: &types.Receipt{GasUsed: 10}},
			{Receipt: nil}, // skipped transaction
			{Receipt: &types.Receipt{GasUsed: 20}},
		},
		ExecutionCost: core_types.ExecutionCost(30),
	})

	processor := &evmProcessor{processor: runner, stateDb: stateDb}
	success, gasUsed := processor.run(nil)
	require.True(t, success)
	require.Equal(t, uint64(30), gasUsed)
}

func TestEvmProcessor_Run_IfRequestedTransactionIsNotExecuted_TheTransactionIsStillAccepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockevmProcessorRunner(ctrl)
	stateDb := state.NewMockStateDB(ctrl)

	stateDb.EXPECT().InterTxSnapshot()
	tx := &types.Transaction{}
	runner.EXPECT().Run(0, tx).Return(evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{{
			Transaction: &types.Transaction{}, // different transaction
			Receipt:     &types.Receipt{GasUsed: 10},
		}},
		ExecutionCost: core_types.ExecutionCost(10),
	})

	processor := &evmProcessor{processor: runner, stateDb: stateDb}
	success, _ := processor.run(tx)
	require.True(t, success)
}

func TestEvmProcessor_Run_IfExecutionFailed_ReportsAFailedExecutionAndRevertsState(t *testing.T) {
	tx := &types.Transaction{}
	otherTx := &types.Transaction{}

	cases := map[string]struct {
		tx           *types.Transaction
		summary      evmcore.ProcessSummary
		expectAccept bool
	}{
		"nil transaction": {
			tx: nil,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{},
				ExecutionCost:         core_types.ExecutionCost(0),
			},
			expectAccept: false,
		},
		"below threshold no processed transactions": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{},
				ExecutionCost:         core_types.ExecutionCost(0),
			},
			expectAccept: false,
		},
		"below threshold with no receipts": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{
					{Transaction: tx, Receipt: nil},
				},
				ExecutionCost: core_types.ExecutionCost(100),
			},
			expectAccept: false,
		},
		"below threshold with single receipt": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{
					{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100*evmcore.MinBundleEfficiency - 1}},
				},
				ExecutionCost: core_types.ExecutionCost(100),
			},
			expectAccept: false,
		},
		"below threshold with multiple receipts": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{
					{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100/2*evmcore.MinBundleEfficiency - 1}},
					{Transaction: otherTx, Receipt: &types.Receipt{GasUsed: 100/2*evmcore.MinBundleEfficiency - 1}},
				},
				ExecutionCost: core_types.ExecutionCost(100),
			},
			expectAccept: false,
		},
		"above threshold with single receipt": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{
					{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100*evmcore.MinBundleEfficiency + 1}},
				},
				ExecutionCost: core_types.ExecutionCost(100),
			},
			expectAccept: true,
		},
		"above threshold with multiple receipts": {
			tx: tx,
			summary: evmcore.ProcessSummary{
				ProcessedTransactions: []evmcore.ProcessedTransaction{
					{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100/2*evmcore.MinBundleEfficiency + 1}},
					{Transaction: otherTx, Receipt: &types.Receipt{GasUsed: 100/2*evmcore.MinBundleEfficiency + 1}},
				},
				ExecutionCost: core_types.ExecutionCost(100),
			},
			expectAccept: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockevmProcessorRunner(ctrl)
			stateDb := state.NewMockStateDB(ctrl)

			stateDb.EXPECT().InterTxSnapshot()
			runner.EXPECT().Run(0, tc.tx).Return(tc.summary)
			if !tc.expectAccept {
				stateDb.EXPECT().RevertToInterTxSnapshot(gomock.Any())
			}

			processor := &evmProcessor{processor: runner, stateDb: stateDb}
			success, gasUsed := processor.run(tc.tx)
			require.Equal(t, tc.expectAccept, success)
			if tc.expectAccept {
				expectedGasUsed := uint64(0)
				for _, tx := range tc.summary.ProcessedTransactions {
					if tx.Receipt != nil {
						expectedGasUsed += tx.Receipt.GasUsed
					}
				}
				require.Equal(t, expectedGasUsed, gasUsed)
			} else {
				require.Zero(t, gasUsed)
			}
		})
	}
}

func TestEvmProcessor_Release_ReleasesStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	processor := &evmProcessor{stateDb: stateDb}
	stateDb.EXPECT().Release()
	processor.release()
}
