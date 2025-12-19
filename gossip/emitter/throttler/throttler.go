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

type WorldReader interface {
	GetRules() opera.Rules
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetLastEvent(idx.ValidatorID) *inter.Event
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
	lastDominantSet dominantSet, attempt config.Attempt) {

	validators, _ := world.GetEpochValidators()
	for _, id := range validators.IDs() {

		lastEvent := world.GetLastEvent(id)
		if lastEvent == nil {
			// No event has been seen from this validator yet
			continue
		}

		// zero value defaults to offline
		attendance := al.attendance[id]

		// Different tolerance for being online for dominant vs non-dominant validators.
		onlineThreshold := config.DominatingTimeout
		if _, wasDominant := lastDominantSet[id]; !wasDominant {
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
