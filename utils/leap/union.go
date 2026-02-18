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
	"container/heap"
	"sync"
)

// Union merges multiple ordered iterators over unique elements of type T into a
// single iterator that yields all elements present in any of the input
// iterators, in sorted order. The implementation assumes that the input
// iterators yield distinct elements in the natural ordering of type T (i.e., the
// ordering defined by the < operator).
//
// Duplicates yielded by different input iterators or by the same iterator are
// forwarded to the output iterator. If a lack of duplicates is desired, but
// can not be guaranteed by the input iterators, consider wrapping the result of
// Union with Unique.
//
// The resulting iterator takes ownership of the input iterators and will call
// Release on each of them as they are exhausted or when Release is called on
// the union iterator itself.
func Union[T cmp.Ordered](
	iterators ...Iterator[T],
) *unionIterator[T] {
	return UnionFunc(cmp.Less[T], iterators...)
}

// UnionFunc is a generalization of Union that accepts a custom less function to
// compare elements of type T that have no natural ordering or for which a
// custom ordering is desired. The less function must be consistent with the
// ordering of the input iterators, which is not checked by the implementation.
func UnionFunc[T any](
	less func(a, b T) bool,
	iterators ...Iterator[T],
) *unionIterator[T] {
	return &unionIterator[T]{
		heap: iteratorHeap[T]{
			iters: iterators,
			less:  less,
		},
	}
}

// unionIterator implements an iterator that yields the union of multiple input
// iterators. It maintains a min-heap of the input iterators based on their
// current value to efficiently retrieve the smallest current element across all
// iterators.
type unionIterator[T any] struct {
	heap        iteratorHeap[T]
	initialized bool
}

func (it *unionIterator[T]) Next() bool {
	if len(it.heap.iters) == 0 {
		return false
	}

	// The first time Next is called, we need to call Next on all iterators and
	// create the heap sorting them by their current value.
	if !it.initialized {
		it.progressAllIterators(Iterator[T].Next)
		return len(it.heap.iters) > 0
	}

	// In all other cases, Next is called on the smallest iterator. If it is
	// exhausted, we remove it from the heap, and the next iterator becomes
	// active. If not, it is re-inserted into the heap to maintain order.
	smallest := it.heap.iters[0]
	if smallest.Next() {
		heap.Fix(&it.heap, 0)
	} else {
		heap.Pop(&it.heap)
		smallest.Release()
	}
	return len(it.heap.iters) > 0
}

func (it *unionIterator[T]) Current() T {
	if len(it.heap.iters) == 0 {
		var zero T
		return zero
	}
	return it.heap.iters[0].Current()
}

func (it *unionIterator[T]) Seek(value T) bool {
	return it.progressAllIterators(func(iter Iterator[T]) bool {
		return iter.Seek(value)
	})
}

func (it *unionIterator[T]) Release() {
	for _, iter := range it.heap.iters {
		iter.Release()
	}
	it.heap.iters = nil
}

// progressAllIterators advances all iterators using the provided progress
// function (either Next or Seek). It removes any iterators that are exhausted
// after the operation. It returns true if there is at least one iterator
// remaining after the operation.
func (it *unionIterator[T]) progressAllIterators(
	progress func(Iterator[T]) bool,
) bool {
	// Advance all iterators in parallel. This is particularly beneficial if
	// there are many iterators that are I/O bound.
	var wg sync.WaitGroup
	remaining := make([]Iterator[T], len(it.heap.iters))
	for i, iter := range it.heap.iters {
		wg.Go(func() {
			if progress(iter) {
				remaining[i] = iter
			} else {
				iter.Release()
			}
		})
	}
	wg.Wait()

	// Collect the iterators that are still valid.
	iters := make([]Iterator[T], 0, len(remaining))
	for _, iter := range remaining {
		if iter != nil {
			iters = append(iters, iter)
		}
	}

	// Rebuild the heap with the remaining iterators.
	it.heap = iteratorHeap[T]{iters: iters, less: it.heap.less}
	heap.Init(&it.heap)
	it.initialized = true
	return len(iters) > 0
}

// -- Heap Implementation --

// iteratorHeap is a min-heap of iterators based on their current value. It
// implements heap.Interface to be used with container/heap.
type iteratorHeap[T any] struct {
	iters []Iterator[T]
	less  func(a, b T) bool
}

func (h *iteratorHeap[T]) Len() int {
	return len(h.iters)
}

func (h *iteratorHeap[T]) Less(i, j int) bool {
	return h.less(h.iters[i].Current(), h.iters[j].Current())
}

func (h *iteratorHeap[T]) Swap(i, j int) {
	h.iters[i], h.iters[j] = h.iters[j], h.iters[i]
}

func (h *iteratorHeap[T]) Push(x any) {
	h.iters = append(h.iters, x.(Iterator[T]))
}

func (h *iteratorHeap[T]) Pop() any {
	last := h.iters[len(h.iters)-1]
	h.iters = h.iters[:len(h.iters)-1]
	return last
}
