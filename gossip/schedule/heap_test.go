package schedule

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeap_ElementsAreSorted(t *testing.T) {
	const N = 10

	// Create a shuffled list of entries and add them to the queue.
	entries := make([]int, N)
	for i := range N {
		entries[i] = i
	}
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
	queue := newHeap[int](func(a, b int) int {
		return b - a
	})
	for _, e := range entries {
		queue.Add(e, e)
	}

	// Pop elements from the queue and check that they are sorted.
	for i := range entries {
		b, ok := queue.Peek()
		if !ok {
			t.Fatal("expected to peek an element")
		}
		if want, got := i, b; want != got {
			t.Errorf("expected to peek element with number %d, got %v", want, got)
		}

		b, ok = queue.Pop()
		if !ok {
			t.Fatal("expected to pop an element")
		}
		if want, got := i, b; want != got {
			t.Errorf("expected to pop element with number %d, got %v", want, got)
		}
	}

	if _, ok := queue.Peek(); ok {
		t.Fatal("expected to peek no more elements")
	}

	if _, ok := queue.Pop(); ok {
		t.Fatal("expected to pop no more elements")
	}
}

func TestHeap_CanUpdateValue(t *testing.T) {
	require := require.New(t)
	queue := newHeap[int](func(a, b int) int {
		return a - b
	})

	queue.Add(1, 1)
	queue.Add(2, 2)
	queue.Add(3, 3)

	got, found := queue.Peek()
	require.True(found)
	require.Equal(3, got)

	if !queue.Update(2, 4) {
		t.Fatal("expected to update element 2")
	}

	got, found = queue.Peek()
	require.True(found)
	require.Equal(4, got)

	if !queue.Update(2, 2) {
		t.Fatal("expected to update element 2")
	}

	got, found = queue.Peek()
	require.True(found)
	require.Equal(3, got)
}

func TestHeap_Update_UpdateOfNonExistingValuesAreIgnored(t *testing.T) {
	require := require.New(t)
	queue := newHeap[int](func(a, b int) int {
		return a - b
	})

	queue.Add(1, 1)
	queue.Add(2, 2)
	require.True(queue.Update(2, 4))
	require.False(queue.Update(3, 3))
}

func TestHeap_ZeroHeapCanBeUsedToStoreAndRetrieveElements(t *testing.T) {
	queue := heap[int, int]{}

	for i := 0; i < 10; i++ {
		queue.Add(i, i)
	}

	retrieved := []int{}
	for cur, ok := queue.Pop(); ok; cur, ok = queue.Pop() {
		retrieved = append(retrieved, cur)
	}

	if want, got := 10, len(retrieved); want != got {
		t.Fatalf("expected to get %d elements, got %d", want, got)
	}

	slices.Sort(retrieved)
	for i, cur := range retrieved {
		if want, got := i, cur; want != got {
			t.Errorf("expected to get element %d, got %d", want, got)
		}
	}
}
