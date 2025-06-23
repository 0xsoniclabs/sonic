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

func (em *Emitter) isAllowedToEmit() bool {
	passedTime := time.Since(em.prevEmittedAtTime)
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
	if last := em.lastTimeAnEventWasConfirmed; !last.IsZero() {
		lastConfirmationTime = last
	} else {
		// If we have not seen any event confirmed so far, we take the current time
		// as the last confirmation time. Thus, during start-up we would not unnecessarily
		// slow down the event emission for the very first event. The switch into the stall
		// mode is delayed by the stall-threshold.
		now := time.Now()
		em.lastTimeAnEventWasConfirmed = now
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
