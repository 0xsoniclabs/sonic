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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// MaxNumEstimableTransactions is the maximum number of transactions
// that can be included in a bundle for gas estimation.
// The algorithm to estimate bundle gas is O(n^2),
// therefore an upper bound is introduced.
const MaxNumEstimableTransactions = 16

type RPCExecutionStep struct {
	From common.Address `json:"from"`
	Hash common.Hash    `json:"hash"`
}

type RPCExecutionPlan struct {
	Flags    bundle.ExecutionFlags `json:"flags"`
	Steps    []RPCExecutionStep    `json:"steps"`
	Earliest rpc.BlockNumber       `json:"earliest"`
	Latest   rpc.BlockNumber       `json:"latest"`
}

// NewRPCExecutionPlan converts a bundle.ExecutionPlan to an RPCExecutionPlan for JSON-RPC responses.
func NewRPCExecutionPlan(plan bundle.ExecutionPlan) RPCExecutionPlan {
	steps := make([]RPCExecutionStep, len(plan.Steps))
	for i, step := range plan.Steps {
		steps[i] = RPCExecutionStep{
			From: step.From,
			Hash: step.Hash,
		}
	}

	return RPCExecutionPlan{
		Flags:    plan.Flags,
		Steps:    steps,
		Earliest: rpc.BlockNumber(plan.Range.Earliest),
		Latest:   rpc.BlockNumber(plan.Range.Latest),
	}
}
