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

package throttler

import (
	"fmt"
	"maps"
	"math"
	"math/rand"
	"slices"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func TestComputeDominantSet_IdentifiesDominantSet_WhenStakeDistributionIsDominated(t *testing.T) {

	// This test covers various basic cases for ease of development

	const testThreshold = 0.75

	tests := map[string]struct {
		stakes      []int64
		expectedSet []idx.ValidatorID
	}{
		"no validators": {
			stakes: nil,
		},
		"single validator": {
			stakes:      []int64{100},
			expectedSet: []idx.ValidatorID{1},
		},
		"two equal validators": {
			stakes:      []int64{50, 50},
			expectedSet: []idx.ValidatorID{1, 2},
		},
		"two validators one dominant": {
			stakes:      []int64{80, 20},
			expectedSet: []idx.ValidatorID{1},
		},
		"three validators one dominant": {
			stakes:      []int64{80, 10, 10},
			expectedSet: []idx.ValidatorID{1},
		},
		"three validators two dominant": {
			stakes:      []int64{40, 40, 20},
			expectedSet: []idx.ValidatorID{1, 2},
		},
		"four equal validators, first three dominate": {
			stakes:      []int64{25, 25, 25, 25},
			expectedSet: []idx.ValidatorID{1, 2, 3},
		},
		"one unit below threshold": {
			stakes:      []int64{25, 25, 24, 24, 2},
			expectedSet: []idx.ValidatorID{1, 2, 3, 4},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stakes := makeValidatorsFromStakes(test.stakes...)
			threshold := computeNeededStake(pos.Weight(100), testThreshold)
			set := computeDominantSet(stakes, threshold)
			require.ElementsMatch(t, test.expectedSet, slices.Collect(maps.Keys(set)))
		})
	}
}

func TestComputeDominantSet_UniformDistributionsAreDominated(t *testing.T) {
	// A set of n validators with equal stake shall have a dominant set
	// of size ceil(n * threshold). Because rounding of float, the set may
	// may be exceeded by one validator.

	for threshold := float64(0.01); threshold <= 1.; threshold += 0.01 {
		for n := 1; n <= 100; n++ {
			t.Run(fmt.Sprintf("n=%d,threshold=%.2f", n, threshold), func(t *testing.T) {
				validators := makeValidatorsFromStakes(slices.Repeat([]int64{10}, n)...)
				StakeThreshold := computeNeededStake(validators.TotalWeight(), threshold)
				set := computeDominantSet(validators, StakeThreshold)

				expectedCount := int(float64(n) * threshold)
				require.NotEmpty(t, set, "dominating set cannot be empty")
				require.GreaterOrEqual(t, len(set), expectedCount)
				require.GreaterOrEqual(t, sumStake(set, validators), StakeThreshold)
			})
		}
	}
}

func TestComputeDominantSet_ReturnsInputValidatorsSet_WhenStakeCannotBeMet(t *testing.T) {

	for validatorCount := 1; validatorCount <= 10; validatorCount++ {
		validators := makeValidatorsFromStakes(slices.Repeat([]int64{10}, validatorCount)...)
		neededStake := 10*validatorCount + 1

		set := computeDominantSet(validators, pos.Weight(neededStake))
		require.ElementsMatch(t,
			validators.IDs(),
			slices.Collect(maps.Keys(set)),
		)
	}
}

func TestComputeDominantSet_ZeroThresholdResultsInEmptyDominantSet(t *testing.T) {
	validators := makeValidatorsFromStakes(10, 20, 30)
	set := computeDominantSet(validators, 0)
	require.Empty(t, set, "dominant set shall be empty for zero threshold")
}

func TestComputeDominantSet_IsIndependentFromStakeOrder(t *testing.T) {
	// The dominant set calculation does not sort validators by stake,
	// this is done by the [pos.Validators] object itself. Nevertheless the code
	// is highly dependent on this behavior, so we test it here.

	tests := map[string]struct {
		stakes      []int64
		expectedSet []idx.ValidatorID
	}{
		"ascending": {
			stakes:      []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectedSet: []idx.ValidatorID{5, 6, 7, 8, 9, 10},
		},
		"descending": {
			stakes:      []int64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			expectedSet: []idx.ValidatorID{1, 2, 3, 4, 5, 6},
		},
		"random": {
			stakes:      []int64{3, 7, 2, 9, 1, 8, 4, 6, 10, 5},
			expectedSet: []idx.ValidatorID{2, 4, 6, 8, 9, 10},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sum := int64(0)
			for _, stake := range test.stakes {
				sum += stake
			}
			threshold := 0.75

			// Create validators in ascending order
			validators := makeValidatorsFromStakes(test.stakes...)
			set := computeDominantSet(validators, computeNeededStake(validators.TotalWeight(), threshold))
			require.ElementsMatch(t, test.expectedSet, slices.Collect(maps.Keys(set)))
		})
	}
}

