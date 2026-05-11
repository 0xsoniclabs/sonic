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

package emitter

import (
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (em *Emitter) ForceEventEmissionForTesting(
	txs []*types.Transaction,
) error {
	if !em.config.AllowForcedEmission {
		return fmt.Errorf("forced event emission is not allowed by configuration")
	}

	if !em.isValidator() {
		return fmt.Errorf("cannot emit events, not a validator")
	}

	em.world.Lock()
	defer em.world.Unlock()

	e, err := em.createForcedEventForTesting(txs)
	if e == nil || err != nil {
		return err
	}

	em.syncStatus.prevLocalEmittedID = e.ID()

	err = em.world.Process(e)
	if err != nil {
		em.Log.Error("Self-event connection failed", "err", err.Error())
		return err
	}
	// write event ID to avoid double-signing in future after a crash
	em.writeLastEmittedEventID(e.ID())
	// broadcast the event
	em.world.Broadcast(e)

	return nil
}

func (em *Emitter) createForcedEventForTesting(
	txs []*types.Transaction,
) (*inter.EventPayload, error) {
	var (
		selfParentSeq  idx.Event
		selfParentTime inter.Timestamp
		parents        hash.Events
		maxLamport     idx.Lamport
	)

	// Find parents
	selfParent, parents, ok := em.chooseParents(idx.Epoch(em.epoch.Load()), em.config.Validator.ID)
	if !ok {
		return nil, fmt.Errorf("failed to pick parents for the forced event")
	}
	prevEmitted := em.readLastEmittedEventID()
	if prevEmitted != nil && prevEmitted.Epoch() >= idx.Epoch(em.epoch.Load()) {
		if selfParent == nil || *selfParent != *prevEmitted {
			return nil, fmt.Errorf("internal error, self parent doesn't match the last emitted event")
		}
	}

	// Set parent-dependent fields
	parentHeaders := make(inter.Events, len(parents))
	for i, p := range parents {
		parent := em.world.GetEvent(p)
		if parent == nil {
			em.Log.Crit("Emitter: head not found", "mutEvent", p.String())
		}
		parentHeaders[i] = parent
		if parentHeaders[i].Creator() == em.config.Validator.ID && i != 0 {
			// there are 2 heads from me, i.e. due to a fork, chooseParents could have found multiple self-parents
			em.Error(5*time.Second, "I've created a fork, events emitting isn't allowed", "creator", em.config.Validator.ID)
			return nil, fmt.Errorf("fork detected, events emitting isn't allowed")
		}
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
	}

	selfParentSeq = 0
	selfParentTime = 0
	var selfParentHeader *inter.Event
	if selfParent != nil {
		selfParentHeader = parentHeaders[0]
		selfParentSeq = selfParentHeader.Seq()
		selfParentTime = selfParentHeader.CreationTime()
	}

	version := uint8(0)
	if em.world.GetRules().Upgrades.SingleProposerBlockFormation {
		version = 3
	} else if em.world.GetRules().Upgrades.Sonic {
		version = 2
	} else if em.world.GetRules().Upgrades.Llr {
		version = 1
	}

	mutEvent := &inter.MutableEventPayload{}
	mutEvent.SetVersion(version)
	mutEvent.SetEpoch(idx.Epoch(em.epoch.Load()))
	mutEvent.SetSeq(selfParentSeq + 1)
	mutEvent.SetCreator(em.config.Validator.ID)

	mutEvent.SetParents(parents)
	mutEvent.SetLamport(maxLamport + 1)
	mutEvent.SetCreationTime(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParentTime+1))

	// fill optional extra data field
	em.fillExtraData(mutEvent)

	// set consensus fields
	err := em.world.Build(mutEvent, nil)
	if err != nil {
		if err == ErrNotEnoughGasPower {
			validators := em.validators.Load()
			em.Warn(time.Second, "Not enough gas power to emit event. Too small stake?",
				"stake%", 100*float64(validators.Get(em.config.Validator.ID))/float64(validators.TotalWeight()))
		} else {
			em.Log.Warn("Dropped event while emitting", "err", err)
		}
		return nil, fmt.Errorf("failed to build event: %w", err)
	}

	if version == 3 {
		// In single proposer mode, we would have to wait until it's the local
		// validator's turn to emit a proposal. This is a bit more complicated,
		// and can be implemented if needed. For now, it is skipped.
		return nil, fmt.Errorf("forced transaction emissions not yet supported in single proposer mode")
	} else {
		// Add txs
		mutEvent.SetTxs(txs)

		// Fix gas power usage.
		for _, tx := range txs {
			mutEvent.SetGasPowerUsed(mutEvent.GasPowerUsed() + tx.Gas())
			mutEvent.SetGasPowerLeft(mutEvent.GasPowerLeft().Sub(tx.Gas()))
		}

		// calc Payload hash
		mutEvent.SetPayloadHash(inter.CalcPayloadHash(mutEvent))
	}

	// sign
	bSig, err := em.world.EventsSigner.Sign(common.Hash(mutEvent.HashToSign()))
	if err != nil {
		em.Error(time.Second, "Failed to sign event", "err", err)
		return nil, err
	}
	var sig inter.Signature
	copy(sig[:], bSig)
	mutEvent.SetSig(sig)

	// build clean event
	event := mutEvent.Build()

	// set mutEvent name for debug
	em.nameEventForDebug(event)

	return event, nil
}
