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

package throttling

import (
	"math"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

type DominantSet map[idx.ValidatorID]struct{}

// TODO: docstring
//
// This function uses the [pos.Validators] object methods to have a deterministic order
// of validators with equal stakes.
func ComputeDominantSet(validators *pos.Validators, totalStake pos.Weight, threshold float64) DominantSet {

	res := make(DominantSet)
	accumulated := pos.Weight(0)

	thresholdStake := pos.Weight(math.Ceil(float64(totalStake) * threshold))

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
