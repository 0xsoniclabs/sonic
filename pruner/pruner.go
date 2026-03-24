package pruner

import (
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// EpochPruner is responsible for pruning old epochs from the database. It runs in
// a separate goroutine and periodically checks for epochs that are older than a certain
// threshold and removes them from the database. The threshold is defined by the chosen
// SafeEpochOracle.
type EpochPruner struct {
	oracle SafeEpochOracle
	store  *gossip.Store
}

// NewEpochPruner creates a new instance of EpochPruner with the given SafeEpochOracle
// and gossip.Store.
func NewEpochPruner(oracle SafeEpochOracle, store *gossip.Store) *EpochPruner {
	return &EpochPruner{
		oracle: oracle,
		store:  store,
	}
}

// Prune removes all epochs from the database that are older than the safe epoch threshold.
func (p *EpochPruner) Prune() error {
	endEpoch := p.oracle.GetSafeEpoch()
	for epoch := 0; epoch < endEpoch; epoch++ {
		p.store.ForEachEpochEvent(idx.Epoch(epoch), func(event *inter.EventPayload) bool {
			p.store.DelEvent(event.ID())
			return true
		})
		if err := p.store.Commit(); err != nil {
			return err
		}
	}
	return nil
}
