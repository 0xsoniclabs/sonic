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
	plan := bundle.ExecutionPlan{
		Steps: []bundle.ExecutionStep{
			{
				From: common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Hash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
			},
			{
				From: common.HexToAddress("0x3333333333333333333333333333333333333333"),
				Hash: common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
			},
		},
		Flags: bundle.EF_TolerateInvalid | bundle.EF_OneOf,
		Range: bundle.BlockRange{
			Earliest: 100,
			Latest:   200,
		},
	}

	rpcPlan := NewRPCExecutionPlan(plan)

	require.Equal(t, plan.Flags, rpcPlan.Flags)
	require.Equal(t, rpc.BlockNumber(plan.Range.Earliest), rpcPlan.Earliest)
	require.Equal(t, rpc.BlockNumber(plan.Range.Latest), rpcPlan.Latest)
	require.Len(t, rpcPlan.Steps, len(plan.Steps))
	for i, step := range plan.Steps {
		require.Equal(t, step.From, rpcPlan.Steps[i].From)
		require.Equal(t, step.Hash, rpcPlan.Steps[i].Hash)
	}
}
