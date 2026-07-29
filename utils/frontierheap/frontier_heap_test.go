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
	"cmp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFrontierHeap_AddSequence_EmptySequenceIsNoOp(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.AddSequence(nil)
	require.Empty(t, h.heap.sequences)
}

func TestFrontierHeap_AddSequence_GreatestHeadIsAtTheRoot(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	for _, head := range []int{2, 4, 5, 3} {
		h.AddSequence([]int{head})
	}
	require.Len(t, h.heap.sequences, 4)
	require.Equal(t, 5, h.heap.sequences[0][0])
}

func TestFrontierHeap_Peek_ReturnsGreatestFrontierHeadWithoutConsumingIt(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])

	h.AddSequence([]int{1, 5})
	got, ok := h.Peek()
	require.True(t, ok)
	require.Equal(t, 1, got)

	h.AddSequence([]int{3})
	got, ok = h.Peek()
	require.True(t, ok)
	require.Equal(t, 3, got)
}

func TestFrontierHeap_Peek_EmptyHeapReturnsFalse(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	_, ok := h.Peek()
	require.False(t, ok)
}

func TestFrontierHeap_Shift_DrainsFrontiersInDescendingOrder(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.AddSequence([]int{9, 2, 8})
	h.AddSequence([]int{7, 3})
	h.AddSequence([]int{6})
	require.Equal(t, []int{9, 7, 6, 3, 2, 8}, drain(h))
}

func TestFrontierHeap_Shift_RemovesHeadOfFirstSequence(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.AddSequence([]int{9, 2, 8})

	require.Equal(t, [][]int{{9, 2, 8}}, h.heap.sequences)
	h.Shift()
	require.Equal(t, [][]int{{2, 8}}, h.heap.sequences)
	h.Shift()
	require.Equal(t, [][]int{{8}}, h.heap.sequences)
}

func TestFrontierHeap_Shift_EmptyHeapReturnsFalse(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	_, ok := h.Shift()
	require.False(t, ok)
}

func TestFrontierHeap_PopSequence_RemovesSequenceOfGreatestHead(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.AddSequence([]int{5, 4, 3})
	h.AddSequence([]int{2})
	got, ok := h.PopSequence()
	require.True(t, ok)
	require.Equal(t, []int{5, 4, 3}, got)
	require.Equal(t, [][]int{{2}}, h.heap.sequences)
}

func TestFrontierHeap_PopSequence_ReturnsFullFirstSequence(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.heap.sequences = [][]int{{4, 3}, {2, 5}}
	got, ok := h.PopSequence()
	require.True(t, ok)
	require.Equal(t, []int{4, 3}, got)
	got, ok = h.PopSequence()
	require.True(t, ok)
	require.Equal(t, []int{2, 5}, got)
}

func TestFrontierHeap_PopSequence_EmptyHeapReturnsFalse(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	_, ok := h.PopSequence()
	require.False(t, ok)
}

func TestFrontierHeap_Copy_IsConsumedIndependentlyOfTheOriginal(t *testing.T) {
	h := NewFrontierHeap(cmp.Compare[int])
	h.AddSequence([]int{5, 4})
	h.AddSequence([]int{3})

	copy := h.Copy()
	require.Equal(t, []int{5, 4, 3}, drain(h))
	require.Equal(t, []int{5, 4, 3}, drain(copy))
}

func drain[T any](h *FrontierHeap[T]) []T {
	var out []T
	for v, ok := h.Shift(); ok; v, ok = h.Shift() {
		out = append(out, v)
	}
	return out
}
