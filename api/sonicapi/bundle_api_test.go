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

package sonicapi

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestNewRPCExecutionPlan(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	addr3 := common.Address{3}
	hash1 := common.Hash{0x01}
	hash2 := common.Hash{0x02}
	hash3 := common.Hash{0x03}

	ref1 := bundle.TxReference{From: addr1, Hash: hash1}
	ref2 := bundle.TxReference{From: addr2, Hash: hash2}
	ref3 := bundle.TxReference{From: addr3, Hash: hash3}

	tests := []struct {
		name         string
		plan         bundle.ExecutionPlan
		wantSteps    []RPCExecutionStep
		wantFlags    bundle.ExecutionFlags
		wantEarliest rpc.BlockNumber
		wantLatest   rpc.BlockNumber
	}{
		{
			name: "single tx step, default flags",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 1, Latest: 100},
				Root:  bundle.NewTxStep(ref1),
			},
			wantSteps:    []RPCExecutionStep{{From: addr1, Hash: hash1}},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 1,
			wantLatest:   100,
		},
		{
			name: "two tx steps in AllOf group",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 5, Latest: 50},
				Root:  bundle.NewAllOfStep(bundle.NewTxStep(ref1), bundle.NewTxStep(ref2)),
			},
			wantSteps: []RPCExecutionStep{
				{From: addr1, Hash: hash1},
				{From: addr2, Hash: hash2},
			},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 5,
			wantLatest:   50,
		},
		{
			name: "three tx steps in AllOf group",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 0, Latest: 1000},
				Root: bundle.NewAllOfStep(
					bundle.NewTxStep(ref1),
					bundle.NewTxStep(ref2),
					bundle.NewTxStep(ref3),
				),
			},
			wantSteps: []RPCExecutionStep{
				{From: addr1, Hash: hash1},
				{From: addr2, Hash: hash2},
				{From: addr3, Hash: hash3},
			},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 0,
			wantLatest:   1000,
		},
		{
			name: "TolerateFailed flag preserved",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 10, Latest: 20},
				Root:  bundle.NewTxStep(ref1).WithFlags(bundle.EF_TolerateFailed),
			},
			wantSteps:    []RPCExecutionStep{{From: addr1, Hash: hash1}},
			wantFlags:    bundle.EF_TolerateFailed,
			wantEarliest: 10,
			wantLatest:   20,
		},
		{
			name: "TolerateInvalid flag preserved",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 10, Latest: 20},
				Root:  bundle.NewTxStep(ref1).WithFlags(bundle.EF_TolerateInvalid),
			},
			wantSteps:    []RPCExecutionStep{{From: addr1, Hash: hash1}},
			wantFlags:    bundle.EF_TolerateInvalid,
			wantEarliest: 10,
			wantLatest:   20,
		},
		{
			name: "both tolerate flags preserved",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 1, Latest: 5},
				Root:  bundle.NewTxStep(ref1).WithFlags(bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid),
			},
			wantSteps:    []RPCExecutionStep{{From: addr1, Hash: hash1}},
			wantFlags:    bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid,
			wantEarliest: 1,
			wantLatest:   5,
		},
		{
			name: "block range boundaries preserved",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 42, Latest: 99},
				Root:  bundle.NewTxStep(ref1),
			},
			wantSteps:    []RPCExecutionStep{{From: addr1, Hash: hash1}},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 42,
			wantLatest:   99,
		},
		{
			name: "nested groups return steps in referenced order",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 1, Latest: 10},
				Root: bundle.NewAllOfStep(
					bundle.NewAllOfStep(
						bundle.NewTxStep(ref1),
						bundle.NewTxStep(ref2),
					),
					bundle.NewTxStep(ref3),
				),
			},
			wantSteps: []RPCExecutionStep{
				{From: addr1, Hash: hash1},
				{From: addr2, Hash: hash2},
				{From: addr3, Hash: hash3},
			},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 1,
			wantLatest:   10,
		},
		{
			name: "step From and Hash fields correctly mapped",
			plan: bundle.ExecutionPlan{
				Range: bundle.BlockRange{Earliest: 0, Latest: 0},
				Root:  bundle.NewTxStep(ref2),
			},
			wantSteps:    []RPCExecutionStep{{From: addr2, Hash: hash2}},
			wantFlags:    bundle.EF_Default,
			wantEarliest: 0,
			wantLatest:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRPCExecutionPlan(tt.plan)
			require.Equal(t, tt.wantSteps, got.Steps)
			require.Equal(t, tt.wantFlags, got.Flags)
			require.Equal(t, tt.wantEarliest, got.Earliest)
			require.Equal(t, tt.wantLatest, got.Latest)
		})
	}
}
