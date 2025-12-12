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

//go:generate mockgen -source=throttling.go -destination=throttling_mock.go -package=throttling

type attempt uint64

type validatorAttendance struct {
	lastSeenSeq idx.Event
	lastSeenAt  attempt

	online bool
}
type attendanceList map[idx.ValidatorID]validatorAttendance

type ThrottlingState struct {
	// throttler configuration parameters
	thisValidatorID     idx.ValidatorID
	dominatorsThreshold float64
	shortTimeout        attempt
	longTimeout         attempt

	// means to access the world state
	world WorldReader

	// internal state
	attempt             attempt
	lastEmissionAttempt attempt
	attendanceList      attendanceList
	lastDominatingSet   dominantSet
}

type WorldReader interface {
	GetRules() opera.Rules
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetLastEvent(idx.ValidatorID) *inter.Event
}

func NewThrottlingState(
	validatorID idx.ValidatorID,
	dominatingPercentile float64,
	shortTimeout uint64,
	longTimeout uint64,
	stateReader WorldReader,
) *ThrottlingState {
	return &ThrottlingState{
		thisValidatorID: validatorID,
		// Clamp the threshold between 0.7 and 1 to avoid extreme values.
		// 0.7 is a conservative approximation of the Byzantine fault tolerance limit (2/3+1).
		dominatorsThreshold: min(max(dominatingPercentile, 0.7), 1),
		shortTimeout:        attempt(shortTimeout),
		// longTimeout dictates heartbeat of suppressed validators, these nodes
		// will emit heartbeat twice per longTimeout attempts.
		// note: longTimeout smaller than 4 will trigger heartbeats every attempt.
		longTimeout: attempt(longTimeout),
		world:       stateReader,

		attendanceList: make(attendanceList),
	}
}

type SkipEventEmissionReason int

const (
	SkipEventEmission SkipEventEmissionReason = iota
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
	ts.attempt++

	// reset state on epoch start
	if event.SelfParent() == nil {
		ts.resetState()
	}

	ts.updateAttendance()

	skip := ts.canSkip(event)

	if skip != SkipEventEmission {
		ts.lastEmissionAttempt = ts.attempt
	}

	return skip
}

func (ts *ThrottlingState) canSkip(event inter.EventPayloadI) SkipEventEmissionReason {

	if len(event.Transactions()) > 0 {
		return DoNotSkipEvent_CarriesTransactions
	}

	if event.SelfParent() == nil {
		return DoNotSkipEvent_Genesis
	}

	rules := ts.world.GetRules()
	heartbeatTimeout := min(
		ts.longTimeout/2,
		attempt(rules.Economy.BlockMissedSlack/2))
	if ts.lastEmissionAttempt+heartbeatTimeout <= ts.attempt {
		return DoNotSkipEvent_Heartbeat
	}

	// Filter offline validators based on their attendance
	allValidators, _ := ts.world.GetEpochValidators()
	onlineValidators := ts.computeOnlineValidators(allValidators)

	// Compute dominant set among online validators
	ts.lastDominatingSet = ComputeDominantSet(
		onlineValidators,
		allValidators.TotalWeight(),
		ts.dominatorsThreshold,
	)

	if len(ts.lastDominatingSet) == 0 {
		return DoNotSkipEvent_StakeNotDominated
	}
	if _, isDominant := ts.lastDominatingSet[ts.thisValidatorID]; isDominant {
		return DoNotSkipEvent_DominantStake
	}

	return SkipEventEmission
}

func (ts *ThrottlingState) updateAttendance() {
	validators, _ := ts.world.GetEpochValidators()
	for _, id := range validators.IDs() {

		lastEvent := ts.world.GetLastEvent(id)
		if lastEvent == nil {
			continue
		}

		attendance, exists := ts.attendanceList[id]

		// different tolerance for being online for dominant vs non-dominant validators
		onlineThreshold := ts.shortTimeout
		if exists && attendance.online {
			if _, wasDominant := ts.lastDominatingSet[id]; !wasDominant {
				onlineThreshold = ts.longTimeout
			}
		}

		if attendance.lastSeenSeq == lastEvent.Seq() {
			attendance.online = attendance.lastSeenAt+onlineThreshold > ts.attempt
			ts.attendanceList[id] = attendance
		} else {
			ts.attendanceList[id] = validatorAttendance{
				lastSeenSeq: lastEvent.Seq(),
				lastSeenAt:  ts.attempt,
				online:      true,
			}
		}
	}
}

func (ts *ThrottlingState) computeOnlineValidators(allValidators *pos.Validators) *pos.Validators {
	builder := pos.NewBuilder()
	for id, attendance := range ts.attendanceList {
		if attendance.online {
			builder.Set(id, allValidators.Get(id))
		}
	}
	return builder.Build()
}

func (ts *ThrottlingState) resetState() {
	ts.attempt = 0
	ts.lastEmissionAttempt = 0
	ts.attendanceList = make(map[idx.ValidatorID]validatorAttendance)
}
