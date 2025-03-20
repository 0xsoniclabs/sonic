package schedule

import (
	"fmt"
	"math"
	"testing"
)

func TestRandomGenerator_Seed_LeadsToSameSequence(t *testing.T) {
	for seed := range uint64(10) {
		gen1 := randomGenerator{}
		gen2 := randomGenerator{}
		gen1.seed(seed)
		gen2.seed(seed)
		for range 10 {
			if gen1.next() != gen2.next() {
				t.Fatal("different sequence")
			}
		}
	}
}

func TestRandomGenerator_SamplesRangeUniformly(t *testing.T) {
	const S = 10_000
	for n := range 5 {
		n := n + 1
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			samples := make([]int, n)
			gen := randomGenerator{}
			for range S {
				samples[gen.randN(uint64(n))]++
			}

			min := S
			max := 0
			for _, s := range samples {
				if s < min {
					min = s
				}
				if s > max {
					max = s
				}
			}

			if float64(max)/float64(min) > 1.05 {
				t.Errorf("max/min ratio too high: %v", float64(max)/float64(min))
			}
		})
	}
}

func TestRandomGenerator_randN_CornerCases(t *testing.T) {
	limit := []uint64{
		1,
		math.MaxInt64,
		uint64(math.MaxInt64) + 1, // maximizes the chance for re-sampling to ~50%
		math.MaxUint64 - 1,
		math.MaxUint64,
	}
	for _, n := range limit {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			gen := randomGenerator{}
			for range 10 {
				if got := gen.randN(n); got >= n {
					t.Errorf("got %d, want < %d", got, n)
				}
			}
		})
	}
}
