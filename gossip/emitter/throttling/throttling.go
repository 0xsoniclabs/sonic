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
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

//go:generate mockgen -source=throttling.go -destination=throttling_mock.go -package=throttling

// ThrottlingState manages the state for event emission throttling based on
// dominant set of validators.
// This type contains the state needed to decide whether to skip event emission
// for a given validator, based on its stake and the stake distribution among
// all validators, and properties of the events to be emitted.
type ThrottlingState struct {
	thisValidatorID      idx.ValidatorID
	dominatingPercentile float64

	maxSkippedEventsWithSameFrameNumber uint

	world WorldReader

	lastEventFrame          idx.Frame
	lastEventTime           inter.Timestamp
	lastEmissionBlockNumber idx.Block
}

type WorldReader interface {
	GetRules() opera.Rules
	GetLatestBlockIndex() idx.Block
	GetEpochValidators() (*pos.Validators, idx.Epoch)
}

// NewThrottlingState creates a new ThrottlingState for a given validator ID
// and dominating percentile threshold.
//
// dominatingPercentile parameter specifies the fraction of total stake
// that defines the dominant set of validators. For example, a value of 0.75
// means that the dominant set is the smallest set of validators whose combined
// stake is at least 75% of the total stake.
//
// stalledFrameTimeout parameter specifies the duration to consider an event
// as stalled in the current frame.
func NewThrottlingState(
	validatorID idx.ValidatorID,
	dominatingPercentile float64,
	maxSkippedEventsWithSameFrameNumber uint,
	stateReader WorldReader,
) *ThrottlingState {
	return &ThrottlingState{
		thisValidatorID:                     validatorID,
		dominatingPercentile:                dominatingPercentile,
		maxSkippedEventsWithSameFrameNumber: maxSkippedEventsWithSameFrameNumber,
		world:                               stateReader,
	}
}

// SkipEventEmission determines whether to skip the emission of the given event.
//
// It returns true if the event emission should be skipped, false otherwise.
// If the event is not skipped, the internal state is updated to reflect the
// last emitted event's frame and creation time.
func (ts *ThrottlingState) SkipEventEmission(event inter.EventPayloadI) bool {
	skip := ts.skipEvent(event)
	if !skip {
		ts.lastEventFrame = event.Frame()
		ts.lastEventTime = event.CreationTime()
		ts.lastEmissionBlockNumber = ts.world.GetLatestBlockIndex()
	}
	return skip
}

func (ts *ThrottlingState) skipEvent(event inter.EventPayloadI) bool {
	// Do not skip emission if the event carries transactions
	if len(event.Transactions()) > 0 {
		return false
	}

	rules := ts.world.GetRules()

	// Do not skip emission if the event is in the same frame as the last emitted event
	// for a given period of time.
	// This means that there no progress in the network can be observed. The stake of this
	// validator stake may be needed to reach quorum in the current frame.
	maxSameFrameDuration := time.Duration(ts.maxSkippedEventsWithSameFrameNumber) * time.Duration(rules.Emitter.Interval)
	if ts.lastEventFrame == event.Frame() &&
		time.Since(ts.lastEventTime.Time()) >= maxSameFrameDuration {
		return false
	}

	// TODO: consider BlockMissedSlack
	blockMissedSlack := rules.Economy.BlockMissedSlack
	currentBlockNumber := ts.world.GetLatestBlockIndex()
	if currentBlockNumber-ts.lastEmissionBlockNumber > blockMissedSlack/2 {
		return false
	}

	// Compute dominant set and check if this validator belongs to it.
	validators, _ := ts.world.GetEpochValidators()
	totalStake := validators.TotalWeight()
	threshold := pos.Weight(float64(totalStake) * ts.dominatingPercentile)
	dominantSet, dominated := ComputeDominantSet(validators, threshold)
	if !dominated {
		return false
	}

	_, isInDominantSet := dominantSet[ts.thisValidatorID]
	return !isInDominantSet
}
