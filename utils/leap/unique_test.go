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
)

func TestUnique_FiltersDuplicates(t *testing.T) {
	tests := map[string]struct {
		input    []int
		expected []int
	}{
		"no duplicates": {
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		"with duplicates": {
			input:    []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		"mixed duplicates": {
			input:    []int{1, 1, 2, 3, 3, 4, 5, 5, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		"all duplicates": {
			input:    []int{1, 1, 1, 1, 1},
			expected: []int{1},
		},
		"empty input": {
			input:    []int{},
			expected: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := slices.Collect(All(Unique(newIter(tc.input...))))
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnique_Seek_SkipsElements(t *testing.T) {
	tests := map[string]struct {
		input       []int
		seekValue   int
		expectedCur int
		expectedOk  bool
	}{
		"seek existing value": {
			input:       []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5},
			seekValue:   3,
			expectedCur: 3,
			expectedOk:  true,
		},
		"seek non-existing value": {
			input:       []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5},
			seekValue:   6,
			expectedCur: 0,
			expectedOk:  false,
		},
		"seek to first value": {
			input:       []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5},
			seekValue:   1,
			expectedCur: 1,
			expectedOk:  true,
		},
		"seek to last value": {
			input:       []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5},
			seekValue:   5,
			expectedCur: 5,
			expectedOk:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			uniqueIter := Unique(newIter(tc.input...))
			ok := uniqueIter.Seek(tc.seekValue)
			require.Equal(t, tc.expectedOk, ok)

			if ok {
				cur := uniqueIter.Cur()
				require.Equal(t, tc.expectedCur, cur)
			}

			if uniqueIter.Next() {
				cur := uniqueIter.Cur()
				require.Greater(t, cur, tc.expectedCur)
			}
		})
	}
}
