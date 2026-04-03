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

package epochcheck

import (
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

var (
	// ErrNotRelevant indicates the event's epoch isn't equal to current epoch.
	ErrNotRelevant = errors.New("event is too old or too new")
	// ErrAuth indicates that event's creator isn't authorized to create events in current epoch.
	ErrAuth = errors.New("event creator isn't a validator")
)

// Reader returns currents epoch and its validators group.
type Reader interface {
	GetEpochValidators() (*consensus.Validators, consensus.Epoch)
}

// Checker which require only current epoch info
type Checker struct {
	reader Reader
}

func New(reader Reader) *Checker {
	return &Checker{
		reader: reader,
	}
}

// Validate event
func (v *Checker) Validate(e consensus.Event) error {
	// check epoch first, because validators group is returned only for the current epoch
	validators, epoch := v.reader.GetEpochValidators()
	if e.Epoch() != epoch {
		return ErrNotRelevant
	}
	if !validators.Exists(e.Creator()) {
		return ErrAuth
	}
	return nil
}
