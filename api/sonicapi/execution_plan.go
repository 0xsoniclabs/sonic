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
	"encoding/json"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RPCExecutionPlanComposable is the JSON-serializable representation of the execution plan
// that can be returned by the API. It is designed to be easily convertible to and from the internal bundle.ExecutionPlan structure, while also being flexible enough to accommodate
// different representations of the execution steps (e.g. for proposals or for actual execution).
//
// An example of the JSON structure of an RPCExecutionPlanComposable could be:
//
//	{
//	  "blockRange": {
//	    "first": 12345678,
//	    "length": 100
//	  },
//	  "steps": [
//	    {
//	      "from": "0xabc123abc123abc123abc123abc123abc123abc1",
//	      "hash": "0xdef456def456def456def456def456def456def456def456def456def456def4",
//	    },
//	    {
//	      "oneOf": true,
//	      "steps": [
//	        {
//	          "from": "0xabc123abc123abc123abc123abc123abc123abc1",
//	          "hash": "0xdef456def456def456def456def456def456def456def456def456def456def4",
//	          "tolerateFailed": true,
//	        },
//	        {
//	          "from": "0xabc123abc123abc123abc123abc123abc123abc1",
//	          "hash": "0xdef456def456def456def456def456def456def456def456def456def456def4",
//	          "tolerateInvalid": true
//	        }
//	      ]
//	    }
//	  ]
//	}
type RPCExecutionPlanComposable struct {
	BlockRange RPCRange `json:"blockRange"`
	RPCExecutionPlanGroup
}

type RPCExecutionPlanGroup struct {
	TolerateFailures bool  `json:"tolerateFailures,omitempty"`
	OneOf            bool  `json:"oneOf,omitempty"`
	Steps            []any `json:"steps"`
}

