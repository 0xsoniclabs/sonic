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
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			Range: BlockRange{First: 10, Length: 12},
		},
		"plan with different start": {
			Root:  step1,
			Range: BlockRange{First: 11, Length: 12},
		},
		"plan with different length": {
			Root:  step1,
			Range: BlockRange{First: 10, Length: 14},
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

func TestExecutionPlan_encode_StartsWithVersionNumber(t *testing.T) {
	executionPlan := ExecutionPlan{
		Range:  BlockRange{First: 10, Length: 12},
		Period: TimePeriod{Start: 100, Duration: 200},
		Root: NewTxStep(TxReference{
			From: common.Address{1},
			Hash: common.Hash{2},
		}),
	}

	buf := &bytes.Buffer{}
	require.NoError(t, executionPlan.encode(buf))
	require.NotEmpty(t, buf.Bytes())
	require.Equal(t, byte(1), buf.Bytes()[0])
}

func TestExecutionPlan_encode_DetectsShortWriteForVersionNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := NewMockWriter(ctrl)
	writer.EXPECT().Write(gomock.Any()).Return(0, nil)

	executionPlan := ExecutionPlan{
		Range:  BlockRange{First: 10, Length: 12},
		Period: TimePeriod{Start: 100, Duration: 200},
		Root: NewTxStep(TxReference{
			From: common.Address{1},
			Hash: common.Hash{2},
		}),
	}

	require.ErrorContains(t, executionPlan.encode(writer), "failed to write version byte")
}

func TestExecutionPlan_encode_ReportsWriteError(t *testing.T) {
	ctrl := gomock.NewController(t)

	executionPlan := ExecutionPlan{
		Range:  BlockRange{First: 10, Length: 12},
		Period: TimePeriod{Start: 100, Duration: 200},
		Root: NewTxStep(TxReference{
			From: common.Address{1},
			Hash: common.Hash{2},
		}),
	}

	numWrites := 0
	writer := NewMockWriter(ctrl)
	writer.EXPECT().Write(gomock.Any()).DoAndReturn(
		func(data []byte) (int, error) {
			numWrites++
			return len(data), nil
		},
	).AnyTimes()

	require.NoError(t, executionPlan.encode(writer))
	require.NotZero(t, numWrites)

	for i := range numWrites {
		t.Run(fmt.Sprintf("issue_after=%d", i), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issue := fmt.Errorf("injected issue")
			writer := NewMockWriter(ctrl)
			gomock.InOrder(
				writer.EXPECT().Write(gomock.Any()).Return(1, nil).Times(i),
				writer.EXPECT().Write(gomock.Any()).Return(0, issue),
				writer.EXPECT().Write(gomock.Any()).AnyTimes(),
			)
			require.ErrorContains(t, executionPlan.encode(writer), issue.Error())
		})
	}
}

func TestExecutionPlan_decode_ReportsVersionMismatch(t *testing.T) {
	data := []byte{0xFF}
	var executionPlan ExecutionPlan
	require.ErrorContains(t,
		executionPlan.decode(bytes.NewReader(data)),
		"unsupported execution plan version",
	)
}

