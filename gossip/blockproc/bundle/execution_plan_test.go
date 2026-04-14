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
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestExecutionPlan_Hash_ComputesDeterministicHash(t *testing.T) {

	ref1 := TxReference{
		From: common.Address{1},
		Hash: common.Hash{2},
	}

	ref2 := TxReference{
		From: common.Address{3},
		Hash: common.Hash{4},
	}

	step1 := NewTxStep(ref1)
	step2 := NewTxStep(ref2)

	tests := map[string]ExecutionPlan{
		"plan with single step": {
			Root: step1,
		},
		"plan with different single step": {
			Root: step2,
		},
		"plan with single step and execution flags 1": {
			Root: step1.WithFlags(EF_TolerateFailed),
		},
		"plan with single step and execution flags 2": {
			Root: step1.WithFlags(EF_TolerateInvalid),
		},
		"plan with single step and execution flags 3": {
			Root: step2.WithFlags(EF_TolerateFailed | EF_TolerateInvalid),
		},
		"plan with all-of group": {
			Root: NewAllOfStep(step1, step2),
		},
		"plan with different all-of group": {
			Root: NewAllOfStep(step2, step1),
		},
		"plan with all-of group tolerating failed": {
			Root: NewAllOfStep(step1, step2).WithFlags(EF_TolerateFailed),
		},
		"plan with one-of group": {
			Root: NewOneOfStep(step1, step2),
		},
		"plan with different one-of group": {
			Root: NewOneOfStep(step2, step1),
		},
		"plan with one-of group and tolerating failed": {
			Root: NewOneOfStep(step1, step2).WithFlags(EF_TolerateFailed),
		},
		"plan with nested groups": {
			Root: NewOneOfStep(
				NewAllOfStep(step1, step2),
				NewAllOfStep(step2, step1),
			),
		},
		"plan with different nested groups": {
			Root: NewOneOfStep(
				NewAllOfStep(step2, step1),
				NewAllOfStep(step1, step2),
			),
		},
		"plan with block range": {
			Root:  step1,
			Range: BlockRange{Earliest: 10, Latest: 20},
		},
		"plan with different start": {
			Root:  step1,
			Range: BlockRange{Earliest: 11, Latest: 20},
		},
		"plan with different end": {
			Root:  step1,
			Range: BlockRange{Earliest: 10, Latest: 21},
		},
	}

	seenHashes := make(map[common.Hash]struct{})
	for name, executionPlan := range tests {
		t.Run(name, func(t *testing.T) {

			hasher := crypto.NewKeccakState()
			require.NoError(t, executionPlan.encode(hasher))
			computed := common.BytesToHash(hasher.Sum(nil))

			require.Equal(t, executionPlan.Hash(), computed)
			require.NotContains(t, seenHashes, computed, "hash should be unique for different plans")
			seenHashes[computed] = struct{}{}
		})
	}
}

func TestExecutionStep_GetTransactionReferencesInReferencedOrder_ReturnsReferencesInCorrectOrder(t *testing.T) {

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{1}}
	ref3 := TxReference{From: common.Address{1}}
	ref4 := TxReference{From: common.Address{1}}

	tests := map[string]struct {
		input ExecutionStep
		want  []TxReference
	}{
		"empty": {
			input: ExecutionStep{},
			want:  nil,
		},
		"single": {
			input: NewTxStep(ref1),
			want:  []TxReference{ref1},
		},
		"allOf group": {
			input: NewAllOfStep(
				NewTxStep(ref1),
				NewTxStep(ref2),
				NewTxStep(ref3),
				NewTxStep(ref4),
			),
			want: []TxReference{ref1, ref2, ref3, ref4},
		},
		"duplicate references": {
			input: NewOneOfStep(
				NewTxStep(ref1),
				NewTxStep(ref2),
				NewTxStep(ref1),
			),
			want: []TxReference{ref1, ref2, ref1},
		},
		"nested groups": {
			input: NewOneOfStep(
				NewAllOfStep(NewTxStep(ref1), NewTxStep(ref2)),
				NewAllOfStep(NewTxStep(ref1), NewTxStep(ref3)),
				NewAllOfStep(NewTxStep(ref2), NewTxStep(ref3)),
			),
			want: []TxReference{ref1, ref2, ref1, ref3, ref2, ref3},
		},
		// Also provide a clear definition for an invalid case. Even though it
		// should not show up in practice, it is possible and should have a
		// defined behavior.
		"invalid single and group step": {
			input: ExecutionStep{
				single: &single{txRef: ref1},
				group: &group{steps: []ExecutionStep{
					NewTxStep(ref2), NewTxStep(ref3),
				}},
			},
			want: []TxReference{ref1, ref2, ref3},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			got := tc.input.GetTransactionReferencesInReferencedOrder()
			require.Equal(tc.want, got)
		})
	}
}

func TestNewTxStep_CreatesExecutionStepWithSingleTransaction(t *testing.T) {
	ref := TxReference{
		From: common.Address{1},
		Hash: common.Hash{2},
	}

	step := NewTxStep(ref)
	require := require.New(t)
	require.NotNil(step.single)
	require.Nil(step.group)
	require.Equal(ref, step.single.txRef)
}

func TestNewAllOfStep_CreatesExecutionStepWithAllOfGroup(t *testing.T) {
	step1 := NewTxStep(TxReference{From: common.Address{1}})
	step2 := NewTxStep(TxReference{From: common.Address{2}})

	step := NewAllOfStep(step1, step2)
	require := require.New(t)
	require.Nil(step.single)
	require.NotNil(step.group)
	require.False(step.group.oneOf)
	require.Equal([]ExecutionStep{step1, step2}, step.group.steps)
}

