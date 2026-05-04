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

func asPointer[T any](v T) *T { return &v }

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
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultFailed, core_types.ExecutionCost(0)),
		runner.EXPECT().RevertToSnapshot(2),

		runner.EXPECT().CreateSnapshot().Return(3),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)),
		runner.EXPECT().Run(tx3).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)),
		// no revert for second branch since it succeeds

		// third branch should not be executed since one of the branches already succeeded
	)

	success, _ := RunBundle(bundle, runner)
	require.True(t, success)
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

			result, _ := runStep(test.step, nil, runner)
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
			result, _ := runStep(step, nil, nil)
			require.False(t, result)
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

			result, _ := runGroup(test.group, nil, runner)
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

			result, _ := runGroup(*test.step.group, nil, runner)
			require.Equal(t, test.wanted, result)
		})
	}
}

func Test_runAllOfGroup_EmptySteps_ReturnsSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)
	runner.EXPECT().CreateSnapshot().Return(1)

	success, _ := runAllOfGroup(nil, nil, runner)
	require.True(t, success)
}

func Test_runAllOfGroup_ReturnsTrueIfAllTransactionsPass(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{From: common.Address{1}}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)).Times(3),
	)

	steps := []ExecutionStep{
		NewTxStep(ref), NewTxStep(ref), NewTxStep(ref),
	}

	txs := map[TxReference]*types.Transaction{
		ref: tx,
	}

	success, _ := runAllOfGroup(steps, txs, runner)
	require.True(t, success)
}

func Test_runAllOfGroup_StopsAtFirstFailedTransaction(t *testing.T) {
	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{}),
		types.NewTx(&types.LegacyTx{}),
		types.NewTx(&types.LegacyTx{}),
	}

	refs := []TxReference{}
	for i := range txs {
		refs = append(refs, TxReference{From: common.Address{byte(i)}})
	}

	index := map[TxReference]*types.Transaction{}
	for i, ref := range refs {
		index[ref] = txs[i]
	}

	steps := []ExecutionStep{}
	for _, ref := range refs {
		steps = append(steps, NewTxStep(ref))
	}

	for firstFailed := range txs {
		t.Run(fmt.Sprintf("first failed tx index: %d", firstFailed), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			head := runner.EXPECT().CreateSnapshot().Return(1)
			for i := range firstFailed {
				head = runner.EXPECT().
					Run(txs[i]).
					Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)).
					After(head)
			}
			head = runner.EXPECT().Run(txs[firstFailed]).
				Return(core_types.TransactionResultFailed, core_types.ExecutionCost(0)).
				After(head)

			runner.EXPECT().RevertToSnapshot(1).After(head)

			success, _ := runAllOfGroup(steps, index, runner)
			require.False(t, success)
		})
	}
}

func Test_runOneOfGroup_ForEmptySteps_ReturnsFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().RevertToSnapshot(1),
	)

	success, _ := runOneOfGroup(nil, nil, runner)
	require.False(t, success)
}

func Test_runOneOfGroup_RollsBackAndReturnsFailedIfAllTransactionsFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultFailed, core_types.ExecutionCost(0)).Times(3),
		runner.EXPECT().RevertToSnapshot(1),
	)

	txs := map[TxReference]*types.Transaction{
		ref: tx,
	}

	steps := []ExecutionStep{
		NewTxStep(ref), NewTxStep(ref), NewTxStep(ref),
	}

	success, _ := runOneOfGroup(steps, txs, runner)
	require.False(t, success)
}

func Test_runOneOfGroup_StopsAtFirstSuccessfulTransaction(t *testing.T) {
	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{}),
		types.NewTx(&types.LegacyTx{}),
		types.NewTx(&types.LegacyTx{}),
	}

	refs := []TxReference{}
	for i := range txs {
		refs = append(refs, TxReference{From: common.Address{byte(i)}})
	}

	index := map[TxReference]*types.Transaction{}
	for i, ref := range refs {
		index[ref] = txs[i]
	}

	steps := []ExecutionStep{}
	for _, ref := range refs {
		steps = append(steps, NewTxStep(ref))
	}

	for firstSuccess := range txs {
		t.Run(fmt.Sprintf("first successful tx index: %d", firstSuccess), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			head := runner.EXPECT().CreateSnapshot().Return(1)
			for i := range firstSuccess {
				head = runner.EXPECT().
					Run(txs[i]).
					Return(core_types.TransactionResultFailed, core_types.ExecutionCost(0)).
					After(head)
			}
			runner.EXPECT().Run(txs[firstSuccess]).
				Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(0)).
				After(head)

			success, _ := runOneOfGroup(steps, index, runner)
			require.True(t, success)
		})
	}
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

				runner.EXPECT().Run(tx).Return(result, core_types.ExecutionCost(0))

				single := single{flags: flag}
				got, _ := runSingle(single, txs, runner)
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
			result, _ := runSingle(single, nil, nil)
			want := isTolerated(core_types.TransactionResultInvalid, flags)
			require.Equal(t, want, result)
		})
	}
}