type RPCExecutionStepComposable struct {
	TolerateFailed  bool           `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool           `json:"tolerateInvalid,omitempty"`
	From            common.Address `json:"from"`
	Hash            common.Hash    `json:"hash"`
}

// RPCRange represents the block range for which the execution plan is valid.
type RPCRange struct {
	First  hexutil.Uint64 `json:"first"`
	Length hexutil.Uint64 `json:"length"`
}

// NewRPCExecutionPlanComposable converts a bundle.ExecutionPlan to an RPCExecutionPlan that can be returned by the API.
func NewRPCExecutionPlanComposable(plan bundle.ExecutionPlan) (RPCExecutionPlanComposable, error) {
	visitor := makeExecutionPlanVisitor(
		func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (any, error) {
			return RPCExecutionStepComposable{
				TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
				TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
				From:            txRef.From,
				Hash:            txRef.Hash,
			}, nil
		})

	if err := plan.Root.Accept(visitor); err != nil {
		return RPCExecutionPlanComposable{},
			fmt.Errorf("failed to convert execution plan: %w", err)
	}

	return RPCExecutionPlanComposable{
		BlockRange:            fromBundleRange(plan.Range),
		RPCExecutionPlanGroup: visitor.result,
	}, nil
}

func ToBundleExecutionPlan(rpcPlan RPCExecutionPlanComposable) (bundle.ExecutionPlan, error) {
	root, err := toBundleExecutionGroup(rpcPlan.RPCExecutionPlanGroup)
	if err != nil {
		return bundle.ExecutionPlan{}, fmt.Errorf("failed to convert execution plan: %w", err)
	}

	return bundle.ExecutionPlan{
		Range:  rpcPlan.BlockRange.toBundleBlockRange(),
		Root:   root,
		Period: bundle.MakeUnrestrictedTimePeriod(),
	}, nil
}

func toBundleExecutionPlanLevel(level any) (bundle.ExecutionStep, error) {
	switch l := level.(type) {
	case RPCExecutionStepComposable:
		ref := bundle.NewTxStep(bundle.TxReference{
			From: l.From,
			Hash: l.Hash,
		})
		flags := bundle.EF_Default
		if l.TolerateFailed {
			flags |= bundle.EF_TolerateFailed
		}
		if l.TolerateInvalid {
			flags |= bundle.EF_TolerateInvalid
		}
		return ref.WithFlags(flags), nil

	case RPCExecutionPlanGroup:
		return toBundleExecutionGroup(l)
	}
	return bundle.ExecutionStep{},
		fmt.Errorf("invalid execution plan level: must have either executionStep or group")
}

func toBundleExecutionGroup(l RPCExecutionPlanGroup) (bundle.ExecutionStep, error) {
	steps := make([]bundle.ExecutionStep, len(l.Steps))
	for i, stepLevel := range l.Steps {
		step, err := toBundleExecutionPlanLevel(stepLevel)
		if err != nil {
			return bundle.ExecutionStep{}, fmt.Errorf("invalid execution plan level: %w", err)
		}
		steps[i] = step
	}

	// Single child without flags does not need to be in an extra group
	if !l.TolerateFailures && len(steps) == 1 {
		return steps[0], nil
	}

	group := bundle.NewGroupStep(l.OneOf, steps...)
	if l.TolerateFailures {
		group = group.WithFlags(bundle.EF_TolerateFailed)
	}
	return group, nil
}

func (t *RPCExecutionPlanComposable) UnmarshalJSON(data []byte) error {
	var raw struct {
		BlockRange       RPCRange          `json:"blockRange,omitempty"`
		TolerateFailures bool              `json:"tolerateFailures,omitempty"`
		OneOf            bool              `json:"oneOf,omitempty"`
		Steps            []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.BlockRange = raw.BlockRange
	t.OneOf = raw.OneOf
	t.TolerateFailures = raw.TolerateFailures
	t.Steps = make([]any, len(raw.Steps))
	for i, rawStep := range raw.Steps {
		step, empty, err := unmarshalBundleGroup[RPCExecutionStepComposable](rawStep)
		if err != nil {
			return err
		}
		if !empty {
			t.Steps[i] = step
		}
	}
	return nil
}

// makeExecutionPlanVisitor creates a new instance of toJsonExecutionPlanVisitor with the provided toLeaf function.
// This visitor can be used to convert a bundle.ExecutionPlan into a json capable
// structure where the leaf nodes are customizable.
// This allows to create the same structure for different use cases, such as
// an execution plan or a proposal of a plan where all the transactions are txArguments
func makeExecutionPlanVisitor(
	toLeaf func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (any, error),
) *toJsonExecutionPlanVisitor {
	return &toJsonExecutionPlanVisitor{
		toLeaf: toLeaf,
	}
}

type toJsonExecutionPlanVisitor struct {
	toLeaf     func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (any, error)
	result     RPCExecutionPlanGroup
	groupStack []*RPCExecutionPlanGroup
}

func (v *toJsonExecutionPlanVisitor) Step(flags bundle.ExecutionFlags, txRef bundle.TxReference) error {
	leaf, err := v.toLeaf(flags, txRef)
	if err != nil {
		return fmt.Errorf("failed to convert execution step: %w", err)
	}

	if len(v.groupStack) == 0 {
		v.result.Steps = append(v.result.Steps, leaf)
	} else {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, leaf)
	}

	return nil
}

func (v *toJsonExecutionPlanVisitor) BeginGroup(oneOf bool, tolerateFailed bool) {
	group := RPCExecutionPlanGroup{
		OneOf:            oneOf,
		TolerateFailures: tolerateFailed,
	}
	v.groupStack = append(v.groupStack, &group)
}

func (v *toJsonExecutionPlanVisitor) EndGroup() {
	closedGroup := v.groupStack[len(v.groupStack)-1]
	v.groupStack = v.groupStack[:len(v.groupStack)-1]

	if len(v.groupStack) > 0 {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, *closedGroup)
	} else {
		v.result.Steps = append(v.result.Steps, *closedGroup)
	}
}

func (r RPCRange) toBundleBlockRange() bundle.BlockRange {
	return bundle.BlockRange{
		First:  uint64(r.First),
		Length: uint64(r.Length),
	}
}

func fromBundleRange(r bundle.BlockRange) RPCRange {
	return RPCRange{
		First:  hexutil.Uint64(r.First),
		Length: hexutil.Uint64(r.Length),
	}
}
