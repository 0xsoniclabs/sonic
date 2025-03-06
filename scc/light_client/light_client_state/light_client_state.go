package lc_state

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// State  holds the current state of the light client.
type State struct {
	committee  scc.Committee
	period     scc.Period
	headNumber idx.Block
	headHash   common.Hash
}

// NewState creates a new State with the given committee.
func NewState(committee scc.Committee) *State {
	return &State{
		committee: committee,
	}
}

// Head returns the block number of the latest known block.
func (s *State) Head() idx.Block {
	return s.headNumber
}

// Sync updates the light client state using data from the provider.
// This serves as the primary process for synchronizing the light client state
// with the network.
// If success, the local client will reflect the most recent block and
// its corresponding certification committee.
func (s *State) Sync(provider provider.Provider) (idx.Block, error) {

	// Get the latest block number from the provider.

	// get period for the latest block

	// sync from current to latest

	// verify latest block certificate with latest committee

	// return the latest block number
	return idx.Block(0), nil
}