func Test_isTolerated_InterpretsExecutionFlagsCorrectly(t *testing.T) {
	tests := []struct {
		flags             ExecutionFlags
		result            core_types.TransactionResult
		expectedTolerated bool
	}{
		{flags: EF_Default, result: core_types.TransactionResultInvalid, expectedTolerated: false},
		{flags: EF_Default, result: core_types.TransactionResultFailed, expectedTolerated: false},
		{flags: EF_Default, result: core_types.TransactionResultSuccessful, expectedTolerated: true},
		{flags: EF_Default, result: 99, expectedTolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid, result: core_types.TransactionResultInvalid, expectedTolerated: true},
		{flags: EF_TolerateInvalid, result: core_types.TransactionResultFailed, expectedTolerated: false},
		{flags: EF_TolerateInvalid, result: core_types.TransactionResultSuccessful, expectedTolerated: true},
		{flags: EF_TolerateInvalid, result: 99, expectedTolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateFailed, result: core_types.TransactionResultInvalid, expectedTolerated: false},
		{flags: EF_TolerateFailed, result: core_types.TransactionResultFailed, expectedTolerated: true},
		{flags: EF_TolerateFailed, result: core_types.TransactionResultSuccessful, expectedTolerated: true},
		{flags: EF_TolerateFailed, result: 99, expectedTolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultInvalid, expectedTolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultFailed, expectedTolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: core_types.TransactionResultSuccessful, expectedTolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: 99, expectedTolerated: false}, // unknown result treated as failed
	}

	for _, test := range tests {
		require.Equal(t,
			test.expectedTolerated,
			isTolerated(test.result, test.flags),
			"flags: %b, result: %d", test.flags, test.result,
		)
	}
}

// --- ExecutionCost tests ---

func TestRunBundle_ReturnsExecutionCostOfSuccessfulBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{From: common.Address{1}}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(100)),
	)

	b := &TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{ref: tx},
		Plan:         ExecutionPlan{Root: NewAllOfStep(NewTxStep(ref))},
	}

	success, execCost := RunBundle(b, runner)
	require.True(t, success)
	require.Equal(t, core_types.ExecutionCost(100), execCost)
}

func Test_runAllOfGroup_ExecutionCostIsSumOfGasFromNonRevertedAndRevertedSteps(t *testing.T) {
	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})

	// The steps have an execution cost of 100 and 200 gas respectively.
	cases := map[string]struct {
		tx1Result    core_types.TransactionResult
		tx2Result    *core_types.TransactionResult
		expectedCost core_types.ExecutionCost
	}{
		"both succeed": {
			tx1Result:    core_types.TransactionResultSuccessful,
			tx2Result:    asPointer(core_types.TransactionResultSuccessful),
			expectedCost: core_types.ExecutionCost(300),
		},
		"first succeeds, second fails": {
			tx1Result:    core_types.TransactionResultSuccessful,
			tx2Result:    asPointer(core_types.TransactionResultFailed),
			expectedCost: core_types.ExecutionCost(300),
		},
		"first fails": {
			tx1Result:    core_types.TransactionResultFailed,
			expectedCost: core_types.ExecutionCost(100), // second step not executed
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			runner.EXPECT().CreateSnapshot().Return(1)
			runner.EXPECT().Run(tx1).Return(tc.tx1Result, core_types.ExecutionCost(100))
			if tc.tx2Result != nil {
				runner.EXPECT().Run(tx2).Return(*tc.tx2Result, core_types.ExecutionCost(200))
			}
			if tc.tx1Result == core_types.TransactionResultFailed || (tc.tx2Result != nil && *tc.tx2Result == core_types.TransactionResultFailed) {
				runner.EXPECT().RevertToSnapshot(1)
			}

			txs := map[TxReference]*types.Transaction{ref1: tx1, ref2: tx2}
			steps := []ExecutionStep{NewTxStep(ref1), NewTxStep(ref2)}

			_, execCost := runAllOfGroup(steps, txs, runner)
			require.Equal(t, tc.expectedCost, execCost)
		})
	}
}

