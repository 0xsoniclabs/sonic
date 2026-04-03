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
