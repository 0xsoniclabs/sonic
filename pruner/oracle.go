package pruner

import "github.com/Fantom-foundation/lachesis-base/inter/idx"

// SafeEpochOracle is an interface that defines the methods required to
// determine which epochs are safe to prune. It provides a way to abstract the logic
// for determining the safe epoch threshold, allowing for different implementations
// based on specific requirements or criteria.
type SafeEpochOracle interface {
	// GetSafeEpoch returns the current safe epoch threshold.
	// Epochs older than this threshold are considered safe to prune.
	GetSafeEpoch() idx.Epoch
}

type fixedSafeEpochOracle struct {
	safeEpoch idx.Epoch
}

// NewFixedSafeEpochOracle creates a new instance of fixedSafeEpochOracle with
// the given safe epoch threshold.
func NewFixedSafeEpochOracle(safeEpoch idx.Epoch) SafeEpochOracle {
	return &fixedSafeEpochOracle{safeEpoch: safeEpoch}
}

// GetSafeEpoch returns the fixed safe epoch threshold.
func (o *fixedSafeEpochOracle) GetSafeEpoch() idx.Epoch {
	return o.safeEpoch
}
