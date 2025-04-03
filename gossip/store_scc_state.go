package gossip

import (
	"iter"

	"github.com/0xsoniclabs/sonic/scc/node"
	"github.com/0xsoniclabs/sonic/utils/result"
)

// GetLatestCertificationChainState retrieves the most recent certification
// chain state available in the store. States are ordered by their block
// height. If there is none, an error is returned.
func (s *Store) GetLatestCertificationChainState() (node.State, error) {
	key, err := findHighestKey(s.table.CertificationChainStates)
	if err != nil {
		return node.State{}, err
	}
	data, err := s.table.CertificationChainStates.Get(getKey(key))
	if err != nil {
		return node.State{}, err
	}
	state := node.State{}
	if err := state.Deserialize(data); err != nil {
		return node.State{}, err
	}
	return state, nil
}

// AddCertificationChainState adds a new certification chain state to the
// store. States are indexed by their block height. If there is a state for
// the same block hight already in the store, it is overwritten.
func (s *Store) AddCertificationChainState(state node.State) error {
	data, err := state.Serialize()
	if err != nil {
		return err
	}
	key := getKey(uint64(state.GetBlockHeight()))
	return s.table.CertificationChainStates.Put(key, data)
}

// EnumerateCertificationChainStates iterates over all certification chain states
// in the store. The states are yielded in ascending order of their block height.
// If an error occurs during iteration, it is yielded as a result and the iteration
// continues with the next state if there is one.
func (s *Store) EnumerateCertificationChainStates() iter.Seq[result.T[node.State]] {
	return func(yield func(result.T[node.State]) bool) {
		iter := s.table.CertificationChainStates.NewIterator(nil, nil)
		defer iter.Release()
		for iter.Next() {
			state := node.State{}
			if err := state.Deserialize(iter.Value()); err != nil {
				if !yield(result.Error[node.State](err)) {
					return
				}
			}
			if !yield(result.New(state)) {
				break
			}
		}
	}
}
