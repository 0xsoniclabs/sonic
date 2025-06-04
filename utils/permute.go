package utils

import (
	"iter"
	"slices"
)

// Permute is a utility function that creates an iterator producing all
// permutations of the input slice. Due to the nature of permutations, the
// number of results grows factorially with the size of the input slice, which
// means that practical use is limited to small slices, in particular for test
// case generation.
//
// The resulting iterator produces one result at a time. However, the
// implementation is not optimized for performance nor memory usage. If you
// consider using this function in production code, consider improving its
// implementation.
func Permute[T any](list []T) iter.Seq[[]T] {
	list = slices.Clone(list) // clone to avoid modifying the original slice
	return func(yield func([]T) bool) {
		if len(list) == 0 {
			yield(list)
			return
		}
		for i := range list {
			list[0], list[i] = list[i], list[0] // swap
			for cur := range Permute(list[1:]) {
				if !yield(append([]T{list[0]}, cur...)) {
					return
				}
			}
			list[0], list[i] = list[i], list[0] // swap back
		}
	}
}