func TestNewOneOfStep_CreatesExecutionStepWithOneOfGroup(t *testing.T) {
	step1 := NewTxStep(TxReference{From: common.Address{1}})
	step2 := NewTxStep(TxReference{From: common.Address{2}})

	step := NewOneOfStep(step1, step2)
	require := require.New(t)
	require.Nil(step.single)
	require.NotNil(step.group)
	require.True(step.group.oneOf)
	require.Equal([]ExecutionStep{step1, step2}, step.group.steps)
}

func TestExecutionStep_WithFlags_ReturnsNewExecutionStepWithUpdatedFlags(t *testing.T) {
	flags := []ExecutionFlags{
		EF_TolerateFailed,
		EF_TolerateInvalid,
		EF_TolerateFailed | EF_TolerateInvalid,
	}

	for _, flag := range flags {
		step := NewTxStep(TxReference{})
		updated := step.WithFlags(flag)

		require := require.New(t)
		require.NotEqual(step, updated, "WithFlags should return a new instance")
		require.Equal(flag, updated.single.flags)
		require.Equal(step.single.txRef, updated.single.txRef)
	}
}

func TestExecutionStep_WithFlags_PanicsWhenTolerateInvalidFlagIsUsedForGroup(t *testing.T) {
	step := NewAllOfStep(NewTxStep(TxReference{}))
	require.Panics(t, func() {
		step.WithFlags(EF_TolerateInvalid)
	}, "WithFlags should panic when TolerateInvalid flag is used for a group")
}

func TestExecutionStep_EncodingAndDecodingAreAligned(t *testing.T) {
	ref1 := TxReference{
		From: common.Address{1},
		Hash: common.Hash{2},
	}

	ref2 := TxReference{
		From: common.Address{3},
		Hash: common.Hash{4},
	}

	step1 := NewTxStep(ref1)
	step2 := NewTxStep(ref2)

	tests := map[string]ExecutionStep{
		"single step":                        step1,
		"different single step":              step2,
		"single step and execution flags 1":  step1.WithFlags(0x1),
		"single step and execution flags 2":  step1.WithFlags(0x2),
		"single step and execution flags 3":  step2.WithFlags(0x3),
		"all-of group":                       NewAllOfStep(step1, step2),
		"different all-of group":             NewAllOfStep(step2, step1),
		"all-of group tolerating failed":     NewAllOfStep(step1, step2).WithFlags(EF_TolerateFailed),
		"one-of group":                       NewOneOfStep(step1, step2),
		"different one-of group":             NewOneOfStep(step2, step1),
		"one-of group and tolerating failed": NewOneOfStep(step1, step2).WithFlags(EF_TolerateFailed),
		"nested groups": NewOneOfStep(
			NewAllOfStep(step1, step2),
			NewAllOfStep(step2, step1),
		),
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			var buf bytes.Buffer
			require.NoError(input.encode(&buf))

			data := buf.Bytes()

			var decoded ExecutionStep
			require.NoError(decoded.decode(&buf))

			require.Equal(input, decoded)

			var buf2 bytes.Buffer
			require.NoError(decoded.encode(&buf2))

			require.Equal(data, buf2.Bytes())
		})
	}
}

func TestExecutionStep_encode_FailsOnInvalidStep(t *testing.T) {
	tests := map[string]ExecutionStep{
		"empty step": {},
		"step with both single and group set": {
			single: &single{txRef: TxReference{}},
			group:  &group{steps: []ExecutionStep{}},
		},
	}

	for name, step := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, step.encode(nil), "can not encode invalid execution step")
		})
	}
}

func TestExecutionStep_decode_FailsOnInvalidInput(t *testing.T) {
	data := []byte("invalid rlp data")
	var s ExecutionStep
	require.Error(t, s.decode(bytes.NewReader(data)))
}

func TestExecutionStep_String_PrintsReadableRepresentation(t *testing.T) {
	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}

	step1 := NewTxStep(ref1)
	step2 := NewTxStep(ref2)
	step3 := NewTxStep(ref3)

	tests := map[string]struct {
		input ExecutionStep
		want  string
	}{
		"zero value": {
			input: ExecutionStep{},
			want:  "InvalidStep",
		},
		"single transaction step": {
			input: NewTxStep(ref1),
			want:  "A",
		},
		"single transaction tolerating invalid": {
			input: NewTxStep(ref1).WithFlags(EF_TolerateInvalid),
			want:  "Step[TolerateInvalid](A)",
		},
		"single transaction tolerating failed": {
			input: NewTxStep(ref1).WithFlags(EF_TolerateFailed),
			want:  "Step[TolerateFailed](A)",
		},
		"single transaction tolerating invalid and failed": {
			input: NewTxStep(ref1).WithFlags(EF_TolerateInvalid | EF_TolerateFailed),
			want:  "Step[TolerateInvalid|TolerateFailed](A)",
		},
		"allOf group": {
			input: NewAllOfStep(step1, step2),
			want:  "AllOf(A,B)",
		},
		"oneOf group": {
			input: NewOneOfStep(step1, step2),
			want:  "OneOf(A,B)",
		},
		"group with execution flags": {
			input: NewAllOfStep(step1, step2).WithFlags(EF_TolerateFailed),
			want:  "TolerateFailed(AllOf(A,B))",
		},
		"repeated transactions": {
			input: NewAllOfStep(step1, step2, step1),
			want:  "AllOf(A,B,A)",
		},
		"nested groups": {
			input: NewOneOfStep(
				NewAllOfStep(step1, step2),
				NewAllOfStep(step1, step3),
				NewAllOfStep(step2, step3),
			),
			want: "OneOf(AllOf(A,B),AllOf(A,C),AllOf(B,C))",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			require.Equal(tc.want, tc.input.String())
		})
	}
}
