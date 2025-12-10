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
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

//go:generate mockgen -source=throttling.go -destination=throttling_mock.go -package=throttling

// ThrottlingState manages the state for event emission throttling based on
// the set of dominant validators.
// This type contains the state needed to decide whether to skip event emission
// for a given validator, based on its stake and the stake distribution among
// all validators, and properties of the events to be emitted.
type ThrottlingState struct {
	// throttler configuration parameters
	thisValidatorID                     idx.ValidatorID
	dominatorsThreshold                 float64
	maxSkippedEventsWithSameFrameNumber uint
	heartbeatFramesCount                uint

	// means to access the world state
	world WorldReader

	// accumulated state
	currentSkippedEventsCount uint
	lastTestedFrame           idx.Frame
	lastEmissionBlockNumber   idx.Block
}

type WorldReader interface {
	GetRules() opera.Rules
	GetLatestBlockIndex() idx.Block
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetLastEvent(epoch idx.Epoch, from idx.ValidatorID) *hash.Event
	GetEvent(hash.Event) *inter.Event
}

// NewThrottlingState creates a new ThrottlingState for a given validator ID
// and dominating percentile threshold.
//
// dominatingPercentile parameter specifies the fraction of total stake
// that defines the dominant set of validators. For example, a value of 0.75
// means that the dominant set is the smallest set of validators whose combined
// stake is at least 75% of the total stake.
//
// maxSkippedEventsWithSameFrameNumber parameter specifies the maximum of consecutive
// skipped events with the same frame number before forcing an emission. This brings
// the stake of this node online when no progress can be observed in the network.
func NewThrottlingState(
	validatorID idx.ValidatorID,
	dominatingPercentile float64,
	maxSkippedEventsWithSameFrameNumber uint,
	heartbeatFramesCount uint,
	stateReader WorldReader,
) *ThrottlingState {
	return &ThrottlingState{
		thisValidatorID: validatorID,
		// Clamp the threshold between 0.7 and 1 to avoid extreme values.
		// 0.7 is a conservative approximation of the Byzantine fault tolerance limit (2/3+1).
		dominatorsThreshold:                 min(max(dominatingPercentile, 0.7), 1),
		maxSkippedEventsWithSameFrameNumber: maxSkippedEventsWithSameFrameNumber,
		heartbeatFramesCount:                heartbeatFramesCount,
		world:                               stateReader,
	}
}

const (
	SkipEventEmission = iota
	DoNotSkipEvent_CarriesTransactions
	DoNotSkipEvent_SameFrameExceeded
	DoNotSkipEvent_TooManyBlocksMissed
	DoNotSkipEvent_DominantStake
	DoNotSkipEvent_Heartbeat
)

// CanSkipEventEmission determines whether to skip the emission of the given event.
//
// It returns true if the event emission should be skipped, false otherwise.
func (ts *ThrottlingState) CanSkipEventEmission(event inter.EventPayloadI) int {
	skip := ts.canSkipEvent(event)
	if ts.lastTestedFrame < event.Frame() {
		ts.currentSkippedEventsCount = 0
		ts.lastTestedFrame = event.Frame()
	}

	if skip == SkipEventEmission {
		ts.currentSkippedEventsCount++
	} else {
		ts.lastEmissionBlockNumber = ts.world.GetLatestBlockIndex()
		ts.currentSkippedEventsCount = 0
	}
	return skip
}

func (ts *ThrottlingState) canSkipEvent(event inter.EventPayloadI) int {
	// Do not skip emission if the event carries transactions
	if len(event.Transactions()) > 0 {
		return DoNotSkipEvent_CarriesTransactions
	}

	rules := ts.world.GetRules()
	currentEpochValidators, epoch := ts.world.GetEpochValidators()

	// The system requires to keep a heartbeat emission every N frames, this
	// ensures that suppressed validators are seen as online by other validators.
	lastEmittedEvent := ts.getLastEmittedEvent(epoch, ts.thisValidatorID)
	if lastEmittedEvent != nil &&
		event.Frame()-lastEmittedEvent.Frame() >= idx.Frame(ts.heartbeatFramesCount) {
		return DoNotSkipEvent_Heartbeat
	}

	// Do not skip emission if the event is in the same frame as the last emitted event
	// for a given period of time.
	// This means that no progress in the network can be observed. The stake of this
	// validator stake may be needed to reach quorum in the current frame.
	if ts.lastTestedFrame == event.Frame() &&
		ts.currentSkippedEventsCount >= ts.maxSkippedEventsWithSameFrameNumber {
		return DoNotSkipEvent_SameFrameExceeded
	}

	// Do not skip emission if too many blocks have been missed since the last emission.
	// This prevents this node from being flagged as inactive, and its stake being slashed.
	//
	// Although it could be argued that a block is produced by a frame when there is
	// load in the system, and therefore this measure could be computed based on frames
	// instead of blocks, using blocks is preserved as a redundant safety measure.
	blockMissedSlack := rules.Economy.BlockMissedSlack
	currentBlockNumber := ts.world.GetLatestBlockIndex()
	if currentBlockNumber > ts.lastEmissionBlockNumber &&
		currentBlockNumber-ts.lastEmissionBlockNumber > blockMissedSlack/2 {
		return DoNotSkipEvent_TooManyBlocksMissed
	}

	// Evaluate whether this validator is in the dominant set of validators.
	// The dominant set is computed considering only online stake, i.e. stake
	// of validators that have emitted an event recently enough.
	onlineValidators := ts.filterOfflineValidators(currentEpochValidators, event, epoch)
	dominantSet := ComputeDominantSet(onlineValidators, currentEpochValidators.TotalWeight(), ts.dominatorsThreshold)
	if _, found := dominantSet[ts.thisValidatorID]; found {
		return DoNotSkipEvent_DominantStake
	}

	return SkipEventEmission
}

// getLastEmittedEvent retrieves the last event emitted by the given validator
// in the specified epoch.
// If no event is found, it returns nil.
func (ts *ThrottlingState) getLastEmittedEvent(epoch idx.Epoch, validatorId idx.ValidatorID) *inter.Event {
	eventId := ts.world.GetLastEvent(epoch, validatorId)
	if eventId == nil {
		return nil
	}

	lastEventSeen := ts.world.GetEvent(*eventId)
	if lastEventSeen == nil {
		return nil
	}
	return lastEventSeen
}

// filterOfflineValidators returns a new Validators object containing only
// the validators from currentEpochValidators that have emitted an event
// recently enough to be considered online.
func (ts *ThrottlingState) filterOfflineValidators(
	currentEpochValidators *pos.Validators,
	event inter.EventPayloadI,
	epoch idx.Epoch,
) *pos.Validators {

	if event.SelfParent() == nil {
		return currentEpochValidators
	}

	builder := pos.NewBuilder()
	for _, validatorId := range currentEpochValidators.IDs() {
		lastEventSeen := ts.getLastEmittedEvent(epoch, validatorId)
		if lastEventSeen == nil {
			continue
		}

		if event.Frame()-lastEventSeen.Frame() <= idx.Frame(ts.heartbeatFramesCount)*2 {
			builder.Set(validatorId, currentEpochValidators.Get(validatorId))
		}
	}
	return builder.Build()
}
