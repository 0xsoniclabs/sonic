package bundle

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestExecutionPlan_Hash_ComputesDeterministicHash(t *testing.T) {

	step1 := ExecutionStep{txRef: &TxReference{
		From: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Hash: common.Hash{0x01},
	}}
	step2 := ExecutionStep{txRef: &TxReference{
		From: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		Hash: common.Hash{0x02},
	}}

	tests := map[string]ExecutionPlan{
		"empty plan": {},
		"plan with transactions": {
			Root: ExecutionStep{steps: []ExecutionStep{step1, step2}},
		},
		"plan with flag 1": {
			Root: ExecutionStep{flags: 0x1, steps: []ExecutionStep{step1}},
		},
		"plan with flag 2": {
			Root: ExecutionStep{flags: 0x2, steps: []ExecutionStep{step1}},
		},
		"plan with flag 3": {
			Root: ExecutionStep{flags: 0x3, steps: []ExecutionStep{step1}},
		},
		"plan with block range": {
			Root:  ExecutionStep{steps: []ExecutionStep{step1}},
			Range: BlockRange{Earliest: 10, Latest: 20},
		},
		"plan with different start": {
			Root:  ExecutionStep{steps: []ExecutionStep{step1}},
			Range: BlockRange{Earliest: 11, Latest: 20},
		},
		"plan with different end": {
			Root:  ExecutionStep{steps: []ExecutionStep{step1}},
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

func TestExecutionStep_EncodingAndDecodingAreAligned(t *testing.T) {
	tests := map[string]ExecutionStep{
		"zero value": {},
		"single transaction step": {
			flags: 0x1,
			txRef: &TxReference{
				From: common.Address{1, 2, 3},
				Hash: common.Hash{0x01},
			},
		},
		"allOf group": {
			flags: 0x2,
			oneOf: false,
			steps: []ExecutionStep{
				{
					flags: 0x3,
					txRef: &TxReference{
						From: common.Address{4, 5, 6},
						Hash: common.Hash{0x02},
					},
				},
				{
					flags: 0x4,
					txRef: &TxReference{
						From: common.Address{7, 8, 9},
						Hash: common.Hash{0x03},
					},
				},
			},
		},
		"oneOf group": {
			flags: 0x5,
			oneOf: true,
			steps: []ExecutionStep{
				{
					flags: 0x6,
					txRef: &TxReference{
						From: common.Address{10, 11, 12},
						Hash: common.Hash{0x04},
					},
				},
			},
		},
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

func TestExecutionStep_decode_FailsOnInvalidInput(t *testing.T) {
	data := []byte("invalid rlp data")
	var s ExecutionStep
	require.Error(t, s.decode(bytes.NewReader(data)))
}

func TestExecutionStep_String_PrintsReadableRepresentation(t *testing.T) {
	tests := map[string]struct {
		input ExecutionStep
		want  string
	}{
		"zero value": {
			input: ExecutionStep{},
			want:  "AllOf()",
		},
		"single transaction step": {
			input: ExecutionStep{
				txRef: &TxReference{
					From: common.Address{1, 2, 3},
				},
			},
			want: "A",
		},
		"single transaction tolerating invalid": {
			input: ExecutionStep{
				flags: EF_TolerateInvalid,
				txRef: &TxReference{
					From: common.Address{1, 2, 3},
				},
			},
			want: "Step[TolerateInvalid](A)",
		},
		"single transaction tolerating failed": {
			input: ExecutionStep{
				flags: EF_TolerateFailed,
				txRef: &TxReference{
					From: common.Address{1, 2, 3},
				},
			},
			want: "Step[TolerateFailed](A)",
		},
		"single transaction tolerating invalid and failed": {
			input: ExecutionStep{
				flags: EF_TolerateInvalid | EF_TolerateFailed,
				txRef: &TxReference{
					From: common.Address{1, 2, 3},
				},
			},
			want: "Step[TolerateInvalid|TolerateFailed](A)",
		},
		"allOf group": {
			input: ExecutionStep{
				oneOf: false,
				steps: []ExecutionStep{
					{txRef: &TxReference{From: common.Address{1}}},
					{txRef: &TxReference{From: common.Address{2}}},
				},
			},
			want: "AllOf(A,B)",
		},
		"oneOf group": {
			input: ExecutionStep{
				oneOf: true,
				steps: []ExecutionStep{
					{txRef: &TxReference{From: common.Address{1}}},
					{txRef: &TxReference{From: common.Address{2}}},
				},
			},
			want: "OneOf(A,B)",
		},
		"group with execution flags": {
			input: ExecutionStep{
				flags: EF_TolerateFailed | EF_TolerateInvalid,
				steps: []ExecutionStep{
					{txRef: &TxReference{From: common.Address{1}}},
					{txRef: &TxReference{From: common.Address{2}}},
				},
			},
			want: "Step[TolerateInvalid|TolerateFailed](AllOf(A,B))",
		},
		"repeated transactions": {
			input: ExecutionStep{
				steps: []ExecutionStep{
					{txRef: &TxReference{From: common.Address{1}}},
					{txRef: &TxReference{From: common.Address{2}}},
					{txRef: &TxReference{From: common.Address{1}}},
				},
			},
			want: "AllOf(A,B,A)",
		},
		"nested groups": {
			input: ExecutionStep{
				oneOf: true,
				steps: []ExecutionStep{
					{
						steps: []ExecutionStep{
							{txRef: &TxReference{From: common.Address{1}}},
							{txRef: &TxReference{From: common.Address{2}}},
						},
					},
					{
						steps: []ExecutionStep{
							{txRef: &TxReference{From: common.Address{1}}},
							{txRef: &TxReference{From: common.Address{3}}},
						},
					},
					{
						steps: []ExecutionStep{
							{txRef: &TxReference{From: common.Address{2}}},
							{txRef: &TxReference{From: common.Address{3}}},
						},
					},
				},
			},
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
