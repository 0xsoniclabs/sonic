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
