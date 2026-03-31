// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

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
