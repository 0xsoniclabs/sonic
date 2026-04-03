// Copyright 2026 Sonic Operations Ltd
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

package wthreshold

import "github.com/0xsoniclabs/consensus/consensus"

type WeightedValue interface {
	Weight() consensus.Weight
}

// FindThresholdValue iterates through a slice of WeightedValues, accumulating total weight
// and returns the first WeightedValue with which the total weight exceeds a given threshold.
// If cumulative weight of all values is less than the threshold, panic.
func FindThresholdValue(values []WeightedValue, threshold consensus.Weight) WeightedValue {
	// Calculate weighted threshold value
	var totalWeight consensus.Weight
	for _, value := range values {
		totalWeight += value.Weight()
		if totalWeight >= threshold {
			return value
		}
	}
	panic("invalid threshold value")
}
