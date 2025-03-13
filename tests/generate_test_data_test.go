package tests

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestDataBasedOnModificationCombinations generates all possible versions of a
// given type based on the combinations of modifications.
// The iterator works around a function modify(T, []Piece) T, which shall modify
// an newly constructed instance of T with the provided piece-modifiers.
//
// Arguments:
//   - constructor: a function that constructs a new instance of T, for each version
//     to be based on an unmodified instance.
//   - pieces: a list of lists of pieces, where each list of pieces represents a
//     domain of possible modifications.
//   - modify: a function that modifies an instance of T with the provided pieces.
//
// Returns:
// - an iterator that yields all possible versions of T based on the combinations
func generateTestDataBasedOnModificationCombinations[T any, Piece any](
	constructor func() T,
	pieces [][]Piece,
	modify func(tx T, modifier []Piece) T,
) iter.Seq[T] {

	return func(yield func(data T) bool) {
		_cartesianProductRecursion(nil, pieces,
			func(pieces []Piece) bool {
				v := constructor()
				v = modify(v, pieces)
				return yield(v)
			})
	}
}

func TestCartesianProduct_CountInstantiations(t *testing.T) {

	count := func(_ int, modifier []int) int {
		var count int
		for _, v := range modifier {
			count += v
		}
		return count
	}

	countInstances := func(pieces [][]int, modifier func(int, []int) int) int {
		var count int
		makeZero := func() int { return 0 }
		for _ = range generateTestDataBasedOnModificationCombinations(makeZero, pieces, modifier) {
			count++
		}
		return count
	}

	assert.Equal(t, 1, countInstances(nil, count))
	assert.Equal(t, 1, countInstances([][]int{{1}}, count))
	assert.Equal(t, 2, countInstances([][]int{{1}, {1, 2}}, count))
	assert.Equal(t, 4, countInstances([][]int{{1, 2}, {1, 2}}, count))
	assert.Equal(t, 4, countInstances([][]int{{1}, {1, 2}, {1, 2}}, count))
	assert.Equal(t, 6, countInstances([][]int{{1}, {1, 2, 3}, {1, 2}}, count))
	assert.Equal(t, 6, countInstances([][]int{{1}, {1, 2}, {1, 2, 3}}, count))
}

func TestCartesianProduct_noPiecesReturnOriginalObject(t *testing.T) {
	makeOriginal := func() int { return 36 }
	it := generateTestDataBasedOnModificationCombinations(
		makeOriginal,
		nil,
		func(v int, pieces []int) int {
			require.Len(t, pieces, 0)
			return v
		},
	)

	versions := make([]int, 0)
	for i := range it {
		versions = append(versions, i)
	}

	require.Equal(t, 1, len(versions))
	require.Equal(t, makeOriginal(), versions[0])
}

func TestCartesianProduct_AcceptsFunctionsAsPieces(t *testing.T) {

	type TestType struct {
		A int
		B int
	}
	type modFunc func(t *TestType)

	setA := func(a int) modFunc {
		return func(t *TestType) {
			t.A = a
		}
	}
	setB := func(a int) modFunc {
		return func(t *TestType) {
			t.B = a
		}
	}

	instances := make([]TestType, 0)
	for v := range generateTestDataBasedOnModificationCombinations(
		func() TestType { return TestType{} },
		[][]modFunc{
			{setA(1), setA(2)},
			{setB(1), setB(2)},
		},
		func(t TestType, modifiers []modFunc) TestType {
			for _, m := range modifiers {
				m(&t)
			}
			return t
		}) {
		instances = append(instances, v)
	}

	assert.Len(t, instances, 4)
	assert.Contains(t, instances, TestType{1, 1})
	assert.Contains(t, instances, TestType{1, 2})
	assert.Contains(t, instances, TestType{2, 1})
	assert.Contains(t, instances, TestType{2, 2})
}

func _cartesianProductRecursion[T any](current []T, elements [][]T, callback func(data []T) bool) bool {
	if len(elements) == 0 {
		return callback(current)
	}

	var next [][]T
	if len(elements) > 1 {
		next = elements[1:]
	}

	for _, element := range elements[0] {
		if !_cartesianProductRecursion(append(current, element), next, callback) {
			return false
		}
	}
	return true
}
