package leap

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefetch_ListsAllElements(t *testing.T) {
	values := []int{1, 2, 3, 4, 5}
	iter := Prefetch(newIter(values...))
	defer iter.Release()
	require.Equal(t, values, slices.Collect(All(iter)))
}

func TestPrefetch_Seek(t *testing.T) {
	require := require.New(t)

	values := []int{1, 2, 3, 4, 5}
	iter := Prefetch(newIter(values...))

	require.True(iter.Next())
	require.Equal(1, iter.Cur())

	require.True(iter.Seek(3))
	require.Equal(3, iter.Cur())

	require.False(iter.Seek(6))
	iter.Release()
}
