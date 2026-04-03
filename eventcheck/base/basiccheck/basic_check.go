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

package basiccheck

import (
	"errors"
	"math"

	"github.com/0xsoniclabs/consensus/consensus"
)

var (
	ErrNoParents     = errors.New("event has no parents")
	ErrNotInited     = errors.New("event field is not initialized")
	ErrHugeValue     = errors.New("too big value")
	ErrDoubleParents = errors.New("event has double parents")
)

type Checker struct{}

// New validator which performs checks which don't require anything except event
func New() *Checker {
	return &Checker{}
}

func (v *Checker) checkLimits(e consensus.Event) error {
	if e.Seq() >= math.MaxInt32-1 || e.Epoch() >= math.MaxInt32-1 || e.Frame() >= math.MaxInt32-1 ||
		e.Lamport() >= math.MaxInt32-1 {
		return ErrHugeValue
	}

	return nil
}

func (v *Checker) checkInited(e consensus.Event) error {
	// it's unsigned, but check for negative in a case if type will change
	if e.Seq() <= 0 || e.Epoch() <= 0 || e.Frame() <= 0 || e.Lamport() <= 0 {
		return ErrNotInited
	}

	if e.Seq() > 1 && len(e.Parents()) == 0 {
		return ErrNoParents
	}

	return nil
}

// Validate event
func (v *Checker) Validate(e consensus.Event) error {
	if err := v.checkLimits(e); err != nil {
		return err
	}
	if err := v.checkInited(e); err != nil {
		return err
	}

	// parents
	if len(e.Parents().Set()) != len(e.Parents()) {
		return ErrDoubleParents
	}

	return nil
}
