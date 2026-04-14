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

package bundle

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestRunBundle_DelegatesToRunStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}

	txs := map[TxReference]*types.Transaction{
		ref1: tx1,
		ref2: tx2,
		ref3: tx3,
	}

	bundle := &TransactionBundle{
		Transactions: txs,
		Plan: ExecutionPlan{
			Root: NewOneOfStep(
				NewAllOfStep(
					NewTxStep(ref1),
					NewTxStep(ref2),
				),
				NewAllOfStep(
					NewTxStep(ref1),
					NewTxStep(ref3),
				),
				NewAllOfStep(
					NewTxStep(ref2),
					NewTxStep(ref3),
				),
			),
		},
	}

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),

		runner.EXPECT().CreateSnapshot().Return(2),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultFailed),
		runner.EXPECT().RevertToSnapshot(2),

		runner.EXPECT().CreateSnapshot().Return(3),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful),
		runner.EXPECT().Run(tx3).Return(core_types.TransactionResultSuccessful),
		// no revert for second branch since it succeeds

		// third branch should not be executed since one of the branches already succeeded
	)

	require.True(t, RunBundle(bundle, runner))
}

func Test_runStep_DispatchesToCorrectExecutionMode(t *testing.T) {
	tests := map[string]struct {
		step            ExecutionStep
		expectSnapshot  bool // < distinguishes group and transaction mode
		expectedSuccess bool // < distinguishes all-of and one-of mode
	}{
		"single transaction": {
			step:            NewTxStep(TxReference{}),
			expectSnapshot:  false,
			expectedSuccess: false, // transaction is missing
		},
		"all of group": {
			step:            NewAllOfStep(),
			expectSnapshot:  true,
			expectedSuccess: true, // empty group should succeed
		},
		"one of group": {
			step:            NewOneOfStep(),
			expectSnapshot:  true,
			expectedSuccess: false, // empty group should fail
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			if test.expectSnapshot {
				runner.EXPECT().CreateSnapshot().Return(1).Times(1)
				if !test.expectedSuccess {
					runner.EXPECT().RevertToSnapshot(1).Times(1)
				}
			}

			result := runStep(test.step, nil, runner)
			require.Equal(t, test.expectedSuccess, result)
		})
	}
}

func Test_runStep_UnknownStepType_ReturnsFalse(t *testing.T) {
	test := map[string]ExecutionStep{
		"neither single nor group": {single: nil, group: nil},
		"single and group":         {single: &single{}, group: &group{}},
	}

	for name, step := range test {
		t.Run(name, func(t *testing.T) {
			require.False(t, runStep(step, nil, nil))
		})
	}
}

func Test_runGroup_DispatchesToCorrectExecutionMode(t *testing.T) {
	tests := map[string]struct {
		group          group
		expectedResult bool // < distinguishes all-of and one-of mode
	}{
		"all of group": {
			group:          group{oneOf: false},
			expectedResult: true, // empty group should succeed
		},
		"one of group": {
			group:          group{oneOf: true},
			expectedResult: false, // empty group should fail
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			runner.EXPECT().CreateSnapshot().Return(1).Times(1)
			if !test.expectedResult {
				runner.EXPECT().RevertToSnapshot(1).Times(1)
			}

			result := runGroup(test.group, nil, runner)
			require.Equal(t, test.expectedResult, result)
		})
	}
}

func Test_runGroup_HandlesTolerateFailedFlag(t *testing.T) {

	alwaysSucceed := NewAllOfStep()
	alwaysFail := NewOneOfStep()

	tests := map[string]struct {
		step   ExecutionStep
		wanted bool
	}{
		"succeeding and failure not tolerated": {
			step:   alwaysSucceed,
			wanted: true,
		},
		"succeeding and failure tolerated": {
			step:   alwaysSucceed.WithFlags(EF_TolerateFailed),
			wanted: true,
		},
		"failing and failure not tolerated": {
			step:   alwaysFail,
			wanted: false,
		},
		"failing and failure tolerated": {
			step:   alwaysFail.WithFlags(EF_TolerateFailed),
			wanted: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			runner.EXPECT().CreateSnapshot().Return(1)
			runner.EXPECT().RevertToSnapshot(1).MaxTimes(1)

			require.Equal(t, test.wanted, runGroup(*test.step.group, nil, runner))
		})
	}
}

func Test_runAllOfGroup_EmptySteps_ReturnsSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)
	runner.EXPECT().CreateSnapshot().Return(1)

	require.True(t, runAllOfGroup(nil, nil, runner))
}

func Test_runAllOfGroup_ReturnsTrueIfAllTransactionsPass(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{From: common.Address{1}}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultSuccessful).Times(3),
	)

	steps := []ExecutionStep{
		NewTxStep(ref), NewTxStep(ref), NewTxStep(ref),
	}

	txs := map[TxReference]*types.Transaction{
		ref: tx,
	}

	require.True(t, runAllOfGroup(steps, txs, runner))
}

