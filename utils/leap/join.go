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
	"iter"
	"sort"
)

//go:generate mockgen -source=join.go -package=leap -destination=join_mock.go

// Join implements the Leapfrog Triejoin algorithm to compute the intersection
// of multiple ordered iterators over comparable elements of type T.
//
// The returned iterator yields all elements that are present in all input
// iterators, in sorted order. If any input iterator is empty, the result is
// empty.
//
// Paper: Leapfrog Triejoin: A Simple, Worst-Case Optimal Join Algorithm
// https://arxiv.org/pdf/1210.0481
func Join[T cmp.Ordered](
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
			return iterators[a].Cur() < iterators[b].Cur()
		})

		// We cycle through the iterators, always advancing the one with the
		// smallest current element. When an iterator is advanced, we seek it to
		// at least the current largest element among all iterators. If all
		// iterators align on the same element, we yield it. This continues
		// until any iterator is exhausted.
		//
		// Note: the following loop retains all iterators sorted by the current
		// key they are pointing to when reading the list of iterators circularly
		// starting from p.

		largest := iterators[len(iterators)-1].Cur()
		for p := 0; ; p = (p + 1) % len(iterators) {
			// The iterator at position p points to the smallest element.
			iter := iterators[p]
			smallest := iter.Cur()
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
			largest = iter.Cur()
		}
	}
}

// Iterator defines a generic iterator over ordered unique elements of type T.
type Iterator[T any] interface {
	// Next advances the iterator to the next element. Initially, the iterator
	// is positioned before the first element, so Next must be called to advance
	// it to the first element. Next returns true if the iterator was advanced
	// to a valid element, and false if the iterator is exhausted.
	Next() bool

	// Cur returns the current element. It must only be called after Next or
	// Seek has returned true.
	Cur() T

	// Seek advances the iterator to the smallest element greater than or equal
	// to the target. It returns true if such an element was found, and false if
	// the iterator is exhausted.
	Seek(T) bool
}
