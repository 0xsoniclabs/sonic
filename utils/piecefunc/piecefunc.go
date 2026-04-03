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

package piecefunc

import "math"

const (
	// DecimalUnit is used to define ratios with integers, it's 1.0
	DecimalUnit = 1e6
	maxVal      = math.MaxUint64/uint64(DecimalUnit) - 1
)

// Dot is a pair of numbers
type Dot struct {
	X uint64
	Y uint64
}

type Func struct {
	dots []Dot
}

func NewFunc(dots []Dot) func(x uint64) uint64 {
	if len(dots) < 2 {
		panic("too few dots")
	}

	var prevX uint64
	for i, dot := range dots {
		if i >= 1 && dot.X <= prevX {
			panic("non monotonic X")
		}
		if dot.Y > maxVal {
			panic("too large Y")
		}
		if dot.X > maxVal {
			panic("too large X")
		}
		prevX = dot.X
	}

	return Func{
		dots: dots,
	}.Get
}

// Mul is multiplication of ratios with integer numbers
func Mul(a, b uint64) uint64 {
	return a * b / DecimalUnit
}

// Div is division of ratios with integer numbers
func Div(a, b uint64) uint64 {
	return a * DecimalUnit / b
}

// Get calculates f(x), where f is a piecewise linear function defined by the pieces
func (f Func) Get(x uint64) uint64 {
	if x < f.dots[0].X {
		return f.dots[0].Y
	}
	if x > f.dots[len(f.dots)-1].X {
		return f.dots[len(f.dots)-1].Y
	}
	// find a piece
	p0 := len(f.dots) - 2
	for i, piece := range f.dots {
		if i >= 1 && i < len(f.dots)-1 && piece.X > x {
			p0 = i - 1
			break
		}
	}
	// linearly interpolate
	p1 := p0 + 1

	x0, x1 := f.dots[p0].X, f.dots[p1].X
	y0, y1 := f.dots[p0].Y, f.dots[p1].Y

	ratio := Div(x-x0, x1-x0)

	return Mul(y0, DecimalUnit-ratio) + Mul(y1, ratio)
}
