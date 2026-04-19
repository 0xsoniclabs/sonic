package checked

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrOverflow_ProvidesExpectedErrorMessage(t *testing.T) {
	require.EqualError(t, ErrOverflow, "arithmetic overflow")
}

func TestUint64_CreatesValueWithGivenInput(t *testing.T) {
	tests := []uint64{
		0, 1, 15, 255, 74645221587, math.MaxUint64 - 1, math.MaxUint64,
	}

	for _, test := range tests {
		wrapped := Uint64(test)
		unwrapped, err := wrapped.Unwrap()
		require.NoError(t, err)
		require.False(t, wrapped.IsOverflown())
		require.Equal(t, test, unwrapped)
	}
}

func TestOverflow_CreatesOverflowValue(t *testing.T) {
	wrapped := Overflow()
	require.True(t, wrapped.IsOverflown())
	_, err := wrapped.Unwrap()
	require.ErrorIs(t, err, ErrOverflow)
}

func TestUnwrap_NonOverflowingValue_ReturnsValue(t *testing.T) {
	tests := []uint64{
		0, 1, 15, 255, 74645221587, math.MaxUint64 - 1, math.MaxUint64,
	}

	for _, test := range tests {
		wrapped := checkedUint64{value: test}
		unwrapped, err := wrapped.Unwrap()
		require.NoError(t, err)
		require.Equal(t, test, unwrapped)
	}
}

func TestUnwrap_OverflowingValue_ReturnsError(t *testing.T) {
	wrapped := checkedUint64{overflow: true}
	_, err := wrapped.Unwrap()
	require.ErrorIs(t, err, ErrOverflow)
}

func TestAdd_AddsNumbersWhenNotOverflowing(t *testing.T) {
	values := []uint64{
		0, 1, 15, 97, math.MaxUint64/2 - 1, math.MaxUint64 / 2,
	}
	for _, a := range values {
		for _, b := range values {
			result, err := Add(a, b).Unwrap()
			require.NoError(t, err)
			require.Equal(t, a+b, result)
		}
	}
}

func TestAdd_ReturnsOverflowWhenResultOverflows(t *testing.T) {
	tests := []struct {
		a, b uint64
	}{
		{math.MaxUint64, 1},
		{math.MaxUint64 - 1, 2},
		{math.MaxUint64 - 5, 10},
		{math.MaxUint64 / 2, math.MaxUint64 - math.MaxUint64/2 + 1},
	}

	for _, test := range tests {
		result := Add(test.a, test.b)
		require.True(t, result.IsOverflown())
	}
}

func TestAdd_AddingAnOverflowingValueReturnsOverflow(t *testing.T) {
	require := require.New(t)
	values := []uint64{
		0, 1, 15, 97, math.MaxUint64 - 1, math.MaxUint64,
	}

	require.Equal(Overflow(), Add(Overflow(), Overflow()))
	for _, a := range values {
		require.Equal(Overflow(), Add(a, Overflow()))
		require.Equal(Overflow(), Add(Overflow(), a))
	}
}

func TestMul_MultipliesNumbersWhenNotOverflowing(t *testing.T) {
	values := []uint64{
		0, 1, 15, 97, math.MaxUint32 - 1, math.MaxUint32,
	}
	for _, a := range values {
		for _, b := range values {
			result, err := Mul(a, b).Unwrap()
			require.NoError(t, err, "a=%d, b=%d", a, b)
			require.Equal(t, a*b, result)
		}
	}
}

func TestMul_ReturnsOverflowWhenResultOverflows(t *testing.T) {
	tests := []struct {
		a, b uint64
	}{
		{math.MaxUint64, 2},
		{math.MaxUint64 - 1, 3},
		{math.MaxUint64 / 2, 3},
		{math.MaxUint32 + 1, math.MaxUint32 + 1},
	}

	for _, test := range tests {
		result := Mul(test.a, test.b)
		require.True(t, result.IsOverflown())
	}
}

func TestMul_MultiplyingWithAnOverflowingValueReturnsOverflow(t *testing.T) {
	require := require.New(t)
	values := []uint64{
		0, 1, 15, 97, math.MaxUint64 - 1, math.MaxUint64,
	}

	require.Equal(Overflow(), Mul(Overflow(), Overflow()))
	for _, a := range values {
		require.Equal(Overflow(), Mul(a, Overflow()))
		require.Equal(Overflow(), Mul(Overflow(), a))
	}
}

func Test_from_ConvertsUint64ToCheckedUint64(t *testing.T) {
	tests := []uint64{
		0, 1, 15, 255, 74645221587, math.MaxUint64 - 1, math.MaxUint64,
	}

	for _, test := range tests {
		result := from(test)
		require.False(t, result.IsOverflown())
		require.Equal(t, test, result.value)
	}
}

func Test_from_ConvertsOtherUnsignedIntsToCheckedUint64(t *testing.T) {
	require := require.New(t)
	require.Equal(Uint64(5), from(uint8(5)))
	require.Equal(Uint64(17), from(uint16(17)))
	require.Equal(Uint64(742), from(uint32(742)))

	require.Equal(Uint64(math.MaxUint), from(uint(math.MaxUint)))
	require.Equal(Uint64(math.MaxUint8), from(uint8(math.MaxUint8)))
	require.Equal(Uint64(math.MaxUint16), from(uint16(math.MaxUint16)))
	require.Equal(Uint64(math.MaxUint32), from(uint32(math.MaxUint32)))
	require.Equal(Uint64(math.MaxUint64), from(uint64(math.MaxUint64)))
}

func Test_from_ReturnsCheckedUint64AsIs(t *testing.T) {
	tests := []checkedUint64{
		Uint64(0),
		Uint64(1),
		Uint64(15),
		Overflow(),
	}

	for _, test := range tests {
		result := from(test)
		require.Equal(t, test, result)
	}
}
