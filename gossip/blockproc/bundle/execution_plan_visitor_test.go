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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundleExecutionPlanVisitor_VisitsStructure(t *testing.T) {

	ctrl := gomock.NewController(t)
	visitor := NewMockExecutionPlanVisitor(ctrl)

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}

	step1 := NewTxStep(ref1)
	step2 := NewTxStep(ref2)

	plan := ExecutionPlan{
		Root: NewOneOfStep(
			NewAllOfStep(step1, step2),
			NewGroupStep(false),
			NewAllOfStep(
				step1.WithFlags(EF_TolerateInvalid),
			),
		).WithFlags(EF_TolerateFailed),
	}

	gomock.InOrder(
		visitor.EXPECT().BeginGroup(true, true),
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().Step(EF_Default, ref1),
		visitor.EXPECT().Step(EF_Default, ref2),
		visitor.EXPECT().EndGroup(),
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().EndGroup(),
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().Step(EF_TolerateInvalid, ref1),
		visitor.EXPECT().EndGroup(),
		visitor.EXPECT().EndGroup(),
	)

	err := plan.Root.Accept(visitor)
	require.NoError(t, err)
}

func TestBundleExecutionPlanVisitor_ForwardsFlags(t *testing.T) {
	ctrl := gomock.NewController(t)
	visitor := NewMockExecutionPlanVisitor(ctrl)

	ref1 := TxReference{From: common.Address{1}}
	step1 := NewTxStep(ref1)

	plan := ExecutionPlan{
		Root: NewAllOfStep(
			step1,
			step1.WithFlags(EF_Default),
			step1.WithFlags(EF_TolerateInvalid),
			step1.WithFlags(EF_TolerateFailed),
			step1.WithFlags(EF_TolerateFailed|EF_TolerateInvalid),
		),
	}

	gomock.InOrder(
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().Step(EF_Default, ref1),
		visitor.EXPECT().Step(EF_Default, ref1),
		visitor.EXPECT().Step(EF_TolerateInvalid, ref1),
		visitor.EXPECT().Step(EF_TolerateFailed, ref1),
		visitor.EXPECT().Step(EF_TolerateInvalid|EF_TolerateFailed, ref1),
		visitor.EXPECT().EndGroup(),
	)

	err := plan.Root.Accept(visitor)
	require.NoError(t, err)
}

func TestBundleExecutionPlanVisitor_ForwardsErrors(t *testing.T) {

	ctrl := gomock.NewController(t)
	visitor := NewMockExecutionPlanVisitor(ctrl)

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}

	step1 := NewTxStep(ref1)
	step2 := NewTxStep(ref2)

	plan := ExecutionPlan{
		Root: NewAllOfStep(
			NewAllOfStep(step1, step1),
			NewAllOfStep(step1, step2),
		),
	}

	expectedError := errors.New("test error")

	gomock.InOrder(
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().BeginGroup(false, false),
		visitor.EXPECT().Step(EF_Default, ref1).Return(expectedError),
		visitor.EXPECT().EndGroup(),
		visitor.EXPECT().EndGroup(),
	)

	err := plan.Root.Accept(visitor)
	require.ErrorIs(t, err, expectedError)
}
