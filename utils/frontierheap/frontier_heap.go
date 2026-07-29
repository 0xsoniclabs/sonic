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

package frontierheap

import (
	"container/heap"
	"slices"
)

// FrontierHeap is a max-heap over the frontier of a set of sequences (sequence
// heads / first elements). Each sequence's remaining elements become reachable
// one by one as the sequence is advanced via Shift.
type FrontierHeap[T any] struct {
	heap sequenceHeap[T]
}

// NewFrontierHeap creates a new max-heap with the given comparison function.
func NewFrontierHeap[T any](order func(a, b T) int) *FrontierHeap[T] {
	return &FrontierHeap[T]{heap: sequenceHeap[T]{order: order}}
}

// AddSequence inserts a sequence into the heap.
// Complexity: O(log s), where s is the number of sequences in the heap.
func (q *FrontierHeap[T]) AddSequence(sequence []T) {
	if len(sequence) == 0 {
		return
	}
	heap.Push(&q.heap, sequence)
}

// Peek returns the greatest frontier head, or a zero value and false if the
// heap is empty.
// Complexity: O(1).
func (q *FrontierHeap[T]) Peek() (T, bool) {
	if q.heap.Len() == 0 {
		var zero T
		return zero, false
	}
	return q.heap.sequences[0][0], true
}

// Shift removes and returns the greatest frontier head and advances its
// sequence: the next element of the same sequence, if any, joins the frontier.
// Returns a zero value and false if the heap is empty.
// Complexity: O(log s), where s is the number of sequences in the heap.
func (q *FrontierHeap[T]) Shift() (T, bool) {
	if q.heap.Len() == 0 {
		var zero T
		return zero, false
	}
	head := q.heap.sequences[0][0]
	if len(q.heap.sequences[0]) > 1 {
		q.heap.sequences[0] = q.heap.sequences[0][1:]
		heap.Fix(&q.heap, 0)
	} else {
		heap.Pop(&q.heap)
	}
	return head, true
}

// PopSequence removes the sequence whose head is the greatest frontier head and
// returns its remaining elements, the first of which is that head. Returns nil
// and false if the heap is empty.
// Complexity: O(log s), where s is the number of sequences in the heap.
func (q *FrontierHeap[T]) PopSequence() ([]T, bool) {
	if q.heap.Len() == 0 {
		return nil, false
	}
	return heap.Pop(&q.heap).([]T), true
}

// Copy returns a heap that can be consumed independently of this one. The
// sequence elements themselves are shared, not copied.
func (q *FrontierHeap[T]) Copy() *FrontierHeap[T] {
	// The sequences are only resliced, never mutated, so a deep copy of each
	// sequence is not necessary.
	return &FrontierHeap[T]{heap: sequenceHeap[T]{
		sequences: slices.Clone(q.heap.sequences),
		order:     q.heap.order,
	}}
}

// -- Heap Implementation --

// sequenceHeap is a heap of non-empty sequences ordered by their order function
// applied to their heads. It implements heap.Interface to be used with
// container/heap and provides only the plain heap operations, without the
// advancing of sequences performed by Shift. It is kept separate from
// FrontierHeap because the interface's Pop would collide with the public one,
// and its methods are building blocks that must not be called directly by
// users of the heap.
type sequenceHeap[T any] struct {
	sequences [][]T
	order     func(a, b T) int
}

func (h *sequenceHeap[T]) Len() int {
	return len(h.sequences)
}

func (h *sequenceHeap[T]) Less(i, j int) bool {
	return h.order(h.sequences[i][0], h.sequences[j][0]) > 0
}

func (h *sequenceHeap[T]) Swap(i, j int) {
	h.sequences[i], h.sequences[j] = h.sequences[j], h.sequences[i]
}

func (h *sequenceHeap[T]) Push(x any) {
	h.sequences = append(h.sequences, x.([]T))
}

func (h *sequenceHeap[T]) Pop() any {
	last := len(h.sequences) - 1
	sequence := h.sequences[last]
	h.sequences[last] = nil
	h.sequences = h.sequences[:last]
	return sequence
}
