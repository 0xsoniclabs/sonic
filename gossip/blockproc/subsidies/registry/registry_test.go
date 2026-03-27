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

package registry

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetCode_CodeIsNotEmpty(t *testing.T) {
	require.NotEmpty(t, GetCode())
}

func TestGetCode_ReturnsCopy(t *testing.T) {
	code1 := GetCode()
	code2 := GetCode()
	code1[0] = 0xff
	require.NotEqual(t, code1[0], code2[0], "GetCode should return a copy")
}

func TestGetAddress_ReturnsNonZero(t *testing.T) {
	addr := GetAddress()
	require.NotEqual(t, (common.Address{}), addr)
}

func TestGetAddress_MatchesExpected(t *testing.T) {
	expected := common.HexToAddress("0x7d0E23398b6CA0eC7Cdb5b5Aad7F1b11215012d2")
	require.Equal(t, expected, GetAddress())
}

func TestFunctionSelectors_AreNonZero(t *testing.T) {
	require.NotZero(t, GetGasConfigFunctionSelector)
	require.NotZero(t, ChooseFundFunctionSelector)
	require.NotZero(t, DeductFeesFunctionSelector)
}

func TestFunctionSelectors_AreDistinct(t *testing.T) {
	selectors := []uint32{
		GetGasConfigFunctionSelector,
		ChooseFundFunctionSelector,
		DeductFeesFunctionSelector,
	}
	seen := make(map[uint32]bool)
	for _, s := range selectors {
		require.False(t, seen[s], "duplicate function selector: %x", s)
		seen[s] = true
	}
}
