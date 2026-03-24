package pruner

// SafeEpochOracle is an interface that defines the methods required to
// determine which epochs are safe to prune. It provides a way to abstract the logic
// for determining the safe epoch threshold, allowing for different implementations
// based on specific requirements or criteria.
type SafeEpochOracle interface {
	// GetSafeEpoch returns the current safe epoch threshold.
	// Epochs older than this threshold are considered safe to prune.
	GetSafeEpoch() int
}

type fixedSafeEpochOracle struct {
	safeEpoch int
}

// NewFixedSafeEpochOracle creates a new instance of fixedSafeEpochOracle with
// the given safe epoch threshold.
func NewFixedSafeEpochOracle(safeEpoch int) SafeEpochOracle {
	return &fixedSafeEpochOracle{safeEpoch: safeEpoch}
}

// GetSafeEpoch returns the fixed safe epoch threshold.
func (o *fixedSafeEpochOracle) GetSafeEpoch() int {
	return o.safeEpoch
}
