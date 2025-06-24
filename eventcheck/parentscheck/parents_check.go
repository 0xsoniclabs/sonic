// Copyright 2025 Sonic Operations Ltd
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

	base "github.com/Fantom-foundation/lachesis-base/eventcheck/parentscheck"

	"github.com/0xsoniclabs/sonic/inter"
)

var (
	ErrPastTime = errors.New("event has lower claimed time than self-parent")
)

// Checker which require only parents list + current epoch info
type Checker struct {
	base *base.Checker
}

// New validator which performs checks, which require known the parents
func New() *Checker {
	return &Checker{
		base: &base.Checker{},
	}
}

// Validate event
func (v *Checker) Validate(e inter.EventI, parents inter.EventIs) error {
	if err := v.base.Validate(e, parents.Bases()); err != nil {
		return err
	}

	if e.SelfParent() != nil {
		selfParent := parents[0]
		if !e.IsSelfParent(selfParent.ID()) {
			// sanity check, self-parent is always first, it's how it's stored
			return base.ErrWrongSelfParent
		}
		// selfParent time
		if e.CreationTime() <= selfParent.CreationTime() {
			return ErrPastTime
		}
	}

	return nil
}
