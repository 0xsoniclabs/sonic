package emitter

import (
	"time"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/sonic/emitter/ancestor"
	"github.com/0xsoniclabs/sonic/utils/piecefunc"

	"github.com/0xsoniclabs/sonic/opera"
)

func scalarUpdMetric(diff consensus.Seq, weight consensus.Weight, totalWeight consensus.Weight) ancestor.Metric {
	return ancestor.Metric(scalarUpdMetricF(uint64(diff)*piecefunc.DecimalUnit)) * ancestor.Metric(weight) / ancestor.Metric(totalWeight)
}

func updMetric(thresholdValue, cur, upd consensus.Seq, validatorIdx consensus.ValidatorIndex, validators *consensus.Validators) ancestor.Metric {
	if upd <= thresholdValue || upd <= cur {
		return 0
	}
	weight := validators.GetWeightByIdx(validatorIdx)
	if thresholdValue < cur {
		return scalarUpdMetric(upd-thresholdValue, weight, validators.TotalWeight()) - scalarUpdMetric(cur-thresholdValue, weight, validators.TotalWeight())
	}
	return scalarUpdMetric(upd-thresholdValue, weight, validators.TotalWeight())
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

func (em *Emitter) recheckIdleTime() {
	em.world.Lock()
	defer em.world.Unlock()
	if em.idle() {
		em.prevIdleTime = time.Now()
	}
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
