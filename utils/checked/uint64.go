package checked

import (
	"math"

	"github.com/0xsoniclabs/carmen/go/common"
)

// ErrOverflow is returned when unwrapping a checked Uint64 that has overflowed.
const ErrOverflow = common.ConstError("arithmetic overflow")

type checkedUint64 struct {
	value    uint64
	overflow bool
}

// Uint64 creates a new checked Uint64 with the given value.
func Uint64(value uint64) checkedUint64 {
	return checkedUint64{value: value}
}

// Overflow creates a new checked Uint64 that has overflowed.
func Overflow() checkedUint64 {
	return checkedUint64{overflow: true}
}

// IsOverflown returns true if the checked Uint64 has overflowed, and false otherwise.
func (s checkedUint64) IsOverflown() bool {
	return s.overflow
}

// Unwrap returns the value of the checked Uint64 if it has not overflowed, and
// an [ErrOverflow] otherwise.
func (s checkedUint64) Unwrap() (uint64, error) {
	if s.overflow {
		return 0, ErrOverflow
	}
	return s.value, nil
}

// Add returns the sum of a and b as a checked Uint64 that can be used for
// further arithmetic operations. If the sum overflows, the returned checked
// Uint64 will be in an overflowed state. If any of the inputs is in an
// overflowed state, the result will also be in an overflowed state.
func Add[A, B arg](a A, b B) checkedUint64 {
	x := from(a)
	y := from(b)
	if x.overflow || y.overflow || x.value > math.MaxUint64-y.value {
		return Overflow()
	}
	return Uint64(x.value + y.value)
}

// Mul returns the product of a and b as a checked Uint64 that can be used for
// further arithmetic operations. If the product overflows, the returned checked
// Uint64 will be in an overflowed state. If any of the inputs is in an
// overflowed state, the result will also be in an overflowed state.
func Mul[A, B arg](a A, b B) checkedUint64 {
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

type arg interface {
	uint8 | uint16 | uint32 | uint64 | uint | checkedUint64
}

// from is an internal convenience utility that converts arithmetic values or
// checked uint64 instances into checkedUint64 values.
func from[T arg](a T) checkedUint64 {
	var res checkedUint64
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
	case checkedUint64:
		res = x
	}
	return res
}