func Test_runAllOfGroup_StopsAtFirstFailedTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}

	txs := map[TxReference]*types.Transaction{
		ref1: tx1,
		ref2: tx2,
		ref3: tx3,
	}

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultFailed),
		// tx3 should not be run
		runner.EXPECT().RevertToSnapshot(1),
	)

	steps := []ExecutionStep{
		NewTxStep(ref1), NewTxStep(ref2), NewTxStep(ref3),
	}

	require.False(t, runAllOfGroup(steps, txs, runner))
}

func Test_runOneOfGroup_ForEmptySteps_ReturnsFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().RevertToSnapshot(1),
	)

	require.False(t, runOneOfGroup(nil, nil, runner))
}

func Test_runOneOfGroup_RollsBackAndReturnsFailedIfAllTransactionsFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultFailed).Times(3),
		runner.EXPECT().RevertToSnapshot(1),
	)

	txs := map[TxReference]*types.Transaction{
		ref: tx,
	}

	steps := []ExecutionStep{
		NewTxStep(ref), NewTxStep(ref), NewTxStep(ref),
	}

	require.False(t, runOneOfGroup(steps, txs, runner))
}

func Test_runOneOfGroup_StopsAtFirstSuccessfulTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}

	txs := map[TxReference]*types.Transaction{
		ref1: tx1,
		ref2: tx2,
		ref3: tx3,
	}

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultFailed),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultSuccessful),
		// tx3 should not be run
	)

	steps := []ExecutionStep{
		NewTxStep(ref1), NewTxStep(ref2), NewTxStep(ref3),
	}

	require.True(t, runOneOfGroup(steps, txs, runner))
}

func Test_runSingle_InterpretsTxResultAsDefinedByFlags(t *testing.T) {
	tests := []core_types.TransactionResult{
		core_types.TransactionResultSuccessful,
		core_types.TransactionResultFailed,
		core_types.TransactionResultInvalid,
	}

	flags := []ExecutionFlags{
		EF_Default,
		EF_TolerateInvalid,
		EF_TolerateFailed,
		EF_TolerateInvalid | EF_TolerateFailed,
	}

	for _, result := range tests {
		for _, flag := range flags {
			t.Run(fmt.Sprintf("result=%d, flags=%b", result, flag), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				runner := NewMockTransactionRunner(ctrl)

				ref := TxReference{}
				tx := types.NewTx(&types.LegacyTx{})
				txs := map[TxReference]*types.Transaction{ref: tx}

				runner.EXPECT().Run(tx).Return(result)

				single := single{flags: flag}
				got := runSingle(single, txs, runner)
				want := isTolerated(result, flag)
				require.Equal(t, want, got)
			})
		}
	}
}

func Test_runSingle_MissingTransaction_AcceptsAsDefinedByFlags(t *testing.T) {
	tests := []ExecutionFlags{
		EF_Default,
		EF_TolerateInvalid,
		EF_TolerateFailed,
		EF_TolerateInvalid | EF_TolerateFailed,
	}

	for _, flags := range tests {
		t.Run(fmt.Sprintf("flags=%b", flags), func(t *testing.T) {
			single := single{flags: flags}
			result := runSingle(single, nil, nil)
			want := isTolerated(core_types.TransactionResultInvalid, flags)
			require.Equal(t, want, result)
		})
	}
}

func Test_isTolerated_InterpretsExecutionFlagsCorrectly(t *testing.T) {
	tests := []struct {
		flags     ExecutionFlags
		result    core_types.TransactionResult
		tolerated bool
	}{
		{flags: EF_Default, result: core_types.TransactionResultInvalid, tolerated: false},
		{flags: EF_Default, result: core_types.TransactionResultFailed, tolerated: false},
		{flags: EF_Default, result: core_types.TransactionResultSuccessful, tolerated: true},
		{flags: EF_Default, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid, result: core_types.TransactionResultInvalid, tolerated: true},
		{flags: EF_TolerateInvalid, result: core_types.TransactionResultFailed, tolerated: false},
		{flags: EF_TolerateInvalid, result: core_types.TransactionResultSuccessful, tolerated: true},
		{flags: EF_TolerateInvalid, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateFailed, result: core_types.TransactionResultInvalid, tolerated: false},
		{flags: EF_TolerateFailed, result: core_types.TransactionResultFailed, tolerated: true},
		{flags: EF_TolerateFailed, result: core_types.TransactionResultSuccessful, tolerated: true},
		{flags: EF_TolerateFailed, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultInvalid, tolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultFailed, tolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultSuccessful, tolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: 99, tolerated: false}, // unknown result treated as failed
	}

	for _, test := range tests {
		require.Equal(t,
			test.tolerated,
			isTolerated(test.result, test.flags),
			"flags: %b, result: %d", test.flags, test.result,
		)
	}
}
