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
	"math"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// dominantSet represents a set of validator IDs which cannot skip event emission.
type dominantSet map[idx.ValidatorID]struct{}

// computeDominantSet computes the dominant set of validators whose cumulative stake
// meets or exceeds the given stake threshold.
//
// In case that the threshold cannot be met, it returns the full set of validators.
// In this case, the sum of all validators' stakes is less than the threshold, the
// set is returned nevertheless because these validators cannot skip event emission.
//
// This function uses the [pos.Validators] object methods to have a deterministic order
// of validators with equal stakes.
func computeDominantSet(validators *pos.Validators, neededStake pos.Weight) dominantSet {

	res := make(dominantSet)
	accumulated := pos.Weight(0)

	// Compute prefix sum of stakes until the threshold stake is reached,
	// once reached, return the set of validators that contributed to it.
	for _, id := range validators.SortedIDs() {
		if accumulated >= neededStake {
			return res
		}
		accumulated += validators.Get(id)
		res[id] = struct{}{}
	}

	// If the threshold stake is not reached, return all validators.
	return res
}

// computeNeededStake computes the stake needed to meet the given threshold.
// It can be used to compute the needed stake for dominant set calculation.
func computeNeededStake(stake pos.Weight, threshold float64) pos.Weight {
	return pos.Weight(math.Ceil(float64(stake) * threshold))
}
