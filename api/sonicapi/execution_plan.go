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
//		{
//	   	"blockRange":{
//				"earliest":"0xa",
//				"latest":"0x15"
//			},
//		    "root":{
//				"group":{
//					"oneOf":true,
//					"steps":[
//						{"group":{
//							"steps":[
//								{"single":{
//									"from":"0x0100000000000000000000000000000000000000",
//									"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
//								}},
//								{"single":{
//									"from":"0x0300000000000000000000000000000000000000",
//									"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
//								}}
//							]
//						}},
//						{"group":{
//							"steps":[
//								{"single":{
//									"from":"0x0300000000000000000000000000000000000000",
//									"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
//								}},
//								{"single":{
//									"from":"0x0100000000000000000000000000000000000000",
//									"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
//								}}
//							]
//						}}
//					]
//				}
//			}
//		}
type RPCExecutionPlanComposable struct {
	BlockRange RPCRange              `json:"blockRange"`
	Root       RPCExecutionPlanLevel `json:"root"`
}

// RPCExecutionPlanGroup represents a group of execution steps in the JSON-serializable execution plan.
type RPCExecutionPlanGroup struct {
	TolerateFailures bool                    `json:"tolerateFailures"`
	OneOf            bool                    `json:"oneOf"`
	Steps            []RPCExecutionPlanLevel `json:"steps,omitempty"`
}

// RPCExecutionStepComposable represents a single execution step in the JSON-serializable execution plan.
type RPCExecutionStepComposable struct {
	TolerateFailed  bool           `json:"tolerateFailed"`
	TolerateInvalid bool           `json:"tolerateInvalid"`
	From            common.Address `json:"from"`
	Hash            common.Hash    `json:"hash"`
}

// RPCExecutionPlanLevel represents a level in the execution plan, which can be either a single step or a group of steps.
type RPCExecutionPlanLevel struct {
	Single *RPCExecutionStepComposable `json:"single,omitempty"`
	Group  *RPCExecutionPlanGroup      `json:"group,omitempty"`
}

// RPCRange represents the block range for which the execution plan is valid.
type RPCRange struct {
	Earliest hexutil.Uint64 `json:"earliest"`
	Latest   hexutil.Uint64 `json:"latest"`
}

// NewRPCExecutionPlanComposable converts a bundle.ExecutionPlan to an RPCExecutionPlan that can be returned by the API.
func NewRPCExecutionPlanComposable(plan bundle.ExecutionPlan) RPCExecutionPlanComposable {

	visitor := newToJsonExecutionPlanVisitor()
	plan.Root.Visit(visitor)

	var root RPCExecutionPlanLevel
	if len(visitor.levelStack) > 0 {
		root = visitor.levelStack[0]
	}

	return RPCExecutionPlanComposable{
		BlockRange: RPCRange{
			Earliest: hexutil.Uint64(plan.Range.Earliest),
			Latest:   hexutil.Uint64(plan.Range.Latest),
		},
		Root: root,
	}
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

func toBundleExecutionPlanLevel(rpcLevel RPCExecutionPlanLevel) (bundle.ExecutionStep, error) {
	if rpcLevel.Single != nil && rpcLevel.Group != nil {
		return bundle.ExecutionStep{},
			fmt.Errorf("invalid execution plan level: cannot have both single and group")
	}

	if rpcLevel.Single != nil {
		ref := bundle.NewTxStep(bundle.TxReference{
			From: rpcLevel.Single.From,
			Hash: rpcLevel.Single.Hash,
		})
		flags := bundle.ExecutionFlags(0)
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

// toJsonExecutionPlanVisitor is an implementation of the ExecutionPlanVisitor interface
type toJsonExecutionPlanVisitor struct {
	levelStack []RPCExecutionPlanLevel
	groupStack []*RPCExecutionPlanGroup
}

// newRPCExecutionPlanVisitor creates a new instance of toJsonExecutionPlanVisitor.
func newToJsonExecutionPlanVisitor() *toJsonExecutionPlanVisitor {
	return &toJsonExecutionPlanVisitor{}
}

func (v *toJsonExecutionPlanVisitor) Step(flags bundle.ExecutionFlags, txRef bundle.TxReference) {
	step := RPCExecutionStepComposable{
		TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
		TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
		From:            txRef.From,
		Hash:            txRef.Hash,
	}

	if len(v.groupStack) > 0 {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, RPCExecutionPlanLevel{
			Single: &step,
		})
	} else {
		v.levelStack = append(v.levelStack, RPCExecutionPlanLevel{
			Single: &step,
		})
	}
}

func (v *toJsonExecutionPlanVisitor) BeginGroup(oneOf bool, tolerateFailed bool) {
	group := &RPCExecutionPlanGroup{
		OneOf:            oneOf,
		TolerateFailures: tolerateFailed,
	}
	v.groupStack = append(v.groupStack, group)
}

func (v *toJsonExecutionPlanVisitor) EndGroup() {
	closedGroup := v.groupStack[len(v.groupStack)-1]
	v.groupStack = v.groupStack[:len(v.groupStack)-1]

	if len(v.groupStack) > 0 {
		currentGroup := v.groupStack[len(v.groupStack)-1]
		currentGroup.Steps = append(currentGroup.Steps, RPCExecutionPlanLevel{
			Group: closedGroup,
		})
	} else {
		v.levelStack = append(v.levelStack, RPCExecutionPlanLevel{
			Group: closedGroup,
		})
	}
}
