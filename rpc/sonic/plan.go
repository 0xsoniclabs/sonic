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

package sonic

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// errNoStepOrGroup is the error of a step that is neither a leaf nor a group,
// which an empty group decodes to.
var errNoStepOrGroup = errors.New("invalid execution plan level: must have either executionStep or group")

// blockRange is the JSON form of the blocks a bundle may run in.
type blockRange struct {
	First  hexutil.Uint64 `json:"first"`
	Length hexutil.Uint64 `json:"length"`
}

func (r blockRange) toBundle() bundle.BlockRange {
	return bundle.BlockRange{First: uint64(r.First), Length: uint64(r.Length)}
}

// group is the JSON form of a group of steps of a plan or a proposal. A step
// is a leaf, a group, or nil for an empty group.
type group struct {
	TolerateFailures bool  `json:"tolerateFailures,omitempty"`
	OneOf            bool  `json:"oneOf,omitempty"`
	Steps            []any `json:"steps"`
}

// planStep is the JSON form of a transaction of an execution plan.
type planStep struct {
	TolerateFailed  bool           `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool           `json:"tolerateInvalid,omitempty"`
	From            common.Address `json:"from"`
	Hash            common.Hash    `json:"hash"`
}

// executionPlan is the JSON form of a bundle's execution plan: the blocks it
// may run in and its root group.
type executionPlan struct {
	BlockRange blockRange `json:"blockRange"`
	group
}

func (p *executionPlan) UnmarshalJSON(data []byte) error {
	var raw struct {
		BlockRange       blockRange        `json:"blockRange,omitempty"`
		TolerateFailures bool              `json:"tolerateFailures,omitempty"`
		OneOf            bool              `json:"oneOf,omitempty"`
		Steps            []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	steps, err := decodeSteps[planStep](raw.Steps, 1)
	if err != nil {
		return err
	}
	*p = executionPlan{BlockRange: raw.BlockRange, group: group{TolerateFailures: raw.TolerateFailures, OneOf: raw.OneOf, Steps: steps}}
	return nil
}

// decodeSteps decodes the steps of a group at a nesting depth, the root
// group's being 0: an object with steps is a group, another one a Leaf. An
// empty group decodes to nil. The depth is checked before a step is parsed,
// so that deeply nested input is refused before its cost grows.
func decodeSteps[Leaf any](raw []json.RawMessage, depth int) ([]any, error) {
	steps := make([]any, len(raw))
	for i, data := range raw {
		step, err := decodeStep[Leaf](data, depth)
		if err != nil {
			return nil, err
		}
		steps[i] = step
	}
	return steps, nil
}

func decodeStep[Leaf any](data []byte, depth int) (any, error) {
	if depth > bundle.MaxGroupNestingDepth {
		return nil, fmt.Errorf("execution plan exceeds maximum nesting depth of %d", bundle.MaxGroupNestingDepth)
	}
	var probe struct {
		Steps *json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	if probe.Steps == nil {
		var leaf Leaf
		if err := json.Unmarshal(data, &leaf); err != nil {
			return nil, err
		}
		return leaf, nil
	}
	var raw struct {
		TolerateFailures bool              `json:"tolerateFailures"`
		OneOf            bool              `json:"oneOf"`
		Steps            []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	steps, err := decodeSteps[Leaf](raw.Steps, depth+1)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, nil
	}
	return group{TolerateFailures: raw.TolerateFailures, OneOf: raw.OneOf, Steps: steps}, nil
}

// newExecutionPlan forms a bundle's execution plan.
func newExecutionPlan(plan bundle.ExecutionPlan) (executionPlan, error) {
	builder := &planBuilder{}
	if err := plan.Root.Accept(builder); err != nil {
		return executionPlan{}, fmt.Errorf("failed to convert execution plan: %w", err)
	}
	return executionPlan{
		BlockRange: blockRange{First: hexutil.Uint64(plan.Range.First), Length: hexutil.Uint64(plan.Range.Length)},
		group:      builder.root,
	}, nil
}

// planBuilder forms the groups of an execution plan as it visits them. The
// root of a plan the module makes is a group.
type planBuilder struct {
	open []*group
	root group
}

func (b *planBuilder) Step(flags bundle.ExecutionFlags, reference bundle.TxReference) error {
	current := b.open[len(b.open)-1]
	current.Steps = append(current.Steps, planStep{
		TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
		TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
		From:            reference.From,
		Hash:            reference.Hash,
	})
	return nil
}

func (b *planBuilder) BeginGroup(oneOf bool, tolerateFailed bool) {
	b.open = append(b.open, &group{OneOf: oneOf, TolerateFailures: tolerateFailed})
}

func (b *planBuilder) EndGroup() {
	closed := b.open[len(b.open)-1]
	b.open = b.open[:len(b.open)-1]
	if len(b.open) == 0 {
		b.root = *closed
		return
	}
	parent := b.open[len(b.open)-1]
	parent.Steps = append(parent.Steps, *closed)
}

// toBundle is the bundle's execution plan the form names.
func (p executionPlan) toBundle() (bundle.ExecutionPlan, error) {
	root, err := p.group.toBundle()
	if err != nil {
		return bundle.ExecutionPlan{}, fmt.Errorf("failed to convert execution plan: %w", err)
	}
	return bundle.ExecutionPlan{Range: p.BlockRange.toBundle(), Root: root, Period: bundle.MakeUnrestrictedTimePeriod()}, nil
}

func (g group) toBundle() (bundle.ExecutionStep, error) {
	steps := make([]bundle.ExecutionStep, len(g.Steps))
	for i, step := range g.Steps {
		var err error
		switch step := step.(type) {
		case planStep:
			steps[i] = bundle.NewTxStep(bundle.TxReference{From: step.From, Hash: step.Hash}).
				WithFlags(executionFlags(step.TolerateFailed, step.TolerateInvalid))
		case group:
			steps[i], err = step.toBundle()
		default:
			err = errNoStepOrGroup
		}
		if err != nil {
			return bundle.ExecutionStep{}, fmt.Errorf("invalid execution plan level: %w", err)
		}
	}
	return newGroupStep(g, steps), nil
}

func newGroupStep(g group, steps []bundle.ExecutionStep) bundle.ExecutionStep {
	step := bundle.NewGroupStep(g.OneOf, steps...)
	if g.TolerateFailures {
		step = step.WithFlags(bundle.EF_TolerateFailed)
	}
	return step
}

func executionFlags(tolerateFailed, tolerateInvalid bool) bundle.ExecutionFlags {
	flags := bundle.EF_Default
	if tolerateFailed {
		flags |= bundle.EF_TolerateFailed
	}
	if tolerateInvalid {
		flags |= bundle.EF_TolerateInvalid
	}
	return flags
}
