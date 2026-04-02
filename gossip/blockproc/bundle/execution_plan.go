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
	"errors"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// ExecutionPlan describes the plan for executing a bundle of transactions. It
// can be a composed hierarchy of groups, where each group can contain sub-groups
// or transactions. For each group, the execution semantic (e.g. AllOf or OneOf)
// can be defined independently. At the leaf level, the steps of the execution
// plan reference transactions to be executed, by specifying the sender and the
// hash of the transaction to be signed.
//
// The execution plan also defines a block range for which the plan is valid.
type ExecutionPlan struct {
	Range BlockRange
	Root  ExecutionStep
}

// Hash computes a deterministic hash of the execution plan, which can be used
// to uniquely identify the plan and verify its integrity. The hash is computed
// based on the structure of the execution steps, their flags, and the block
// range, ensuring that any change in the plan's content will result in a
// different hash.
func (p *ExecutionPlan) Hash() common.Hash {
	hasher := crypto.NewKeccakState()
	_ = p.encode(hasher)
	return common.BytesToHash(hasher.Sum(nil))
}

func (p *ExecutionPlan) encode(writer io.Writer) error {
	return errors.Join(
		p.Range.encode(writer),
		p.Root.encode(writer),
	)
}

func (p *ExecutionPlan) decode(reader io.Reader) error {
	return errors.Join(
		p.Range.decode(reader),
		p.Root.decode(reader),
	)
}

// TxReference represents a single step in an execution plan, referencing a
// transaction to be processed at this point of the plan.
type TxReference struct {
	// From is the sender of the transaction.
	From common.Address
	// Hash is the transaction hash to be signed (not the hash of the
	// transaction including its signature) where the bundle-only marker has
	// been removed.
	Hash common.Hash
}

// NewTxStep creates a step in the execution plan processing a single
// transaction identified by the given TxReference.
func NewTxStep(txRef TxReference) ExecutionStep {
	return ExecutionStep{txRef: &txRef}
}

// NewAllOfStep creates a step in the execution plan that requires all of the
// provided sub-steps to be successfully executed for the step to be considered
// successful.
func NewAllOfStep(subSteps ...ExecutionStep) ExecutionStep {
	return NewGroupStep(false, subSteps...)
}

// NewOneOfStep creates a step in the execution plan that requires at least one
// of the provided sub-steps to be successfully executed for the step to be
// considered successful.
func NewOneOfStep(subSteps ...ExecutionStep) ExecutionStep {
	return NewGroupStep(true, subSteps...)
}

// NewGroupStep creates a step in the execution plan that groups the provided
// sub-steps together, with the specified execution semantic (oneOf or allOf).
func NewGroupStep(oneOf bool, subSteps ...ExecutionStep) ExecutionStep {
	return ExecutionStep{oneOf: oneOf, steps: subSteps}
}

// SetExecutionFlags sets the execution flags for the given step, which can
// modify the interpretation of result of the step during execution. For
// example, the flags can specify whether failed or invalid transactions should
// be tolerated without causing the entire step to be considered failed.
func SetExecutionFlags(step ExecutionStep, flags ExecutionFlags) ExecutionStep {
	step.flags = flags
	return step
}

// ExecutionStep is a single nodes in the hierarchy of processing steps defining an
// execution plan. Each ExecutionStep is either a reference to a transaction to be
// executed or a group of sub-steps. Groups may be marked as "oneOf", meaning
// that only one of the sub-steps needs to be successfully executed for the
// group to be considered successful. Furthermore, for each ExecutionStep, execution
// flags can be defined to specify the behavior of the execution (e.g. whether
// failed or invalid transactions should be tolerated).
type ExecutionStep struct {
	// -- common for individual transactions and groups --
	flags ExecutionFlags

	// -- single transaction fields --
	txRef *TxReference // < if nil, it is a group

	// -- group fields --
	oneOf bool
	steps []ExecutionStep
}

func (s *ExecutionStep) encode(writer io.Writer) error {
	encoding := s.toEncodingV1()
	return rlp.Encode(writer, encoding)
}

func (s *ExecutionStep) decode(reader io.Reader) error {
	var encoding stepEncodingV1
	if err := rlp.Decode(reader, &encoding); err != nil {
		return err
	}
	s.fromEncodingV1(encoding)
	return nil
}

func (s *ExecutionStep) toEncodingV1() stepEncodingV1 {
	encoding := stepEncodingV1{
		Flags: s.flags,
		TxRef: s.txRef,
		OneOf: s.oneOf,
	}
	if s.txRef == nil {
		encoding.Steps = make([]stepEncodingV1, len(s.steps))
		for i, subStep := range s.steps {
			encoding.Steps[i] = subStep.toEncodingV1()
		}
	}
	return encoding
}

func (s *ExecutionStep) fromEncodingV1(encoding stepEncodingV1) {
	s.flags = encoding.Flags
	s.txRef = encoding.TxRef
	s.oneOf = encoding.OneOf
	if encoding.TxRef == nil && len(encoding.Steps) > 0 {
		s.steps = make([]ExecutionStep, len(encoding.Steps))
		for i, subEncoding := range encoding.Steps {
			s.steps[i].fromEncodingV1(subEncoding)
		}
	}
}

// stepEncodingV1 is the RLP encoding structure for a step.
type stepEncodingV1 struct {
	Flags ExecutionFlags
	TxRef *TxReference `rlp:"nil"`
	OneOf bool
	Steps []stepEncodingV1
}

// -- debug and testing utilities --

// String provides a human-readable representation of the step, which can be
// useful for debugging or creating readable unit tests. It assigns a unique
// letter (A, B, C, etc.) to each referenced transaction.
func (s *ExecutionStep) String() string {
	txs := s.GetTransactionReferencesInExecutionOrder()
	references := make(map[TxReference]string)
	for _, tx := range txs {
		if _, found := references[tx]; found {
			continue
		}
		references[tx] = string([]byte{byte('A' + len(references))})
	}
	var out strings.Builder
	s.print(references, &out)
	return out.String()
}

func (s *ExecutionStep) GetTransactionReferencesInExecutionOrder() []TxReference {
	var refs []TxReference
	s.collectReferencedTransactions(&refs)
	return refs
}

func (s *ExecutionStep) collectReferencedTransactions(refs *[]TxReference) {
	if s.txRef != nil {
		*refs = append(*refs, *s.txRef)
	} else {
		for _, subStep := range s.steps {
			subStep.collectReferencedTransactions(refs)
		}
	}
}

func (s *ExecutionStep) print(
	references map[TxReference]string,
	out *strings.Builder,
) {
	if s.flags != EF_Default {
		out.WriteString("Step[")
		out.WriteString(s.flags.String())
		out.WriteString("](")
	}
	if s.txRef != nil {
		out.WriteString(references[*s.txRef])
	} else {
		if s.oneOf {
			out.WriteString("OneOf(")
		} else {
			out.WriteString("AllOf(")
		}
		for i, subStep := range s.steps {
			if i > 0 {
				out.WriteString(",")
			}
			subStep.print(references, out)
		}
		out.WriteString(")")
	}
	if s.flags != EF_Default {
		out.WriteString(")")
	}
}
