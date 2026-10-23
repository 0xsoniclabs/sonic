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

package tests

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ToHexUint64(t *testing.T) {
	tests := []struct {
		name string
		val  uint64
	}{
		{"zero", 0},
		{"normal", 1234},
		{"max", math.MaxUint64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexUint64(tt.val)
			require.Equal(t, tt.val, uint64(*hexVal))
		})
	}
}

func Test_ToHexBigInt(t *testing.T) {
	tests := []struct {
		name string
		val  *big.Int
	}{
		{"zero", big.NewInt(0)},
		{"normal", big.NewInt(1e18)},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexBigInt(tt.val)
			if tt.val == nil {
				require.Nil(t, hexVal)
			} else {
				require.Equal(t, tt.val, (*big.Int)(hexVal))
			}
		})
	}
}
