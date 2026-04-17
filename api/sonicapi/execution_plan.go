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
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RPCExecutionPlanComposable is the JSON-serializable representation of the execution plan
// that is returned by the API. It is designed to be easily serializable to JSON
// and human-readable for integration purposes.
//
// An example of the JSON representation of an execution plan is as follows:
//
//	{
//	   	"blockRange":{
//				"earliest":"0xa",
//				"latest":"0x15"
//		},
//		"root":{
//			"group":{
//				"oneOf":true,
//				"steps":[
//					{"group":{
//						"steps":[
//							{"single":{
//								"from":"0x0100000000000000000000000000000000000000",
//								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
//							}},
//							{"single":{
//								"from":"0x0300000000000000000000000000000000000000",
//								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
//							}}
//						]
//					}},
//					{"group":{
//						"steps":[
//							{"single":{
//								"from":"0x0300000000000000000000000000000000000000",
//								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
//							}},
//							{"single":{
//								"from":"0x0100000000000000000000000000000000000000",
//								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
//							}}
//						]
//					}}
//				]
//			}
//		}
//	}
type RPCExecutionPlanComposable struct {
	BlockRange RPCRange                                          `json:"blockRange"`
	Root       RPCExecutionPlanLevel[RPCExecutionStepComposable] `json:"root"`
}

// RPCExecutionPlanGroup represents a group of execution steps in the JSON-serializable execution plan.
type RPCExecutionPlanGroup[T any] struct {
	TolerateFailures bool                       `json:"tolerateFailures,omitempty"`
	OneOf            bool                       `json:"oneOf,omitempty"`
	Steps            []RPCExecutionPlanLevel[T] `json:"steps"`
}

