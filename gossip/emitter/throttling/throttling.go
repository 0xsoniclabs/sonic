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

package throttling

import (
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// ThrottlingState manages the state for event emission throttling based on
// dominant set of validators.
// This type contains the state needed to decide whether to skip event emission
// for a given validator, based on its stake and the stake distribution among
// all validators, and properties of the events to be emitted.
type ThrottlingState struct {
	thisValidatorID      idx.ValidatorID
	dominatingPercentile float64
	validators           *pos.Validators

	lastEventFrame  idx.Frame
	lastEventTime   inter.Timestamp
	totalStake      pos.Weight
	stalledInterval inter.Timestamp
}

// NewThrottlingState creates a new ThrottlingState for a given validator ID
// and dominating percentile threshold.
//
// The dominatingPercentile parameter specifies the fraction of total stake
// that defines the dominant set of validators. For example, a value of 0.75
// means that the dominant set is the smallest set of validators whose combined
// stake is at least 75% of the total stake.
func NewThrottlingState(validatorID idx.ValidatorID, dominatingPercentile float64) *ThrottlingState {
	return &ThrottlingState{
		thisValidatorID:      validatorID,
		dominatingPercentile: dominatingPercentile,
	}
}

// OnNewEpoch updates the throttling state for a new epoch with the given
// validators and opera rules.
func (ts *ThrottlingState) OnNewEpoch(validators *pos.Validators, rules opera.Rules) {
	ts.stalledInterval = rules.Emitter.StalledInterval / 2
	ts.validators = validators
	ts.totalStake = validators.TotalWeight()
}

// SkipEventEmission determines whether to skip the emission of the given event.
//
// It returns true if the event emission should be skipped, false otherwise.
//
// If the event is not skipped, the internal state is updated to reflect the
// last emitted event's frame and creation time.
func (ts *ThrottlingState) SkipEventEmission(event inter.EventPayloadI) bool {
	skip := ts.skipEvent(event)
	if !skip {
		ts.lastEventFrame = event.Frame()
		ts.lastEventTime = event.CreationTime()
	}
	return skip
}

func (ts *ThrottlingState) skipEvent(event inter.EventPayloadI) bool {
	// Do not skip emission if the event carries transactions
	if len(event.Transactions()) > 0 {
		return false
	}

	// Do not skip emission if the event is in the same frame as the last emitted event.
	// this meanst that there is no progress in the network, and the stake of this
	// validator might be needed to confirm transactions.
	if event.Frame() == ts.lastEventFrame {
		return false
	}

	// Do not skip if emission is stalled for too long.
	// This prevents emitters from being flagged as slacking during epoch
	// ceiling.
	if (event.CreationTime() - ts.lastEventTime) <= ts.stalledInterval {
		return false
	}

	// Compute dominant set and check if this validator belongs to it.
	thresshold := pos.Weight(float64(ts.totalStake) * ts.dominatingPercentile)
	dominantSet, dominated := ComputeDominantSet(ts.validators, thresshold)
	if !dominated {
		return false
	}

	_, isInDominantSet := dominantSet[ts.thisValidatorID]
	return !isInDominantSet
}
