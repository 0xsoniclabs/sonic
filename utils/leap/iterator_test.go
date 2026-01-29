package leap

import (
	"cmp"
	"iter"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// This file contains tests and utilities for the Iterator interface used in
// this package for testing the provided algorithms.

// -- Iterator to slice conversion --

// All converts an Iterator into an iter.Seq, allowing iteration using
// the iter package's conventions.
func All[T any](iter Iterator[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for iter.Next() {
			if !yield(iter.Current()) {
				return
			}
		}
	}
}

func TestAll_ListsAllElements(t *testing.T) {
	ctrl := gomock.NewController(t)
	iter := NewMockIterator[int](ctrl)

	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Current().Return(1),
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Current().Return(2),
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Current().Return(3),
		iter.EXPECT().Next().Return(false),
	)

	require.Equal(t, []int{1, 2, 3}, slices.Collect(All(iter)))
}

// -- Fake Iterator --

// listIterator is a simple generic iterator implementations for testing.
type listIterator[T cmp.Ordered] struct {
	values []T
	pos    int
}

// newIter creates a new listIterator over the given values.
func newIter[T cmp.Ordered](values ...T) *listIterator[T] {
	slices.SortFunc(values, cmp.Compare)
	return &listIterator[T]{values: values, pos: -1}
}

func (it *listIterator[T]) Next() bool {
	if len(it.values) == it.pos+1 {
		return false
	}
	it.pos++
	return it.pos < len(it.values)
}

func (it *listIterator[T]) Seek(value T) bool {
	newPos, _ := slices.BinarySearchFunc(it.values, value, cmp.Compare)
	it.pos = max(it.pos, newPos)
	return it.pos < len(it.values)
}

func (it *listIterator[T]) Current() T {
	return it.values[it.pos]
}

func (it *listIterator[T]) Release() {
	// noop
}

func TestListIterator_ListsAllElements(t *testing.T) {
	tests := [][]int{
		nil,
		{},
		{1, 2, 3},
		{1, 1, 2, 2, 3, 3},
	}

	for _, input := range tests {
		iter := newIter(input...)
		result := slices.Collect(All(iter))
		want := input
		if len(want) == 0 {
			want = nil
		}
		require.Equal(t, want, result)
	}
}

func TestListIterator_Seek_CanSkipElements(t *testing.T) {
	require := require.New(t)

	iter := newIter(1, 2, 3, 4, 5, 8, 10)

	require.True(iter.Next())
	require.Equal(1, iter.Current())

	// Seek forward ...
	require.True(iter.Seek(3))
	require.Equal(3, iter.Current())

	require.True(iter.Seek(5))
	require.Equal(5, iter.Current())

	require.True(iter.Seek(7))
	require.Equal(8, iter.Current())

	// Seek backward is ignored.
	require.True(iter.Seek(2))
	require.Equal(8, iter.Current())

	require.True(iter.Seek(-2))
	require.Equal(8, iter.Current())

	// Seek past end ...
	require.False(iter.Seek(12))
	iter.Release()
}
