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
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_RunBundle_HandlesExecutionModeCorrectly(t *testing.T) {
	require := require.New(t)
	require.True(RunBundle(&TransactionBundle{Flags: EF_AllOf}, nil))
	require.False(RunBundle(&TransactionBundle{Flags: EF_OneOf}, nil))
}

func Test_runAllOfBundle_ReturnsTrueForEmptyBundle(t *testing.T) {
	emptyBundle := &TransactionBundle{Bundle: nil}
	result := runAllOfBundle(emptyBundle, nil)
	require.True(t, result)
}

func Test_runAllOfBundle_ReturnsTrueIfAllTransactionsSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(tx).Return(TransactionResultSuccessful).Times(3)

	bundle := &TransactionBundle{
		Bundle: []*types.Transaction{tx, tx, tx},
	}

	result := runAllOfBundle(bundle, runner)
	require.True(t, result)
}

func Test_runAllOfBundle_StopsAtFirstFailedTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})
	gomock.InOrder(
		runner.EXPECT().Run(tx1).Return(TransactionResultSuccessful),
		runner.EXPECT().Run(tx2).Return(TransactionResultFailed),
		// tx3 should not be run
	)

	bundle := &TransactionBundle{
		Bundle: []*types.Transaction{tx1, tx2, tx3},
	}

	result := runAllOfBundle(bundle, runner)
	require.False(t, result)
}

func Test_runOneOfBundle_ReturnsFalseForEmptyBundle(t *testing.T) {
	emptyBundle := &TransactionBundle{Bundle: nil}
	result := runOneOfBundle(emptyBundle, nil)
	require.False(t, result)
}

func Test_runOneOfBundle_ReturnsFalseIfAllTransactionsFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx := types.NewTx(&types.LegacyTx{})
	runner.EXPECT().Run(tx).Return(TransactionResultFailed).Times(3)

	bundle := &TransactionBundle{
		Bundle: []*types.Transaction{tx, tx, tx},
	}

	result := runOneOfBundle(bundle, runner)
	require.False(t, result)
}

func Test_runOneOfBundle_StopsAtFirstSuccessfulTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMockTransactionRunner(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{})
	tx2 := types.NewTx(&types.LegacyTx{})
	tx3 := types.NewTx(&types.LegacyTx{})

	gomock.InOrder(
		runner.EXPECT().Run(tx1).Return(TransactionResultFailed),
		runner.EXPECT().Run(tx2).Return(TransactionResultSuccessful),
		// tx3 should not be run
	)

	bundle := &TransactionBundle{
		Bundle: []*types.Transaction{tx1, tx2, tx3},
	}

	result := runOneOfBundle(bundle, runner)
	require.True(t, result)
}

func Test_isTolerated_InterpretsExecutionFlagsCorrectly(t *testing.T) {
	tests := []struct {
		flags     ExecutionFlag
		result    TransactionResult
		tolerated bool
	}{
		{flags: 0, result: TransactionResultInvalid, tolerated: false},
		{flags: 0, result: TransactionResultFailed, tolerated: false},
		{flags: 0, result: TransactionResultSuccessful, tolerated: true},
		{flags: 0, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid, result: TransactionResultInvalid, tolerated: true},
		{flags: EF_TolerateInvalid, result: TransactionResultFailed, tolerated: false},
		{flags: EF_TolerateInvalid, result: TransactionResultSuccessful, tolerated: true},
		{flags: EF_TolerateInvalid, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateFailed, result: TransactionResultInvalid, tolerated: false},
		{flags: EF_TolerateFailed, result: TransactionResultFailed, tolerated: true},
		{flags: EF_TolerateFailed, result: TransactionResultSuccessful, tolerated: true},
		{flags: EF_TolerateFailed, result: 99, tolerated: false}, // unknown result treated as failed

		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: TransactionResultInvalid, tolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: TransactionResultFailed, tolerated: true},
		{flags: EF_TolerateInvalid | EF_TolerateFailed, result: TransactionResultSuccessful, tolerated: true},
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
