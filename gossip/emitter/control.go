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

	"github.com/Fantom-foundation/lachesis-base/emitter/ancestor"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"

	"github.com/0xsoniclabs/sonic/opera"
)

func scalarUpdMetric(diff idx.Event, weight pos.Weight, totalWeight pos.Weight) ancestor.Metric {
	return ancestor.Metric(scalarUpdMetricF(uint64(diff)*piecefunc.DecimalUnit)) * ancestor.Metric(weight) / ancestor.Metric(totalWeight)
}

func updMetric(median, cur, upd idx.Event, validatorIdx idx.Validator, validators *pos.Validators) ancestor.Metric {
	if upd <= median || upd <= cur {
		return 0
	}
	weight := validators.GetWeightByIdx(validatorIdx)
	if median < cur {
		return scalarUpdMetric(upd-median, weight, validators.TotalWeight()) - scalarUpdMetric(cur-median, weight, validators.TotalWeight())
	}
	return scalarUpdMetric(upd-median, weight, validators.TotalWeight())
}

func (em *Emitter) timeSinceLastEmit() time.Duration {
	var lastTime time.Time
	if last := em.prevEmittedAtTime.Load(); last != nil {
		lastTime = *last
	}
	return time.Since(lastTime)
}

func (em *Emitter) isAllowedToEmit() bool {
	passedTime := em.timeSinceLastEmit()
	if passedTime < 0 {
		passedTime = 0
	}

	// If a emitter interval is defined, all other heuristics are ignored.
	interval := em.getEmitterIntervalLimit()
	return passedTime >= interval
}

func (em *Emitter) getEmitterIntervalLimit() time.Duration {
	rules := em.world.GetRules().Emitter

	var lastConfirmationTime time.Time
	if last := em.lastTimeAnEventWasConfirmed.Load(); last != nil {
		lastConfirmationTime = *last
	} else {
		// If we have not seen any event confirmed so far, we take the current time
		// as the last confirmation time. Thus, during start-up we would not unnecessarily
		// slow down the event emission for the very first event. The switch into the stall
		// mode is delayed by the stall-threshold.
		now := time.Now()
		em.lastTimeAnEventWasConfirmed.Store(&now)
		lastConfirmationTime = now
	}

	return getEmitterIntervalLimit(rules, time.Since(lastConfirmationTime))
}

func getEmitterIntervalLimit(
	rules opera.EmitterRules,
	delayOfLastConfirmedEvent time.Duration,
) time.Duration {
	// Check for a network-stall situation in which events emitting should be slowed down.
	stallThreshold := time.Duration(rules.StallThreshold)
	if delayOfLastConfirmedEvent > stallThreshold {
		return time.Duration(rules.StalledInterval)
	}

	// Use the regular emitter interval.
	return time.Duration(rules.Interval)
}
