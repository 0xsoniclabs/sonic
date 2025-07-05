// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package parentscheck

import (
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

var (
	ErrWrongSeq        = errors.New("event has wrong sequence time")
	ErrWrongLamport    = errors.New("event has wrong Lamport time")
	ErrWrongSelfParent = errors.New("event is missing self-parent")
)

// Checker performs checks, which require the parents list
type Checker struct{}

// New checker which performs checks, which require the parents list
func New() *Checker {
	return &Checker{}
}

// Validate event
func (v *Checker) Validate(e consensus.Event, parents consensus.Events) error {
	if len(e.Parents()) != len(parents) {
		panic("parentscheck: expected event's parents as an argument")
	}

	// double parents are checked by basiccheck

	// lamport
	maxLamport := consensus.Lamport(0)
	for _, p := range parents {
		maxLamport = consensus.MaxLamport(maxLamport, p.Lamport())
	}
	if e.Lamport() != maxLamport+1 {
		return ErrWrongLamport
	}

	// self-parent
	for i, p := range parents {
		if (p.Creator() == e.Creator()) != e.IsSelfParent(e.Parents()[i]) {
			return ErrWrongSelfParent
		}
	}

	// seq
	if (e.Seq() == 1) != (e.SelfParent() == nil) {
		return ErrWrongSeq
	}
	if e.SelfParent() != nil {
		selfParent := parents[0]
		if !e.IsSelfParent(selfParent.ID()) {
			// sanity check, self-parent is always first, it's how it's stored
			return ErrWrongSelfParent
		}
		if e.Seq() != selfParent.Seq()+1 {
			return ErrWrongSeq
		}
	}

	return nil
}
