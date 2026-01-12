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

package throttler

import (
	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

//go:generate mockgen -source=throttler.go -destination=throttler_mock.go -package=throttler

// ThrottlingState holds the state required to decide if an event can be safely skipped,
// or if the validator must emit it to bring the stake online.
type ThrottlingState struct {
	// throttler configuration parameters
	thisValidatorID idx.ValidatorID
	config          config.ThrottlerConfig

	// means to access the world state
	world WorldReader

	// internal state
	attempt        config.Attempt
	lastEmission   config.Attempt
	attendanceList attendanceList
}

func NewThrottlingState(
	validatorID idx.ValidatorID,
	config config.ThrottlerConfig,
	stateReader WorldReader,
) ThrottlingState {

	return ThrottlingState{
		thisValidatorID: validatorID,
		world:           stateReader,
		config:          config,
		attendanceList:  newAttendanceList(),
	}
}

// SkipEventEmissionReason represents the reason for skipping or not skipping event emission.
// This type is used for the testing of the different reasons to avoid skipping emission.
type SkipEventEmissionReason int

const (
	SkipEventEmission SkipEventEmissionReason = iota
	DoNotSkipEvent_ThrottlerDisabled
	DoNotSkipEvent_CarriesTransactions
	DoNotSkipEvent_DominantStake
	DoNotSkipEvent_StakeNotDominated
	DoNotSkipEvent_Heartbeat
	DoNotSkipEvent_Genesis
)

// CanSkipEventEmission determines whether to skip the emission of the given event.
//
// It returns true if the event emission should be skipped, false otherwise.
func (ts *ThrottlingState) CanSkipEventEmission(event inter.EventPayloadI) SkipEventEmissionReason {
	if !ts.config.Enabled {
		return DoNotSkipEvent_ThrottlerDisabled
	}

	ts.attempt++

	// reset state on epoch start
	if event.SelfParent() == nil {
		ts.resetState()
	}

	// Update attendance list based on the observed events.
	oldDominatingSet := ts.computeOnlineDominantSet()
	ts.attendanceList.updateAttendance(ts.world, ts.config, oldDominatingSet, ts.attempt)

	// Determine whether this event can be skipped.
	newDominatingSet := ts.computeOnlineDominantSet()
	_, isDominant := newDominatingSet[ts.thisValidatorID]

	skip := ts.canSkip(event, isDominant)

	if skip != SkipEventEmission {
		ts.lastEmission = ts.attempt
	}

	return skip
}

// computeOnlineDominantSet computes the dominant accounting only for validators
// considered online.
func (ts *ThrottlingState) computeOnlineDominantSet() dominantSet {
	allValidators, _ := ts.world.GetEpochValidators()
	onlineValidators := ts.getOnlineValidators(allValidators)

	// Compute dominant set among online validators
	return computeDominantSet(
		onlineValidators,
		computeNeededStake(
			allValidators.TotalWeight(),
			ts.config.DominantStakeThreshold,
		),
	)
}

// canSkip determines if the event emission can be skipped based on the current throttling state.
// When it is safe to skip emission, the function returns SkipEventEmission.
// any other case, it return a reason to not skipping emission for this event.
func (ts *ThrottlingState) canSkip(
	event inter.EventPayloadI,
	isPartOfDominatingSet bool,
) SkipEventEmissionReason {

	if len(event.Transactions()) > 0 {
		return DoNotSkipEvent_CarriesTransactions
	}

	if event.SelfParent() == nil {
		return DoNotSkipEvent_Genesis
	}

	// Evaluate heartbeat condition,
	// - Has this validator not participated in blocks for too long?
	//   This prevents suppressed validators from being slashed for inactivity
	// - Has this validator not emitted for too long?
	//   This prevents other validators from flagging suppressed validators as offline.
	rules := ts.world.GetRules()
	NonDominatingTimeout := min(
		ts.config.NonDominatingTimeout/2,
		config.Attempt(rules.Economy.BlockMissedSlack)/2)
	if ts.lastEmission+NonDominatingTimeout <= ts.attempt {
		return DoNotSkipEvent_Heartbeat
	}

	if isPartOfDominatingSet {
		return DoNotSkipEvent_DominantStake
	}

	return SkipEventEmission
}

// getOnlineValidators returns a subset of validators in epoch which are currently considered online.
func (ts *ThrottlingState) getOnlineValidators(allValidators *pos.Validators) *pos.Validators {
	builder := pos.NewBuilder()
	for _, id := range allValidators.IDs() {
		if ts.thisValidatorID == id || ts.attendanceList.isOnline(id) {
			builder.Set(id, allValidators.Get(id))
		}
	}
	return builder.Build()
}

// resetState clears the internal state of the throttler, to be called on epoch start.
func (ts *ThrottlingState) resetState() {
	ts.attempt = 0
	ts.lastEmission = 0
	ts.attendanceList = newAttendanceList()
}

// validatorAttendance holds information about a validator's online status.
type validatorAttendance struct {
	lastSeenSeq idx.Event
	lastSeenAt  config.Attempt

	online bool
}

// attendanceList is a helper tool to track validators' online status.
type attendanceList struct {
	attendance map[idx.ValidatorID]validatorAttendance
}

func newAttendanceList() attendanceList {
	return attendanceList{
		attendance: make(map[idx.ValidatorID]validatorAttendance),
	}
}

// updateAttendance updates the attendance list based on the current world state and configuration.
func (al *attendanceList) updateAttendance(
	world WorldReader, config config.ThrottlerConfig,
	dominantSet dominantSet, attempt config.Attempt) {

	validators, _ := world.GetEpochValidators()
	for _, id := range validators.IDs() {

		lastEvent := world.GetLastEvent(id)
		if lastEvent == nil {
			continue
		}

		// zero value defaults to offline
		attendance := al.attendance[id]

		// Different tolerance for being online for dominant vs non-dominant validators.
		onlineThreshold := config.DominatingTimeout
		if _, isPartOfDominantSet := dominantSet[id]; !isPartOfDominantSet {
			onlineThreshold = config.NonDominatingTimeout
		}

		if attendance.lastSeenSeq >= lastEvent.Seq() {
			// if no progress has been made, re-evaluate online status
			// once a validator is marked offline, it stays offline until a new event is seen
			attendance.online = attendance.online && attendance.lastSeenAt+onlineThreshold > attempt
			al.attendance[id] = attendance
		} else {
			// if any progress has been made, mark as online
			al.attendance[id] = validatorAttendance{
				lastSeenSeq: lastEvent.Seq(),
				lastSeenAt:  attempt,
				online:      true,
			}
		}
	}
}

func (al *attendanceList) isOnline(id idx.ValidatorID) bool {
	attendance, exists := al.attendance[id]
	return exists && attendance.online
}

// WorldReader of the event throttler is an abstraction over the world state needed
// to make throttling decisions.
type WorldReader interface {
	GetRules() opera.Rules
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetLastEvent(idx.ValidatorID) *inter.Event
}
