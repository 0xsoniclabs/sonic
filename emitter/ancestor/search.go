package ancestor

import "github.com/0xsoniclabs/consensus/consensus"

// SearchStrategy defines a criteria used to estimate the "best" subset of parents to emit event with.
type SearchStrategy interface {
	// Choose chooses the hash from the specified options
	Choose(existingParents consensus.EventHashes, options consensus.EventHashes) int
}

// ChooseParents returns estimated parents subset, according to provided strategy
// max is max num of parents to link with (including self-parent)
// returns set of parents to link, len(res) <= max
func ChooseParents(existingParents consensus.EventHashes, options consensus.EventHashes, strategies []SearchStrategy) consensus.EventHashes {
	optionsSet := options.Set()
	parents := make(consensus.EventHashes, 0, len(strategies)+len(existingParents))
	parents = append(parents, existingParents...)
	for _, p := range existingParents {
		optionsSet.Erase(p)
	}

	for i := 0; i < len(strategies) && len(optionsSet) > 0; i++ {
		curOptions := optionsSet.Slice() // shuffle options
		best := strategies[i].Choose(parents, curOptions)
		parents = append(parents, curOptions[best])
		optionsSet.Erase(curOptions[best])
	}

	return parents
}
