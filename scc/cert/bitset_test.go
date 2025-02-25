package cert

import (
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
	data, err := b.MarshalJSON()
	require.NoError(err)
	require.Equal(`"0x02100000000000000000000000000008"`, string(data))
}

func TestBitSet_UnmarshalJSON(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	err := b.UnmarshalJSON([]byte(`"0x02100000000000000000000000000008"`))
	require.NoError(err)
	require.Equal([]uint8{1, 12, 123}, b.Entries())
}

func TestBitSet_InvalidUnmarshalJSON(t *testing.T) {
	require := require.New(t)

	var b BitSet[uint8]
	err := b.UnmarshalJSON([]byte(`"g"`))
	require.Error(err)
}
