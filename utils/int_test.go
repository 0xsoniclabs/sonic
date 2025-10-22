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
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestBigIntToU256_Clamps_Inputs(t *testing.T) {
	tests := map[string]struct {
		input    *big.Int
		expected *uint256.Int
	}{
		"nil input": {
			input:    nil,
			expected: nil,
		},
		"negative input": {
			input:    big.NewInt(-1),
			expected: uint256.NewInt(0),
		},
		"zero input": {
			input:    big.NewInt(0),
			expected: uint256.NewInt(0),
		},
		"small positive input": {
			input:    big.NewInt(42),
			expected: uint256.NewInt(42),
		},
		"max uint256 input": {
			input: func() *big.Int {
				b := new(big.Int).SetUint64(1)
				b.Lsh(b, 256)
				b.Sub(b, big.NewInt(1))
				return b
			}(),
			expected: func() *uint256.Int {
				u := new(uint256.Int)
				u.SetAllOne()
				return u
			}(),
		},
		"overflowing input": {
			input: func() *big.Int {
				b := new(big.Int).SetUint64(1)
				b.Lsh(b, 256)
				return b
			}(),
			expected: func() *uint256.Int {
				u := new(uint256.Int)
				u.SetAllOne()
				return u
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := BigIntToUint256Clamped(test.input)
			require.Equal(t, test.expected, got, "BigIntToUint256Clamped(%v) = %v; want %v", test.input, got, test.expected)
		})
	}
}
