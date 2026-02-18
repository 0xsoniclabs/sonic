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

package leap

import (
	"cmp"
	"iter"
	"sort"
)

// Join implements the Leapfrog Triejoin algorithm to compute the intersection
// of multiple ordered iterators over unique elements of type T.
//
// The returned iterator yields all elements that are present in all input
// iterators, in sorted order. Consequently, if any input iterator is empty,
// the result is empty.
//
// Note: elements in the incoming iterators must be ordered according to the
// natural ordering of type T (i.e., the ordering defined by the < operator).
// Also, there must be no duplicate elements within each input iterator. These
// preconditions are not checked by Join and must be ensured by the caller.
//
// Paper: Leapfrog Triejoin: A Simple, Worst-Case Optimal Join Algorithm
// https://arxiv.org/pdf/1210.0481
func Join[T cmp.Ordered](
	iterators ...Iterator[T],
) iter.Seq[T] {
	return JoinFunc(cmp.Less[T], iterators...)
}

// JoinFunc is a generalization of Join that accepts a custom less function to
// compare elements of type T that have no natural ordering or for which a
// custom ordering is desired. The less function must be consistent with the
// ordering of the input iterators.
func JoinFunc[T comparable](
	less func(a, b T) bool,
	iterators ...Iterator[T],
) iter.Seq[T] {
	return func(yield func(T) bool) {
		if len(iterators) == 0 {
			return
		}

		// Initialize the iterators.
		for _, it := range iterators {
			if !it.Next() {
				return // If any iterator is empty, the join is empty.
			}
		}

		// Sort iterators by their current elements, smallest first.
		sort.Slice(iterators, func(a, b int) bool {
			return less(iterators[a].Current(), iterators[b].Current())
		})

		// We cycle through the iterators, always advancing the one with the
		// smallest current element. When an iterator is advanced, we seek it to
		// at least the current largest element among all iterators. If all
		// iterators align on the same element, we yield it. This continues
		// until any iterator is exhausted.
		//
		// Note: the following loop retains all iterators sorted by the current
		// key they are pointing to when reading the list of iterators like a
		// ring buffer starting at position p. The iterator at position p is
		// always the one with the smallest current key, and the one at
		// position (p-1) mod len(iterators) is the one with the largest key.

		largest := iterators[len(iterators)-1].Current()
		for p := 0; ; p = (p + 1) % len(iterators) {
			// The iterator at position p points to the smallest element.
			iter := iterators[p]
			smallest := iter.Current()
			if smallest == largest { // All iterators are aligned.
				if !yield(largest) {
					return
				}
				if !iter.Next() {
					return
				}
			} else {
				// Advance the iterator to at least the candidate key.
				if !iter.Seek(largest) {
					return
				}
			}
			largest = iter.Current()
		}
	}
}
