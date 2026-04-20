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

package checked

import (
	"math"

	"github.com/0xsoniclabs/carmen/go/common"
)

// ErrOverflow is returned when unwrapping a U64 that has overflowed.
const ErrOverflow = common.ConstError("arithmetic overflow")

type U64 struct {
	value    uint64
	overflow bool
}

// Uint64 creates a new checked U64 with the given value.
func Uint64(value uint64) U64 {
	return U64{value: value}
}

// Overflow creates a new checked U64 that has overflowed.
func Overflow() U64 {
	return U64{overflow: true}
}

// IsOverflown returns true if the checked U64 has overflowed, and false otherwise.
func (s U64) IsOverflown() bool {
	return s.overflow
}

// Unwrap returns the value of the checked U64 if it has not overflowed, and
// an [ErrOverflow] otherwise.
func (s U64) Unwrap() (uint64, error) {
	if s.overflow {
		return 0, ErrOverflow
	}
	return s.value, nil
}

// Add returns the sum of a and b as a checked U64 that can be used for
// further arithmetic operations. If the sum overflows, the returned checked
// U64 will be in an overflowed state. If any of the inputs is in an
// overflowed state, the result will also be in an overflowed state.
func Add[A, B unsignedInt](a A, b B) U64 {
	x := from(a)
	y := from(b)
	if x.overflow || y.overflow || x.value > math.MaxUint64-y.value {
		return Overflow()
	}
	return Uint64(x.value + y.value)
}

// Mul returns the product of a and b as a checked U64 that can be used for
// further arithmetic operations. If the product overflows, the returned checked
// U64 will be in an overflowed state. If any of the inputs is in an
// overflowed state, the result will also be in an overflowed state.
func Mul[A, B unsignedInt](a A, b B) U64 {
	x := from(a)
	y := from(b)
	if x.overflow || y.overflow {
		return Overflow()
	}
	if y.value != 0 && x.value > math.MaxUint64/y.value {
		return Overflow()
	}
	return Uint64(x.value * y.value)
}

type unsignedInt interface {
	uint8 | uint16 | uint32 | uint64 | uint | U64
}

// from is an internal convenience utility that converts arithmetic values or
// checked U64 instances into U64 values.
func from[T unsignedInt](a T) U64 {
	var res U64
	switch x := any(a).(type) {
	case uint:
		res = Uint64(uint64(x))
	case uint8:
		res = Uint64(uint64(x))
	case uint16:
		res = Uint64(uint64(x))
	case uint32:
		res = Uint64(uint64(x))
	case uint64:
		res = Uint64(x)
	case U64:
		res = x
	}
	return res
}
