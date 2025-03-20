package schedule

import (
	"math"
)

// randomGenerator is a simple random number generator based on the xorshift*
// algorithm. It can be seeded and it is guaranteed to produce the same sequence
// of random numbers given the same seed.
// For details see: https://en.wikipedia.org/wiki/Xorshift#xorshift*
type randomGenerator struct {
	state uint64
}

// seed initializes the xorShiftStar random number generator with the given seed.
func (x *randomGenerator) seed(seed uint64) {
	x.state = seed
}

// next returns the next random number using the xorshift* algorithm. The result
// approximately uniformly distributed in the range [0, 2^64).
// Based on https://en.wikipedia.org/wiki/Xorshift#xorshift*
func (x *randomGenerator) next() uint64 {
	// make sure that the current state is not zero, as it might be if the
	// generator was not seeded or seeded with 0.
	if x.state == 0 {
		x.state = 1
	}
	const factor = uint64(0x2545F4914F6CDD1D)
	rand := x.state
	rand ^= rand >> 12
	rand ^= rand << 25
	rand ^= rand >> 27
	x.state = rand
	return rand * factor
}

// randN returns a uniformly sampled random number in the range [0, n) using
// the xorshift* algorithm. If n is zero, 0 is returned.
func (x *randomGenerator) randN(n uint64) uint64 {
	const uint64Max = math.MaxUint64

	// The next() function produces a random value r in the range [0, 2^64). A
	// simple way to get a random number in the range [0, n) is to take the
	// remainder of r divided by n. However, this introduces a bias if 2^64 is
	// not a multiple of n.
	//
	// For instance, let's assume next() would produces values in the range
	// [0, 2^4)=[0, 16) and we want to get a random number in the range [0, 3).
	// The following cases are possible:
	//    - r is 0, 3, 6, 9, 12, or 15  => the result is 0
	//    - r is 1, 4, 7, 10, or 13     => the result is 1
	//    - r is 2, 5, 8, 11, or 14     => the result is 2
	// This means that 0 is slightly more likely to be returned than 1 or 2.
	//
	// To fix this is to decide that in case of a random value r being greater
	// or equal than the largest multiple of n in the range covered by next(),
	// instead of returning r % n, we discard r and request a new random value.
	// This is repeated until we get a value below that limit.
	//
	// By deciding to treat too high values as undecided and repeating the
	// process, every value in the range [0, n) has the same probability of
	// being returned.
	//
	// The probability of producing a sequence of random values that are all
	// above the limit is very low for small values significantly smaller than
	// 2^64, quickly approaching zero, and thus practically negligible.

	// trivial cases (also avoids division by zero)
	if n <= 1 {
		return 0
	}

	// limit is at least uint64Max/2, thus the chances for a re-sampling are
	// at most 50%, resulting in a chance of 2^-k for k re-samplings.
	limit := uint64Max - (uint64Max % n)
	for {
		rand := x.next()
		if rand < limit {
			return rand % n
		}
	}
}
