package ancestor

import (
	"math/rand"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
)

/*
 * RandomStrategy
 */

// RandomStrategy is used in tests, when vector clock isn't available
type RandomStrategy struct {
	r *rand.Rand
}

func NewRandomStrategy(r *rand.Rand) *RandomStrategy {
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec
	}
	return &RandomStrategy{
		r: r,
	}
}

// Choose chooses the hash from the specified options
func (st *RandomStrategy) Choose(_ consensus.EventHashes, options consensus.EventHashes) int {
	return st.r.Intn(len(options))
}
