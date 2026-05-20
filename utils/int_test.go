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

package utils

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestBigIntToUint256_ValidConversions(t *testing.T) {
	tests := map[string]struct {
		input    *big.Int
		expected *uint256.Int
	}{
		"nil": {
			input:    nil,
			expected: nil,
		},
		"zero": {
			input:    big.NewInt(0),
			expected: uint256.NewInt(0),
		},
		"one": {
			input:    big.NewInt(1),
			expected: uint256.NewInt(1),
		},
		"small value": {
			input:    big.NewInt(42),
			expected: uint256.NewInt(42),
		},
		"max uint64": {
			input: new(big.Int).SetUint64(^uint64(0)),
			expected: func() *uint256.Int {
				return new(uint256.Int).SetUint64(^uint64(0))
			}(),
		},
		"max uint64 + 1": {
			input: new(big.Int).Add(
				new(big.Int).SetUint64(^uint64(0)),
				big.NewInt(1),
			),
			expected: func() *uint256.Int {
				u := new(uint256.Int).SetUint64(^uint64(0))
				u.AddUint64(u, 1)
				return u
			}(),
		},
		"max uint128": {
			input: func() *big.Int {
				b := new(big.Int).Lsh(big.NewInt(1), 128)
				b.Sub(b, big.NewInt(1))
				return b
			}(),
			expected: func() *uint256.Int {
				u, _ := uint256.FromBig(new(big.Int).Sub(
					new(big.Int).Lsh(big.NewInt(1), 128),
					big.NewInt(1),
				))
				return u
			}(),
		},
		"exactly 32 bytes (max uint256)": {
			input: func() *big.Int {
				b := new(big.Int).Lsh(big.NewInt(1), 256)
				b.Sub(b, big.NewInt(1))
				return b
			}(),
			expected: func() *uint256.Int {
				u := new(uint256.Int)
				u.SetAllOne()
				return u
			}(),
		},
		"1 wei": {
			input:    big.NewInt(1),
			expected: uint256.NewInt(1),
		},
		"1 ether in wei": {
			input: new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			expected: func() *uint256.Int {
				u, _ := uint256.FromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
				return u
			}(),
		},
		"power of 2 boundary (2^255)": {
			input: new(big.Int).Lsh(big.NewInt(1), 255),
			expected: func() *uint256.Int {
				u, _ := uint256.FromBig(new(big.Int).Lsh(big.NewInt(1), 255))
				return u
			}(),
		},
		"2^256 - 2 (one below max)": {
			input: new(big.Int).Sub(
				new(big.Int).Lsh(big.NewInt(1), 256),
				big.NewInt(2),
			),
			expected: func() *uint256.Int {
				u := new(uint256.Int)
				u.SetAllOne()
				one := uint256.NewInt(1)
				u.Sub(u, one)
				return u
			}(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := BigIntToUint256(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)

			// verify round-trip: converting back should yield the original
			roundTrip := Uint256ToBigInt(result)
			if tt.input == nil {
				require.Nil(t, roundTrip, "round-trip failed: input=%v got=%v", tt.input, roundTrip)
				return
			}
			require.Equal(t, 0, tt.input.Cmp(roundTrip),
				"round-trip failed: input=%v got=%v", tt.input, roundTrip)
		})
	}
}

func TestBigIntToUint256_ErrorCases(t *testing.T) {
	tests := map[string]struct {
		input       *big.Int
		expectedErr string
	}{
		"negative one": {
			input:       big.NewInt(-1),
			expectedErr: "negative",
		},
		"large negative": {
			input:       big.NewInt(-1_000_000_000),
			expectedErr: "negative",
		},
		"min int64": {
			input:       new(big.Int).SetInt64(-(1 << 63)),
			expectedErr: "negative",
		},
		"2^256 (one above max)": {
			input:       new(big.Int).Lsh(big.NewInt(1), 256),
			expectedErr: "exceeding 32 bytes",
		},
		"2^257": {
			input:       new(big.Int).Lsh(big.NewInt(1), 257),
			expectedErr: "exceeding 32 bytes",
		},
		"2^512": {
			input:       new(big.Int).Lsh(big.NewInt(1), 512),
			expectedErr: "exceeding 32 bytes",
		},
		"max uint256 + 1": {
			input: new(big.Int).Add(
				new(big.Int).Sub(
					new(big.Int).Lsh(big.NewInt(1), 256),
					big.NewInt(1),
				),
				big.NewInt(1),
			),
			expectedErr: "exceeding 32 bytes",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := BigIntToUint256(tt.input)
			require.Error(t, err)
			require.Nil(t, result)
			require.ErrorContains(t, err, tt.expectedErr)
		})
	}
}

func TestBigIntToUint256Clamped_Clamps_Inputs(t *testing.T) {
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
