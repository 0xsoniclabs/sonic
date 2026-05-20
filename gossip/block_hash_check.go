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

package gossip

import (
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/utils/errlock"
)

// blockHashChecker tracks block hash disagreements between the local node
// and validators' events. When validators representing more than 2/3 of the
// total stake report block hashes that differ from the local node's hashes,
// the node permanently stops to prevent operating on a fork.
type blockHashChecker struct {
	store     *Store
	errorLock *errlock.ErrorLock

	epoch      idx.Epoch
	validators *pos.Validators

	// Per-block: set of validators that disagree with the local block hash.
	disagreements map[idx.Block]map[idx.ValidatorID]struct{}
}

func newBlockHashChecker(store *Store, errorLock *errlock.ErrorLock) *blockHashChecker {
	return &blockHashChecker{
		store:         store,
		errorLock:     errorLock,
		disagreements: make(map[idx.Block]map[idx.ValidatorID]struct{}),
	}
}

// reset clears all tracked disagreements and sets the new epoch and validators.
// Called on epoch transitions.
func (c *blockHashChecker) reset(epoch idx.Epoch, validators *pos.Validators) {
	c.epoch = epoch
	c.validators = validators
	c.disagreements = make(map[idx.Block]map[idx.ValidatorID]struct{})
}

// check inspects the block hashes in an incoming event and compares them
// against the local block hashes. If disagreeing stake exceeds 2/3 of total
// weight for any block, the node is permanently stopped.
func (c *blockHashChecker) check(e *inter.EventPayload) {
	if c.errorLock == nil {
		return
	}
	bhs := e.BlockHashes()
	if bhs.Start == 0 || len(bhs.Hashes) == 0 {
		return
	}
	if c.validators == nil {
		return
	}
	// Only track disagreements for the current epoch. Events with
	// block hashes from other epochs are irrelevant to the current
	// epoch's disagreement tracking and would grow the map unboundedly.
	if bhs.Epoch != c.epoch {
		return
	}

	creator := e.Creator()
	for i, eventHash := range bhs.Hashes {
		blockNum := bhs.Start + idx.Block(i)
		localBlock := c.store.GetBlock(blockNum)
		if localBlock == nil {
			continue
		}
		// Defense-in-depth: skip blocks that don't belong to the
		// current epoch in the local store.
		if localBlock.Epoch != c.epoch {
			continue
		}
		localHash := hash.Hash(localBlock.Hash())
		if localHash == eventHash {
			continue
		}
		// Record disagreement
		if c.disagreements[blockNum] == nil {
			c.disagreements[blockNum] = make(map[idx.ValidatorID]struct{})
		}
		c.disagreements[blockNum][creator] = struct{}{}
		// Compute total disagreeing stake
		disagreeingStake := pos.Weight(0)
		for vid := range c.disagreements[blockNum] {
			disagreeingStake += c.validators.Get(vid)
		}
		if disagreeingStake > (c.validators.TotalWeight()*2)/3 {
			c.errorLock.Permanent(fmt.Errorf(
				"block hash disagreement: validators with >2/3 stake disagree "+
					"on block %d (local hash %s, epoch %d)",
				blockNum, localHash.String(), c.epoch,
			))
		}
	}
}
