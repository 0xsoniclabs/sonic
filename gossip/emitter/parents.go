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

package emitter

import (
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/emitter/ancestor"
)

// buildSearchStrategies returns a strategy for each parent search
func (em *Emitter) buildSearchStrategies(maxParents consensus.Seq) []ancestor.SearchStrategy {
	strategies := make([]ancestor.SearchStrategy, 0, maxParents)
	if maxParents == 0 {
		return strategies
	}
	payloadStrategy := em.payloadIndexer.SearchStrategy()
	for consensus.Seq(len(strategies)) < 1 {
		strategies = append(strategies, payloadStrategy)
	}
	randStrategy := ancestor.NewRandomStrategy(nil)
	for consensus.Seq(len(strategies)) < maxParents/2 {
		strategies = append(strategies, randStrategy)
	}
	if em.fcIndexer != nil {
		quorumStrategy := em.fcIndexer.SearchStrategy()
		for consensus.Seq(len(strategies)) < maxParents {
			strategies = append(strategies, quorumStrategy)
		}
	} else if em.quorumIndexer != nil {
		quorumStrategy := em.quorumIndexer.SearchStrategy()
		for consensus.Seq(len(strategies)) < maxParents {
			strategies = append(strategies, quorumStrategy)
		}
	}
	return strategies
}

// chooseParents selects an "optimal" parents set for the validator
func (em *Emitter) chooseParents(epoch consensus.Epoch, myValidatorID consensus.ValidatorID) (*consensus.EventHash, consensus.EventHashes, bool) {
	selfParent := em.world.GetLastEvent(epoch, myValidatorID)
	if selfParent == nil {
		return nil, nil, true
	}
	if len(em.world.DagIndex().NoCheaters(selfParent, consensus.EventHashes{*selfParent})) == 0 {
		em.Error(time.Second, "Events emitting isn't allowed due to the doublesign", "validator", myValidatorID)
		return nil, nil, false
	}
	parents := consensus.EventHashes{*selfParent}
	heads := em.world.GetHeads(epoch) // events with no descendants
	parents = ancestor.ChooseParents(parents, heads, em.buildSearchStrategies(em.maxParents-consensus.Seq(len(parents))))
	return selfParent, parents, true
}
