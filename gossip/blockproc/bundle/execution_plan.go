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
	"bytes"
	"errors"
	"fmt"
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
	Root   ExecutionStep
	Range  BlockRange
	Period TimePeriod
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
	// version byte for future compatibility
	n, err := writer.Write([]byte{0x01})
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("failed to write version byte")
	}

	return errors.Join(
		p.Range.encode(writer),
		p.Period.encode(writer),
		p.Root.encode(writer),
	)
}

func (p *ExecutionPlan) decode(reader io.Reader) error {
	// version byte for future compatibility
	var version [1]byte
	if _, err := io.ReadFull(reader, version[:]); err != nil {
		return err
	}
	if version[0] != 0x01 {
		return fmt.Errorf("unsupported execution plan version: %d", version[0])
	}

	return errors.Join(
		p.Range.decode(reader),
		p.Period.decode(reader),
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

// ExecutionStep is a node in the hierarchy of an execution plan describing a
// processing step. It can either be a single transaction to be executed or a
// group of nested execution steps.
type ExecutionStep struct {
	single *single // < -- mutually exclusive with group
	group  *group
}

// single is the structure representing a single transaction execution step,
// containing a reference to the transaction and any execution flags that modify
// the interpretation of the step's result during execution.
type single struct {
	txRef TxReference
	flags ExecutionFlags
}

// group is the structure representing a group of execution steps, which can be
// executed with different semantics (e.g. oneOf or allOf) and can also have
// a flag whether a failure of the group should be tolerated.
type group struct {
	oneOf          bool
	tolerateFailed bool
	steps          []ExecutionStep
}

// NewTxStep creates a step in the execution plan processing a single
// transaction identified by the given TxReference.
func NewTxStep(txRef TxReference) ExecutionStep {
	return ExecutionStep{single: &single{txRef: txRef}}
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
	return ExecutionStep{group: &group{oneOf: oneOf, steps: subSteps}}
}

// WithFlags produces a modified version of this step with the given flags set,
// which can  modify the interpretation of result of the step during execution.
// For example, the flags can specify whether failed or invalid transactions
// should be tolerated without causing the entire step to be considered failed.
func (s ExecutionStep) WithFlags(flags ExecutionFlags) ExecutionStep {
	res := s
	if res.single != nil {
		copy := *res.single
		res.single = &copy
		res.single.flags = flags
	} else if res.group != nil {
		if flags.TolerateInvalid() {
			panic("TolerateInvalid flag is not supported for groups")
		}
		copy := *res.group
		res.group = &copy
		res.group.tolerateFailed = flags&EF_TolerateFailed != 0
	}
	return res
}

// Flags returns the execution flags for this step. For single transaction
// steps it returns the step's own flags. For group steps it recurses into the
// first child step, which enables callers to retrieve the common flags for a
// flat plan where all transaction steps carry the same flags.
func (s *ExecutionStep) Flags() ExecutionFlags {
	if s.single != nil {
		return s.single.flags
	}
	if s.group != nil && len(s.group.steps) > 0 {
		return s.group.steps[0].Flags()
	}
	return EF_Default
}

// valid returns true if the step is valid, meaning that it is either a single
// step or a group step, but not both or neither. This is a basic validation to
// ensure the integrity of the execution plan structure.
func (s *ExecutionStep) valid() bool {
	if s.single != nil && s.group != nil {
		return false
	}
	if s.single == nil && s.group == nil {
		return false
	}
	return true
}

func (s *ExecutionStep) encode(writer io.Writer) error {
	if !s.valid() {
		return fmt.Errorf("can not encode invalid execution step")
	}
	encoding := s.toEncodingV1()
	return rlp.Encode(writer, encoding)
}

func (s *ExecutionStep) decode(reader io.Reader) error {
	// The encoded step is read as raw RLP first, so that its nesting depth can be
	// bounded before it is decoded into the recursive stepEncodingV1 structure.
	// The plan's nesting depth is limited to MaxGroupNestingDepth, but that limit
	// is only enforced by validateStep, after decoding. Since both rlp.Decode and
	// fromEncodingV1 recurse once per nesting level, a maliciously deep plan could
	// exhaust the goroutine stack during decoding, before validation ever runs
	// (a single deeply nested envelope decoded by every node would crash the
	// network). checkStepEncodingDepth guards against this by rejecting overly
	// deep encodings up front, using a walker whose own recursion is bounded by
	// the same limit.
	raw, err := rlp.NewStream(reader, 0).Raw()
	if err != nil {
		return err
	}
	if err := checkStepEncodingDepth(raw); err != nil {
		return err
	}

	var encoding stepEncodingV1
	if err := rlp.DecodeBytes(raw, &encoding); err != nil {
		return err
	}
	s.fromEncodingV1(encoding)
	return nil
}

// maxStepEncodingRlpDepth is the maximum RLP nesting depth permitted for an
// encoded execution step. It is a loose, consensus-neutral anti-DoS bound used
// during decoding, distinct from the precise MaxGroupNestingDepth rule enforced
// later by validateStep. A valid step has at most MaxGroupNestingDepth levels of
// nested groups; each group level contributes a small constant number of RLP
// list levels (the step list, its sub-steps list, and a leaf's TxReference
// list). The factor of 4 leaves ample headroom so that no plan satisfying the
// MaxGroupNestingDepth rule is ever rejected here, while still bounding the
// decode recursion far below any level that could exhaust the goroutine stack.
const maxStepEncodingRlpDepth = 4 * (MaxGroupNestingDepth + 1)

// checkStepEncodingDepth verifies that the RLP nesting depth of an encoded
// execution step does not exceed maxStepEncodingRlpDepth. It walks the raw RLP
// structure, descending into every nested list, but stops as soon as the depth
// limit is exceeded. Its own recursion (and therefore stack usage) is thus
// bounded by the limit, making it safe to run on untrusted input. It must be
// called before decoding the raw bytes into the recursive stepEncodingV1
// structure to prevent stack exhaustion from maliciously deep encodings.
func checkStepEncodingDepth(raw []byte) error {
	return checkRlpNestingDepth(rlp.NewStream(bytes.NewReader(raw), 0), 0)
}

func checkRlpNestingDepth(stream *rlp.Stream, depth int) error {
	if depth > maxStepEncodingRlpDepth {
		return fmt.Errorf(
			"encoded execution step exceeds maximum nesting depth of %d",
			maxStepEncodingRlpDepth,
		)
	}
	if _, err := stream.List(); err != nil {
		if err == rlp.ErrExpectedList {
			// A non-list value (string/byte) can not nest; consume and return.
			_, err := stream.Raw()
			return err
		}
		return err
	}
	for stream.MoreDataInList() {
		if err := checkRlpNestingDepth(stream, depth+1); err != nil {
			return err
		}
	}
	return stream.ListEnd()
}

func (s *ExecutionStep) toEncodingV1() stepEncodingV1 {
	encoding := stepEncodingV1{}
	if s.single != nil {
		encoding.Flags = s.single.flags
		encoding.TxRef = &s.single.txRef
	} else if s.group != nil {
		encoding.OneOf = s.group.oneOf
		encoding.TolerateFailed = s.group.tolerateFailed
		encoding.Steps = make([]stepEncodingV1, len(s.group.steps))
		for i, subStep := range s.group.steps {
			encoding.Steps[i] = subStep.toEncodingV1()
		}
	}
	return encoding
}

func (s *ExecutionStep) fromEncodingV1(encoding stepEncodingV1) {
	s.single = nil
	s.group = nil
	if encoding.TxRef != nil {
		s.single = &single{
			txRef: *encoding.TxRef,
			flags: encoding.Flags,
		}
	} else {
		s.group = &group{
			oneOf:          encoding.OneOf,
			tolerateFailed: encoding.TolerateFailed,
		}
		if len(encoding.Steps) > 0 {
			s.group.steps = make([]ExecutionStep, len(encoding.Steps))
			for i, subEncoding := range encoding.Steps {
				s.group.steps[i].fromEncodingV1(subEncoding)
			}
		}
	}
}

// stepEncodingV1 is the RLP encoding structure for a step.
type stepEncodingV1 struct {
	// single step fields
	Flags ExecutionFlags
	TxRef *TxReference `rlp:"nil"`
	// group step fields
	OneOf          bool
	TolerateFailed bool
	Steps          []stepEncodingV1
}

// -- debug and testing utilities --

// String provides a human-readable representation of the step, which can be
// useful for debugging or creating readable unit tests. It assigns a unique
// letter (A, B, C, etc.) to each referenced transaction.
func (s *ExecutionStep) String() string {
	txs := s.GetTransactionReferencesInReferencedOrder()
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

func (s *ExecutionStep) GetTransactionReferencesInReferencedOrder() []TxReference {
	var refs []TxReference
	s.collectReferencedTransactions(&refs)
	return refs
}

func (s *ExecutionStep) collectReferencedTransactions(refs *[]TxReference) {
	if s.single != nil {
		*refs = append(*refs, s.single.txRef)
	}
	if s.group != nil {
		for _, subStep := range s.group.steps {
			subStep.collectReferencedTransactions(refs)
		}
	}
}

func (s *ExecutionStep) print(
	references map[TxReference]string,
	out *strings.Builder,
) {
	if !s.valid() {
		out.WriteString("InvalidStep")
		return
	}
	if s.single != nil {
		if s.single.flags != EF_Default {
			out.WriteString("Step[")
			out.WriteString(s.single.flags.String())
			out.WriteString("](")
		}
		out.WriteString(references[s.single.txRef])
		if s.single.flags != EF_Default {
			out.WriteString(")")
		}
		return
	}

	if s.group.tolerateFailed {
		out.WriteString("TolerateFailed(")
	}
	if s.group.oneOf {
		out.WriteString("OneOf(")
	} else {
		out.WriteString("AllOf(")
	}
	for i, subStep := range s.group.steps {
		if i > 0 {
			out.WriteString(",")
		}
		subStep.print(references, out)
	}
	out.WriteString(")")

	if s.group.tolerateFailed {
		out.WriteString(")")
	}
}
