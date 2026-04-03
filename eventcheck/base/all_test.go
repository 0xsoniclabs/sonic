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

package eventcheck

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/sonic/eventcheck/base/basiccheck"
	"github.com/0xsoniclabs/sonic/eventcheck/base/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/base/parentscheck"
)

type testReader struct{}

func (tr *testReader) GetEpochValidators() (*consensus.Validators, consensus.Epoch) {
	vb := consensus.NewValidatorsBuilder()
	vb.Set(1, 1)
	return vb.Build(), 1
}

func TestBasicEventValidation(t *testing.T) {
	var tests = []struct {
		e       consensus.Event
		wantErr error
	}{
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), nil},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(0)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrNotInited},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrNoParents},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(math.MaxInt32 - 1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrHugeValue},
	}

	for _, tt := range tests {
		basicCheck := basiccheck.New()
		assert.Equal(t, tt.wantErr, basicCheck.Validate(tt.e))
	}
}

func TestEpochEventValidation(t *testing.T) {
	var tests = []struct {
		e       consensus.Event
		wantErr error
	}{
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetEpoch(1)
			e.SetCreator(1)
			return e
		}(), nil},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetEpoch(2)
			e.SetCreator(1)
			return e
		}(), epochcheck.ErrNotRelevant},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetEpoch(1)
			e.SetCreator(2)
			return e
		}(), epochcheck.ErrAuth},
	}

	for _, tt := range tests {
		tr := new(testReader)
		epochCheck := epochcheck.New(tr)
		assert.Equal(t, tt.wantErr, epochCheck.Validate(tt.e))
	}
}

func TestParentsEventValidation(t *testing.T) {
	var tests = []struct {
		e         consensus.Event
		pe        consensus.Events
		wantErr   error
		wantPanic bool
	}{
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			selfParent := &consensustest.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{1})
			e.SetParents(consensus.EventHashes{selfParent.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				e.SetCreator(1)
				e.SetID([24]byte{1})
				return consensus.Events{e}
			}(),
			nil, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			selfParent := &consensustest.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{2})
			e.SetParents(consensus.EventHashes{selfParent.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				e.SetCreator(1)
				e.SetID([24]byte{1})
				return consensus.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return consensus.Events{e}
			}(),
			parentscheck.ErrWrongLamport, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(2)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return consensus.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			selfParent := &consensustest.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{1})
			e.SetParents(consensus.EventHashes{selfParent.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(2)
				e.SetLamport(1)
				e.SetID([24]byte{1})
				return consensus.Events{e}
			}(),
			parentscheck.ErrWrongSeq, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			return e
		}(),
			nil,
			parentscheck.ErrWrongSeq, false},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			nil,
			nil, true},
	}

	for _, tt := range tests {
		parentsCheck := parentscheck.New()
		if tt.wantPanic {
			assert.Panics(t, func() {
				err := parentsCheck.Validate(tt.e, tt.pe)
				if err != nil {
					return
				}
			})
		} else {
			assert.Equal(t, tt.wantErr, parentsCheck.Validate(tt.e, tt.pe))
		}
	}
}

func TestAllEventValidation(t *testing.T) {
	var tests = []struct {
		e       consensus.Event
		pe      consensus.Events
		wantErr error
	}{
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			nil,
			basiccheck.ErrNotInited},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(),
			nil,
			epochcheck.ErrAuth},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return consensus.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent},
		{func() consensus.Event {
			e := &consensustest.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(2)
			e.SetCreator(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			e.SetParents(consensus.EventHashes{e.ID()})
			return e
		}(),
			func() consensus.Events {
				e := &consensustest.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return consensus.Events{e}
			}(),
			nil},
	}

	tr := new(testReader)

	checkers := Checkers{
		Basiccheck:   basiccheck.New(),
		Epochcheck:   epochcheck.New(tr),
		Parentscheck: parentscheck.New(),
	}

	for _, tt := range tests {
		assert.Equal(t, tt.wantErr, checkers.Validate(tt.e, tt.pe))
	}
}
