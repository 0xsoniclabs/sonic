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

type dominantSet map[idx.ValidatorID]struct{}

// ComputeDominantSet computes the dominant set of validators whose cumulative stake
// meets or exceeds the given threshold of the provided nominal stake.
//
// nominalStake is typically the total stake of all validators in the epoch.
// For outage resilience purposes, this function is to be called with the online validators
// but the nominal stake (the epoch validators stake).
//
// threshold is a real number (between 0 and 1) representing the fraction of the nominal stake
// that the dominant set's cumulative stake must meet or exceed.
//
// This function uses the [pos.Validators] object methods to have a deterministic order
// of validators with equal stakes.
func ComputeDominantSet(validators *pos.Validators, nominalStake pos.Weight, threshold float64) dominantSet {

	res := make(dominantSet)
	accumulated := pos.Weight(0)

	thresholdStake := pos.Weight(math.Ceil(float64(nominalStake) * threshold))

	// Compute prefix sum of stakes until the threshold stake is reached,
	// once reached, return the set of validators that contributed to it.
	for _, id := range validators.SortedIDs() {
		accumulated += validators.Get(id)
		res[id] = struct{}{}
		if accumulated >= pos.Weight(thresholdStake) {
			return res
		}
	}

	// If threshold not reached, return that there is no dominant set.
	return nil
}
