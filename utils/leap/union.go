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
)

func Union[T cmp.Ordered](
	iterators ...Iterator[T],
) *unionIterator[T] {
	return &unionIterator[T]{
		iterators: iterators,
	}
}

type unionIterator[T cmp.Ordered] struct {
	iterators   iteratorHeap[T]
	initialized bool
}

func (it *unionIterator[T]) Next() bool {
	if len(it.iterators) == 0 {
		return false
	}

	// The first time Next is called, we need to call Next on all iterators and
	// create the heap sorting them by their current value.
	if !it.initialized {
		iters := make([]Iterator[T], 0, len(it.iterators))
		for _, iter := range it.iterators {
			if iter.Next() {
				iters = append(iters, iter)
			}
		}
		it.iterators = iteratorHeap[T](iters)
		heap.Init(&it.iterators)
		it.initialized = true
		return len(it.iterators) > 0
	}

	// In all other cases, Next is called on the smallest iterator. If it is
	// exhausted, we remove it from the heap, and the next iterator becomes
	// active. If not, it is re-inserted into the heap to maintain order.
	smallest := it.iterators[0]
	if smallest.Next() {
		heap.Fix(&it.iterators, 0)
	} else {
		heap.Pop(&it.iterators)
	}
	return len(it.iterators) > 0
}

func (it *unionIterator[T]) Cur() T {
	if len(it.iterators) == 0 {
		var zero T
		return zero
	}
	return it.iterators[0].Cur()
}

func (it *unionIterator[T]) Seek(value T) bool {
	remaining := make([]Iterator[T], 0, len(it.iterators))
	hasNext := false
	for _, iter := range it.iterators {
		if iter.Seek(value) {
			remaining = append(remaining, iter)
			hasNext = true
		}
	}
	it.iterators = iteratorHeap[T](remaining)
	heap.Init(&it.iterators)
	it.initialized = true
	return hasNext
}

// -- Heap Implementation --

// iteratorHeap is a min-heap of iterators based on their current value. It
// implements heap.Interface to be used with container/heap.
type iteratorHeap[T cmp.Ordered] []Iterator[T]

func (h iteratorHeap[T]) Len() int {
	return len(h)
}

func (h iteratorHeap[T]) Less(i, j int) bool {
	return h[i].Cur() < h[j].Cur()
}

func (h iteratorHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *iteratorHeap[T]) Push(x any) {
	*h = append(*h, x.(Iterator[T]))
}

func (h *iteratorHeap[T]) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
