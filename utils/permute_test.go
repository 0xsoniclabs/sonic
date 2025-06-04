package utils

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermute_EmptyList_ProducesOneResult(t *testing.T) {
	res := slices.Collect(Permute([]int{}))
	require.Equal(t, [][]int{{}}, res)
}

func TestPermute_SingletonList_ProducesOneResult(t *testing.T) {
	res := slices.Collect(Permute([]int{1}))
	require.Equal(t, [][]int{{1}}, res)
}

func TestPermute_ListOfTwoElements_ProducesTwoResults(t *testing.T) {
	res := slices.Collect(Permute([]int{1, 2}))
	require.ElementsMatch(t, [][]int{{1, 2}, {2, 1}}, res)
}

func TestPermute_ListOfThreeElements_ProducesSixResults(t *testing.T) {
	res := slices.Collect(Permute([]int{1, 2, 3}))
	require.ElementsMatch(t, [][]int{
		{1, 2, 3},
		{1, 3, 2},
		{2, 1, 3},
		{2, 3, 1},
		{3, 1, 2},
		{3, 2, 1},
	}, res)
}

func TestPermute_CanBeAborted(t *testing.T) {
	res := [][]int{}
	for cur := range Permute([]int{1, 2, 3}) {
		res = append(res, cur)
		if len(res) >= 3 { // stop after 3 results
			break
		}
	}
	require.Len(t, res, 3)
	require.ElementsMatch(t, [][]int{
		{1, 2, 3},
		{1, 3, 2},
		{2, 1, 3},
	}, res)
}
