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

package utils

import (
	"math"
	"math/big"
	"testing"
)

func TestBigMin_WithoutArgumentsReturnsNil(t *testing.T) {
	if BigMin() != nil {
		t.Error("BigMin() did not return nil")
	}
}

func TestBigMin_ReturnsTheMinimum(t *testing.T) {
	tests := [][]int{
		{0},
		{0, 1},
		{1, 0},
		{1, 1},
		{1, 2, 3, 4, 5},
		{5, 4, 3, 2, 1},
	}

	for _, test := range tests {
		min := math.MaxInt
		args := make([]*big.Int, len(test))
		for i, v := range test {
			args[i] = big.NewInt(int64(v))
			if v < min {
				min = v
			}
		}
		got := int(BigMin(args...).Int64())
		if got != min {
			t.Errorf("BigMin(%v) = %d; want %d", test, got, min)
		}
	}
}

func TestBigMin_ReturnsThePointerToTheMinimum(t *testing.T) {
	a := big.NewInt(1)
	b := big.NewInt(2)
	if BigMin(a, b) != a {
		t.Error("BigMin(1, 2) did not return the pointer to the minimum")
	}
	if BigMin(b, a) != a {
		t.Error("BigMin(2, 1) did not return the pointer to the minimum")
	}
}

func TestBigMax_WithoutArgumentsReturnsNil(t *testing.T) {
	if BigMax() != nil {
		t.Error("BigMax() did not return nil")
	}
}

func TestBigMax_ReturnsTheMaximum(t *testing.T) {
	tests := [][]int{
		{0},
		{0, 1},
		{1, 0},
		{1, 1},
		{1, 2, 3, 4, 5},
		{5, 4, 3, 2, 1},
	}

	for _, test := range tests {
		max := -1
		args := make([]*big.Int, len(test))
		for i, v := range test {
			args[i] = big.NewInt(int64(v))
			if v > max {
				max = v
			}
		}
		got := int(BigMax(args...).Int64())
		if got != max {
			t.Errorf("BigMax(%v) = %d; want %d", test, got, max)
		}
	}
}

func TestBigMax_ReturnsThePointerToTheMaximum(t *testing.T) {
	a := big.NewInt(1)
	b := big.NewInt(2)
	if BigMax(a, b) != b {
		t.Error("BigMin(1, 2) did not return the pointer to the minimum")
	}
	if BigMax(b, a) != b {
		t.Error("BigMin(2, 1) did not return the pointer to the minimum")
	}
}
