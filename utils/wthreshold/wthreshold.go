package wthreshold

import (
	"github.com/0xsoniclabs/consensus/inter/pos"
)

type WeightedValue interface {
	Weight() pos.Weight
}

// FindThresholdValue iterates through a slice of WeightedValues, accumulating total weight
// and returns the first WeightedValue with which the total weight exceeds a given threshold.
// If cumulative weight of all values is less than the threshold, panic.
func FindThresholdValue(values []WeightedValue, threshold pos.Weight) WeightedValue {
	// Calculate weighted threshold value
	var totalWeight pos.Weight
	for _, value := range values {
		totalWeight += value.Weight()
		if totalWeight >= threshold {
			return value
		}
	}
	panic("invalid threshold value")
}