// RPCExecutionStepComposable represents a single execution step in the JSON-serializable execution plan.
type RPCExecutionStepComposable struct {
	TolerateFailed  bool           `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool           `json:"tolerateInvalid,omitempty"`
	From            common.Address `json:"from"`
	Hash            common.Hash    `json:"hash"`
}

// RPCExecutionPlanLevel represents a level in the execution plan, which can be either a single step or a group of steps.
type RPCExecutionPlanLevel[T any] struct {
	Single *T                        `json:"single,omitempty"`
	Group  *RPCExecutionPlanGroup[T] `json:"group,omitempty"`
}

// RPCRange represents the block range for which the execution plan is valid.
type RPCRange struct {
	Earliest hexutil.Uint64 `json:"earliest"`
	Latest   hexutil.Uint64 `json:"latest"`
}

// NewRPCExecutionPlanComposable converts a bundle.ExecutionPlan to an RPCExecutionPlan that can be returned by the API.
func NewRPCExecutionPlanComposable(plan bundle.ExecutionPlan) (RPCExecutionPlanComposable, error) {

	visitor := makeExecutionPlanVisitor(
		func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (*RPCExecutionStepComposable, error) {
			return &RPCExecutionStepComposable{
				TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
				TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
				From:            txRef.From,
				Hash:            txRef.Hash,
			}, nil
		})

	// because the conversion bundle.TxReference -> RPCExecutionStepComposable cannot fail, we can ignore the error here
	_ = plan.Root.Accept(visitor)

	return RPCExecutionPlanComposable{
		BlockRange: RPCRange{
			Earliest: hexutil.Uint64(plan.Range.Earliest),
			Latest:   hexutil.Uint64(plan.Range.Latest),
		},
		Root: visitor.result,
	}, nil
}

func toBundleExecutionPlan(rpcPlan RPCExecutionPlanComposable) (bundle.ExecutionPlan, error) {

	root, err := toBundleExecutionPlanLevel(rpcPlan.Root)
	if err != nil {
		return bundle.ExecutionPlan{}, fmt.Errorf("invalid execution plan root: %w", err)
	}

	return bundle.ExecutionPlan{
		Range: bundle.BlockRange{
			Earliest: uint64(rpcPlan.BlockRange.Earliest),
			Latest:   uint64(rpcPlan.BlockRange.Latest),
		},
		Root: root,
	}, nil
}

func toBundleExecutionPlanLevel(rpcLevel RPCExecutionPlanLevel[RPCExecutionStepComposable]) (bundle.ExecutionStep, error) {
	if rpcLevel.Single != nil && rpcLevel.Group != nil {
		return bundle.ExecutionStep{},
			fmt.Errorf("invalid execution plan level: cannot have both single and group")
	}

	if rpcLevel.Single != nil {
		ref := bundle.NewTxStep(bundle.TxReference{
			From: rpcLevel.Single.From,
			Hash: rpcLevel.Single.Hash,
		})
		flags := bundle.EF_Default
		if rpcLevel.Single.TolerateFailed {
			flags |= bundle.EF_TolerateFailed
		}
		if rpcLevel.Single.TolerateInvalid {
			flags |= bundle.EF_TolerateInvalid
		}
		return ref.WithFlags(flags), nil
	} else if rpcLevel.Group != nil {
		steps := make([]bundle.ExecutionStep, len(rpcLevel.Group.Steps))
		for i, stepLevel := range rpcLevel.Group.Steps {
			step, err := toBundleExecutionPlanLevel(stepLevel)
			if err != nil {
				return bundle.ExecutionStep{}, fmt.Errorf("invalid execution plan level: %w", err)
			}
			steps[i] = step
		}

		group := bundle.NewGroupStep(rpcLevel.Group.OneOf, steps...)
		if rpcLevel.Group.TolerateFailures {
			group = group.WithFlags(bundle.EF_TolerateFailed)
		}
		return group, nil
	}
	return bundle.ExecutionStep{}, fmt.Errorf("invalid execution plan level: must have either single or group")
}

// makeExecutionPlanVisitor creates a new instance of toJsonExecutionPlanVisitor with the provided toLeaf function.
// This visitor can be used to convert a bundle.ExecutionPlan into a json capable
// structure where the leaf nodes are customizable.
// This allows to create the same structure for different use cases, such as
// an execution plan or a proposal of a plan where all the transactions are txArguments
func makeExecutionPlanVisitor[T any](toLeaf func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (*T, error)) *toJsonExecutionPlanVisitor[T] {
	return &toJsonExecutionPlanVisitor[T]{
		toLeaf: toLeaf,
	}
}

type toJsonExecutionPlanVisitor[T any] struct {
	toLeaf     func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (*T, error)
	result     RPCExecutionPlanLevel[T]
	groupStack []*RPCExecutionPlanGroup[T]
}

func (v *toJsonExecutionPlanVisitor[T]) Step(flags bundle.ExecutionFlags, txRef bundle.TxReference) error {
	leaf, err := v.toLeaf(flags, txRef)
	if err != nil {
		return fmt.Errorf("failed to convert execution step to JSON: %w", err)
	}
	level := RPCExecutionPlanLevel[T]{Single: leaf}
	if len(v.groupStack) > 0 {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, level)
	} else {
		v.result = level
	}
	return nil
}

func (v *toJsonExecutionPlanVisitor[T]) BeginGroup(oneOf bool, tolerateFailed bool) {
	group := &RPCExecutionPlanGroup[T]{
		OneOf:            oneOf,
		TolerateFailures: tolerateFailed,
	}
	v.groupStack = append(v.groupStack, group)
}

func (v *toJsonExecutionPlanVisitor[T]) EndGroup() {
	closedGroup := v.groupStack[len(v.groupStack)-1]
	v.groupStack = v.groupStack[:len(v.groupStack)-1]

	if len(v.groupStack) > 0 {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, RPCExecutionPlanLevel[T]{
			Group: closedGroup,
		})
	} else {
		v.result = RPCExecutionPlanLevel[T]{Group: closedGroup}
	}
}
