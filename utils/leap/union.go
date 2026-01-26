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
	"sync"
)

func Union[T cmp.Ordered](
	iterators ...Iterator[T],
) *unionIterator[T] {
	return UnionFunc(cmp.Less[T], iterators...)
}

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
		it.initialized = true
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

func (it *unionIterator[T]) Cur() T {
	if len(it.heap.iters) == 0 {
		var zero T
		return zero
	}
	return it.heap.iters[0].Cur()
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

func (it *unionIterator[T]) progressAllIterators(
	progress func(Iterator[T]) bool,
) bool {
	var wg sync.WaitGroup
	wg.Add(len(it.heap.iters))
	remaining := make([]Iterator[T], len(it.heap.iters))
	for i, iter := range it.heap.iters {
		go func() {
			defer wg.Done()
			if progress(iter) {
				remaining[i] = iter
			} else {
				iter.Release()
			}
		}()
	}
	wg.Wait()

	iters := make([]Iterator[T], 0, len(remaining))
	for _, iter := range remaining {
		if iter != nil {
			iters = append(iters, iter)
		}
	}

	/*
		iters := make([]Iterator[T], 0, len(it.heap.iters))
		for _, iter := range it.heap.iters {
			if iter.Next() {
				iters = append(iters, iter)
			} else {
				iter.Release()
			}
		}
	*/

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
	return h.less(h.iters[i].Cur(), h.iters[j].Cur())
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
