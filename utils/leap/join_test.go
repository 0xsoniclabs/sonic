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
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestJoin_ListOfValues_FindsCommonElementsInOrder(t *testing.T) {
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
		"two lists with common values": {
			inputs: [][]int{
				{1, 2, 3},
				{2, 3, 4},
			},
			expected: []int{2, 3},
		},
		"three lists with common values": {
			inputs: [][]int{
				{1, 2, 3, 4},
				{2, 3, 4, 5},
				{0, 2, 4, 6},
			},
			expected: []int{2, 4},
		},
		"lists with no common values": {
			inputs: [][]int{
				{1, 3, 5},
				{2, 4, 6},
				{0, 7, 8},
			},
			expected: nil,
		},
		"lists with all common values": {
			inputs: [][]int{
				{1, 2},
				{1, 2},
				{1, 2},
			},
			expected: []int{1, 2},
		},
		"example from paper": { // https://arxiv.org/pdf/1210.0481
			inputs: [][]int{
				{0, 1, 3, 4, 5, 6, 7, 8, 9, 11},
				{0, 2, 6, 7, 8, 9},
				{2, 4, 5, 8, 10},
			},
			expected: []int{8},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			var iters []Iterator[int]
			for _, values := range tc.inputs {
				iters = append(iters, newIter(values...))
			}

			result := slices.Collect(Join(iters...))
			require.Equal(tc.expected, result)
		})
	}
}

func TestJoin_ExhaustiveCombinations(t *testing.T) {

	toIter := func(mask uint32) *listIterator {
		var values []int
		for i := range 32 {
			if (mask & (1 << i)) != 0 {
				values = append(values, i)
			}
		}
		return newIter(values...)
	}

	N := uint32(1 << 6)

	// Joining two iterators.
	for a := range N {
		for b := range N {
			res := Join(toIter(a), toIter(b))
			var expected []int
			for i := range 32 {
				if (a&(1<<i)) != 0 && (b&(1<<i)) != 0 {
					expected = append(expected, i)
				}
			}
			result := slices.Collect(res)
			require.Equal(t, expected, result, "masks: %06b %06b", a, b)
		}
	}

	// Joining three iterators.
	for a := range N {
		for b := range N {
			for c := range N {
				res := Join(toIter(a), toIter(b), toIter(c))
				var expected []int
				for i := range 32 {
					if (a&(1<<i)) != 0 && (b&(1<<i)) != 0 && (c&(1<<i)) != 0 {
						expected = append(expected, i)
					}
				}
				result := slices.Collect(res)
				require.Equal(t, expected, result, "masks: %06b %06b %06b", a, b, c)
			}
		}
	}
}

func TestJoin_IterationCanBeAborted(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMockIterator[int](ctrl)

	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Cur().Return(5).AnyTimes(),
	)
	// No more next call after abort.

	for x := range Join(iter) {
		require.Equal(x, 5)
		break
	}
}

// -- Test Utilities --

// listIterator is a simple iterator over a list of integers for testing.
type listIterator struct {
	values []int
}

// newIter creates a new listIterator over the given values.
func newIter(values ...int) *listIterator {
	elements := []int{0} // < ignored dummy to simplify Next/Cur logic
	elements = append(elements, values...)
	slices.Sort(elements[1:]) // ensure sorted input
	return &listIterator{values: elements}
}

func (it *listIterator) Next() bool {
	if len(it.values) == 0 {
		return false
	}
	it.values = it.values[1:]
	return len(it.values) > 0
}

func (it *listIterator) Seek(value int) bool {
	pos, _ := slices.BinarySearch(it.values, value)
	it.values = it.values[pos:]
	return len(it.values) > 0
}

func (it *listIterator) Cur() int {
	return it.values[0]
}
