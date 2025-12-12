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
	"maps"
	"math/rand"
	"slices"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func TestComputeDominantSet_CanIdentifyWhenStakeDistributionIsDominated(t *testing.T) {
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stakes := makeValidators(test.stakes...)
			set := ComputeDominantSet(stakes, pos.Weight(100), testThreshold)
			require.ElementsMatch(t, test.expectedSet, slices.Collect(maps.Keys(set)))
		})
	}
}

func TestComputeDominantSet_DoesNotExistWhenThresholdNotMet(t *testing.T) {
	stakes := slices.Repeat([]int64{42}, 11)
	validators := makeValidators(stakes...)

	threshold := 1.01
	set := ComputeDominantSet(validators, validators.TotalWeight(), threshold)
	require.Empty(t, set)
}

func TestComputeDominantSet_IsIndependentFromStakeOrder(t *testing.T) {
	// The dominant set calculation does not sort validators by stake,
	// this is done by the Validators object itself. Nevertheless the code
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
			validators := makeValidators(test.stakes...)
			set := ComputeDominantSet(validators, validators.TotalWeight(), threshold)
			require.ElementsMatch(t, test.expectedSet, slices.Collect(maps.Keys(set)))
		})
	}
}

func TestComputeDominantSet_EquivalentStakes_IsDeterministic(t *testing.T) {
	// The dominant set calculation does not sort validators by stake,
	// this is done by the Validators object itself. Nevertheless the code
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

		set := ComputeDominantSet(validators, validators.TotalWeight(), 0.7)
		ids := make([]idx.ValidatorID, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}

		require.ElementsMatch(t, []idx.ValidatorID{1, 2, 3, 4, 5, 6, 7}, ids)
	}
}

func makeValidators(stakes ...int64) *pos.Validators {
	builder := pos.NewBuilder()
	for i, stake := range stakes {
		builder.Set(idx.ValidatorID(i+1), pos.Weight(stake))
	}
	return builder.Build()
}
