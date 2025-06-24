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

package cert

import (
	"encoding/json"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestBitSet_Default_IsEmpty(t *testing.T) {
	var b BitSet[uint8]
	require.Empty(t, b.Entries())
}

func TestBitSet_Add_AddsElementToSet(t *testing.T) {
	var b BitSet[uint8]
	require.Equal(t, []uint8{}, b.Entries())
	b.Add(1)
	require.Equal(t, []uint8{1}, b.Entries())
	b.Add(123)
	require.Equal(t, []uint8{1, 123}, b.Entries())
}

func TestBitSet_Add_AddingPresentElementsHasNoEffect(t *testing.T) {
	var b BitSet[uint8]
	b.Add(12)
	require.Equal(t, []uint8{12}, b.Entries())
	b.Add(12)
	require.Equal(t, []uint8{12}, b.Entries())
}

func TestBitSet_Contains_IdentifiesPresentElements(t *testing.T) {
	var b BitSet[uint8]
	require.False(t, b.Contains(10))
	require.False(t, b.Contains(12))
	require.False(t, b.Contains(14))
	b.Add(12)
	require.False(t, b.Contains(10))
	require.True(t, b.Contains(12))
	require.False(t, b.Contains(14))
}

func TestBitSet_ContainsAndEntriesAreConsistent(t *testing.T) {
	const N = 100
	var b BitSet[uint8]
	ref := map[uint8]struct{}{}
	for range N {
		x := uint8(rand.IntN(N))
		b.Add(x)
		ref[x] = struct{}{}

		for i := uint8(0); i < N; i++ {
			_, found := ref[i]
			require.Equal(t, found, b.Contains(i))
		}

		require.ElementsMatch(t, maps.Keys(ref), b.Entries())
	}
}

func TestBitSet_String_PrintsListOfEntries(t *testing.T) {
	var b BitSet[uint8]
	require.Equal(t, "{}", b.String())
	b.Add(1)
	require.Equal(t, "{1}", b.String())
	b.Add(12)
	require.Equal(t, "{1, 12}", b.String())
	b.Add(123)
	require.Equal(t, "{1, 12, 123}", b.String())
}

func TestBitSet_MarshalJSON(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	b.Add(1)
	b.Add(12)
	b.Add(123)
	data, err := json.Marshal(b)
	require.NoError(err)
	require.Equal(`"0x02100000000000000000000000000008"`, string(data))
}

func TestBitSet_UnmarshalJSON(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	err := json.Unmarshal([]byte(`"0x02100000000000000000000000000008"`), &b)
	require.NoError(err)
	require.Equal([]uint8{1, 12, 123}, b.Entries())
}

func TestBitSet_InvalidUnmarshalJSON(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	err := json.Unmarshal([]byte(`"g"`), &b)
	require.Error(err)
}

func TestBitSet_EmptySet_CanBeMarshaledAndUnmarshaled(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	data, err := json.Marshal(b)
	require.NoError(err)

	var b2 BitSet[uint8]
	err = json.Unmarshal(data, &b2)
	require.NoError(err)
	require.Equal([]uint8{}, b2.Entries())
}
