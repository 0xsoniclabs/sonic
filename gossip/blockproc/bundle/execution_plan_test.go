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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestExecutionPlan_Hash_ComputesDeterministicHash(t *testing.T) {

	step1 := ExecutionStep{
		From: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Hash: common.Hash{0x01},
	}
	step2 := ExecutionStep{
		From: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		Hash: common.Hash{0x02},
	}

	tests := map[string]ExecutionPlan{
		"empty plan": {},
		"plan with transactions": {
			Steps: []ExecutionStep{step1, step2},
		},
		"plan with flag 1": {
			Steps: []ExecutionStep{step1},
			Flags: 0x1,
		},
		"plan with flag 2": {
			Steps: []ExecutionStep{step1},
			Flags: 0x2,
		},
		"plan with flag 3": {
			Steps: []ExecutionStep{step1},
			Flags: 0x3,
		},
	}

	seenHashes := make(map[common.Hash]struct{})
	for name, executionPlan := range tests {
		t.Run(name, func(t *testing.T) {

			transactions := make([]any, len(executionPlan.Steps))
			for i, step := range executionPlan.Steps {
				transactions[i] = []any{step.From, step.Hash}
			}
			manualSerialize := []any{
				transactions,
				executionPlan.Flags,
				executionPlan.Range,
			}

			hasher := crypto.NewKeccakState()
			require.NoError(t, rlp.Encode(hasher, manualSerialize))
			computed := common.BytesToHash(hasher.Sum(nil))

			require.Equal(t, executionPlan.Hash(), computed)
			require.NotContains(t, seenHashes, computed, "hash should be unique for different plans")
			seenHashes[computed] = struct{}{}
		})
	}
}