func Test_runOneOfGroup_ExecutionCostIncludesGasFromAllAttemptedSteps(t *testing.T) {
	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})

	// The steps have an execution cost of 100 and 200 gas respectively.
	cases := map[string]struct {
		tx1Result    core_types.TransactionResult
		tx2Result    *core_types.TransactionResult
		expectedCost core_types.ExecutionCost
	}{
		"first succeeds": {
			tx1Result:    core_types.TransactionResultSuccessful,
			expectedCost: core_types.ExecutionCost(100), // second step not executed
		},
		"first fails, second succeeds": {
			tx1Result:    core_types.TransactionResultFailed,
			tx2Result:    asPointer(core_types.TransactionResultSuccessful),
			expectedCost: core_types.ExecutionCost(300),
		},
		"both fail": {
			tx1Result:    core_types.TransactionResultFailed,
			tx2Result:    asPointer(core_types.TransactionResultFailed),
			expectedCost: core_types.ExecutionCost(300),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMockTransactionRunner(ctrl)

			runner.EXPECT().CreateSnapshot().Return(1)
			runner.EXPECT().Run(tx1).Return(tc.tx1Result, core_types.ExecutionCost(100))
			if tc.tx2Result != nil {
				runner.EXPECT().Run(tx2).Return(*tc.tx2Result, core_types.ExecutionCost(200))
			}
			if tc.tx2Result != nil && *tc.tx2Result == core_types.TransactionResultFailed {
				runner.EXPECT().RevertToSnapshot(1)
			}

			txs := map[TxReference]*types.Transaction{ref1: tx1, ref2: tx2}
			steps := []ExecutionStep{NewTxStep(ref1), NewTxStep(ref2)}

			_, execCost := runOneOfGroup(steps, txs, runner)
			require.Equal(t, tc.expectedCost, execCost)
		})
	}
}

func Test_runOneOfGroup_ExecutionCostIncludesGasWhenAllStepsFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultFailed, core_types.ExecutionCost(100)),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultFailed, core_types.ExecutionCost(200)),
		runner.EXPECT().RevertToSnapshot(1),
	)

	txs := map[TxReference]*types.Transaction{ref1: tx1, ref2: tx2}
	steps := []ExecutionStep{NewTxStep(ref1), NewTxStep(ref2)}

	success, execCost := runOneOfGroup(steps, txs, runner)
	require.False(t, success)
	require.Equal(t, core_types.ExecutionCost(300), execCost)
}

func Test_runOneOfGroup_ExecutionCostOnlyCountsFirstSuccessfulStep(t *testing.T) {
	// In a oneOf, once the first step succeeds, subsequent steps are not run.
	// The cost should only include steps that were actually attempted.
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{From: common.Address{1}}
	tx := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1),
		runner.EXPECT().Run(tx).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(1000)),
		// second step is NOT run since first succeeded
	)

	txs := map[TxReference]*types.Transaction{ref: tx}
	steps := []ExecutionStep{NewTxStep(ref), NewTxStep(ref)}

	success, execCost := runOneOfGroup(steps, txs, runner)
	require.True(t, success)
	require.Equal(t, core_types.ExecutionCost(1000), execCost)
}

func TestRunBundle_NestedGroups_AccumulateExecutionCostAcrossLevels(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}
	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})

	// OneOf(AllOf(tx1, tx2), tx3)
	// AllOf branch: tx1 succeeds (100 gas), tx2 fails (200 gas) -> reverted
	// tx3 succeeds (50 gas)
	// Total cost: 100 + 200 + 50 = 350
	gomock.InOrder(
		runner.EXPECT().CreateSnapshot().Return(1), // oneOf snapshot
		runner.EXPECT().CreateSnapshot().Return(2), // allOf snapshot
		runner.EXPECT().Run(tx1).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(100)),
		runner.EXPECT().Run(tx2).Return(core_types.TransactionResultFailed, core_types.ExecutionCost(200)),
		runner.EXPECT().RevertToSnapshot(2), // allOf reverted
		runner.EXPECT().Run(tx3).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(50)),
	)

	txs := map[TxReference]*types.Transaction{ref1: tx1, ref2: tx2, ref3: tx3}
	b := &TransactionBundle{
		Transactions: txs,
		Plan: ExecutionPlan{
			Root: NewOneOfStep(
				NewAllOfStep(NewTxStep(ref1), NewTxStep(ref2)),
				NewTxStep(ref3),
			),
		},
	}

	success, execCost := RunBundle(b, runner)
	require.True(t, success)
	require.Equal(t, core_types.ExecutionCost(350), execCost)
}

func Test_runSingle_ReturnsExecutionCostFromRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	ref := TxReference{From: common.Address{1}}
	tx := types.NewTx(&types.LegacyTx{})
	txs := map[TxReference]*types.Transaction{ref: tx}

	runner.EXPECT().Run(tx).Return(core_types.TransactionResultSuccessful, core_types.ExecutionCost(42))

	s := single{txRef: ref}
	success, execCost := runSingle(s, txs, runner)
	require.True(t, success)
	require.Equal(t, core_types.ExecutionCost(42), execCost)
}

func Test_runSingle_MissingTransaction_ReturnsZeroExecutionCost(t *testing.T) {
	s := single{txRef: TxReference{}, flags: EF_TolerateInvalid}
	success, execCost := runSingle(s, nil, nil)
	require.True(t, success)
	require.Equal(t, core_types.ExecutionCost(0), execCost)
}