func TestComputeDominantSet_IsDeterministic(t *testing.T) {
	// The dominant set calculation does not sort validators by stake,
	// this is done by the [pos.Validators] object itself. Nevertheless the code
	// is highly dependent on this behavior, so we test it here.

	// make test deterministic
	rand := rand.New(rand.NewSource(42))

	testInput := []struct {
		id    idx.ValidatorID
		stake int64
	}{
		{1, 10},
		{2, 10},
		{3, 10},
		{4, 10},
		{5, 10},
		{6, 10},
		{7, 10},
		{8, 10},
		{9, 10},
		{10, 10},
	}

	for range 1000 {
		rand.Shuffle(len(testInput), func(i, j int) {
			testInput[i], testInput[j] = testInput[j], testInput[i]
		})

		builder := pos.NewBuilder()
		for _, validator := range testInput {
			builder.Set(validator.id, pos.Weight(validator.stake))
		}
		validators := builder.Build()

		set := computeDominantSet(validators, computeNeededStake(validators.TotalWeight(), 0.7))
		ids := make([]idx.ValidatorID, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}

		require.ElementsMatch(t, []idx.ValidatorID{1, 2, 3, 4, 5, 6, 7}, ids)
	}
}

func makeValidatorsFromStakes(stakes ...int64) *pos.Validators {
	builder := pos.NewBuilder()
	for i, stake := range stakes {
		builder.Set(idx.ValidatorID(i+1), pos.Weight(stake))
	}
	return builder.Build()
}

func FuzzDominantSet(f *testing.F) {

	for threshold := float64(0.0); threshold <= 1.0; threshold += 0.05 {
		f.Add([]byte{255}, threshold)
		f.Add([]byte{1}, threshold)
		f.Add([]byte{255, 255, 0, 0}, threshold)
		f.Add([]byte{10, 20, 30, 40}, threshold)
	}

	f.Fuzz(func(t *testing.T, byteStakes []byte, threshold float64) {
		if threshold < 0.01 || threshold > 1.0 {
			return
		}

		// pos.Validator imposes a limit on the maximum total stake value
		if len(byteStakes)*255 > math.MaxUint32/2 {
			return
		}

		stakes := make([]int64, len(byteStakes))
		for i, b := range byteStakes {
			stakes[i] = int64(b)
		}
		validators := makeValidatorsFromStakes(stakes...)

		if validators.Len() == 0 {
			// if seed produces empty validators set, skip
			return
		}

		set := computeDominantSet(validators, computeNeededStake(validators.TotalWeight(), threshold))
		require.NotEmpty(t, set, "dominating set cannot be empty")

		// sum the stakes in the dominant set
		dominantStake := sumStake(set, validators)
		dominantThreshold := pos.Weight(math.Ceil(
			float64(validators.TotalWeight()) * threshold))
		require.GreaterOrEqual(t, dominantStake, dominantThreshold)
	})
}

func sumStake(set dominantSet, validators *pos.Validators) pos.Weight {
	dominantStake := pos.Weight(0)
	for id := range set {
		dominantStake += validators.Get(id)
	}
	return dominantStake
}

func TestComputeNeededStake_isRoundedUp(t *testing.T) {
	tests := map[string]struct {
		stake     pos.Weight
		threshold float64
		expected  pos.Weight
	}{
		"exact": {
			stake:     100,
			threshold: 0.5,
			expected:  50,
		},
		"zero threshold": {
			stake:     100,
			threshold: 0.0,
			expected:  0,
		},
		"full threshold": {
			stake:     100,
			threshold: 1.0,
			expected:  100,
		},
		"round up prime": {
			stake:     7,
			threshold: 0.50,
			expected:  4,
		},
		"round up with insufficient precision": {
			stake:     100,
			threshold: 0.3333333,
			expected:  34,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			needed := computeNeededStake(test.stake, test.threshold)
			require.Equal(t, int(test.expected), int(needed))
		})
	}
}