func TestExecutionPlan_decode_ReportsReadError(t *testing.T) {
	executionPlan := ExecutionPlan{
		Range:  BlockRange{First: 10, Length: 12},
		Period: TimePeriod{Start: 100, Duration: 200},
		Root: NewTxStep(TxReference{
			From: common.Address{1},
			Hash: common.Hash{2},
		}),
	}

	buf := &bytes.Buffer{}
	require.NoError(t, executionPlan.encode(buf))
	encoded := buf.Bytes()
	require.NotEmpty(t, encoded)

	// prune input data to trigger read errors at different points in the
	// decoding process, and verify that an error is returned in each case
	for i := range len(encoded) {
		t.Run(fmt.Sprintf("issue_after=%d", i), func(t *testing.T) {
			reader := bytes.NewBuffer(encoded[:i])
			require.Error(t, executionPlan.decode(reader))
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

func TestExecutionStep_WithFlags_CanBeUsedToResetFlags(t *testing.T) {
	step := NewTxStep(TxReference{
		From: common.Address{12},
	}).WithFlags(EF_TolerateFailed | EF_TolerateInvalid)
	updated := step.WithFlags(EF_Default)

	require := require.New(t)
	require.NotEqual(step, updated, "WithFlags should return a new instance")
	require.Equal(EF_Default, updated.single.flags)
	require.Equal(step.single.txRef, updated.single.txRef)
}

func TestExecutionStep_WithFlags_PanicsWhenTolerateInvalidFlagIsUsedForGroup(t *testing.T) {
	step := NewAllOfStep(NewTxStep(TxReference{}))
	require.Panics(t, func() {
		step.WithFlags(EF_TolerateInvalid)
	}, "WithFlags should panic when TolerateInvalid flag is used for a group")
}

func TestExecutionStep_valid_PassesWhenExactlyOneOfSingleOrGroupIsSet(t *testing.T) {
	correct := []ExecutionStep{
		{single: &single{}},
		{group: &group{}},
	}

	for _, step := range correct {
		require.True(t, step.valid())
	}

	wrong := []ExecutionStep{
		{},                                   // neither single nor group set
		{single: &single{}, group: &group{}}, // both single and group set
	}

	for _, step := range wrong {
		require.False(t, step.valid())
	}
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

// TestExecutionStep_decode_RejectsDeeplyNestedEncoding ensures that decoding a
// maliciously deep execution step is rejected up front by the nesting-depth
// guard, rather than recursing through the whole structure (in rlp.Decode and
// fromEncodingV1) and exhausting the goroutine stack. The test reaching this
// assertion without a stack-overflow crash is itself part of what is verified.
func TestExecutionStep_decode_RejectsDeeplyNestedEncoding(t *testing.T) {
	require := require.New(t)

	// Encode a valid leaf step to use as the innermost element.
	leaf := NewTxStep(TxReference{From: common.Address{1}, Hash: common.Hash{2}})

	// Check 1000 and 1 million nested steps, both of which exceed the
	// MaxGroupNestingDepth limit and should be rejected by the depth guard.
	// The 1 million case lead to a stack overflow before the guard was
	// implemented, so is included to ensure the guard is effective.
	for _, size := range []int{1000, 1_000_000} {
		nested := leaf
		for range size {
			nested = NewAllOfStep(nested)
		}
		var buf bytes.Buffer
		require.NoError(nested.encode(&buf))

		var s ExecutionStep
		require.ErrorContains(s.decode(bytes.NewReader(buf.Bytes())), "nesting depth")
	}
}

// TestExecutionStep_decode_AcceptsEncodingAtNestingLimit ensures the decode-time
// depth guard never rejects a plan that satisfies the consensus nesting rule
// (MaxGroupNestingDepth). A step nested exactly to that limit must still decode,
// validate, and round-trip unchanged.
func TestExecutionStep_decode_AcceptsEncodingAtNestingLimit(t *testing.T) {
	require := require.New(t)

	step := NewTxStep(TxReference{From: common.Address{1}, Hash: common.Hash{2}})
	for range MaxGroupNestingDepth {
		step = NewAllOfStep(step)
	}

	// The plan is valid under the precise consensus rule.
	require.NoError(validateStep(step))

	var buf bytes.Buffer
	require.NoError(step.encode(&buf))
	encoded := buf.Bytes()

	var decoded ExecutionStep
	require.NoError(decoded.decode(bytes.NewReader(encoded)))
	require.Equal(step, decoded)
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

func TestCheckRlpNestingDepth_ReachingMaxDepth_ReturnsError(t *testing.T) {
	require.ErrorContains(t,
		checkRlpNestingDepth(nil, maxStepEncodingRlpDepth+1),
		"encoded execution step exceeds maximum nesting depth",
	)
}

func TestCheckRlpNestingDepth_EmptyInput_ReturnsAnError(t *testing.T) {
	stream := rlp.NewStream(bytes.NewReader([]byte("")), 0)
	require.ErrorContains(t,
		checkRlpNestingDepth(stream, maxStepEncodingRlpDepth),
		"EOF",
	)
}
