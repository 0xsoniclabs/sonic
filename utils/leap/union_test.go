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

package leap

import (
	"cmp"
	"container/heap"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnion_Next_DefaultHasNoNext(t *testing.T) {
	unionIter := Union[int]()
	require.False(t, unionIter.Next())

	unionIter = &unionIterator[int]{}
	require.False(t, unionIter.Next())
}

func TestUnion_Cur_DefaultReturnsZero(t *testing.T) {
	unionIter := Union[int]()
	require.Zero(t, unionIter.Cur())

	unionIter = &unionIterator[int]{}
	require.Zero(t, unionIter.Cur())
}

func TestUnion_ComputesUnionOfInputLists(t *testing.T) {
	tests := map[string]struct {
		inputs   [][]int
		expected []int
	}{
		"no lists": {
			inputs:   nil,
			expected: nil,
		},
		"one empty list": {
			inputs:   [][]int{{}},
			expected: nil,
		},
		"multiple empty lists": {
			inputs:   [][]int{{}, {}, {}},
			expected: nil,
		},
		"one list with values": {
			inputs:   [][]int{{1, 2, 3}},
			expected: []int{1, 2, 3},
		},
		"two lists with overlapping values": {
			inputs: [][]int{
				{1, 2, 3},
				{2, 3, 4},
			},
			expected: []int{1, 2, 2, 3, 3, 4},
		},
		"three lists with overlapping values": {
			inputs: [][]int{
				{1, 2, 3, 4},
				{2, 3, 4, 5},
				{0, 2, 4, 6},
			},
			expected: []int{0, 1, 2, 2, 2, 3, 3, 4, 4, 4, 5, 6},
		},
		"lists with no overlapping values": {
			inputs: [][]int{
				{1, 3, 5},
				{2, 4, 6},
				{0, 7, 8},
			},
			expected: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var iterators []Iterator[int]
			for _, input := range tc.inputs {
				iterators = append(iterators, newIter(input...))
			}
			result := slices.Collect(All(Union(iterators...)))
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnion_Seek_SkipsElements(t *testing.T) {
	tests := map[string]struct {
		inputs   [][]int
		seek     int
		expected []int
	}{
		"seek past all elements": {
			inputs:   [][]int{{1, 2, 3}, {4, 5, 6}},
			seek:     10,
			expected: nil,
		},
		"seek to middle element": {
			inputs:   [][]int{{1, 2, 3}, {4, 5, 6}},
			seek:     4,
			expected: []int{4, 5, 6},
		},
		"seek to first element": {
			inputs:   [][]int{{1, 2, 3}, {4, 5, 6}},
			seek:     1,
			expected: []int{1, 2, 3, 4, 5, 6},
		},
		"seek to non-existent element": {
			inputs:   [][]int{{1, 3, 5}, {7, 9, 11}},
			seek:     6,
			expected: []int{7, 9, 11},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var iterators []Iterator[int]
			for _, input := range tc.inputs {
				iterators = append(iterators, newIter(input...))
			}
			unionIter := Union(iterators...)

			var result []int
			if unionIter.Seek(tc.seek) {
				result = append(result, unionIter.Cur())
				for unionIter.Next() {
					result = append(result, unionIter.Cur())
				}
			}
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIteratorHeap_CanBeUsedAsAHeap(t *testing.T) {
	require := require.New(t)

	iterA := newIter(1, 3)
	iterB := newIter(2, 4)

	iterA.Next() // Cur() == 1
	iterB.Next() // Cur() == 2

	h := iteratorHeap[int]{
		less: cmp.Less[int],
	}

	// Pushing iterA places it at the top of the heap.
	heap.Push(&h, iterA)
	require.Equal(1, h.iters[0].Cur())

	// Pushing iterB keeps iterA at the top of the heap.
	heap.Push(&h, iterB)
	require.Equal(1, h.iters[0].Cur())

	// Advancing iterA moves iterB to the top of the heap.
	h.iters[0].Next()
	heap.Fix(&h, 0)
	require.Equal(2, h.iters[0].Cur())

	// Popping removes iterB, leaving iterA at the top of the heap.
	heap.Pop(&h)
	require.Equal(3, h.iters[0].Cur())

}
