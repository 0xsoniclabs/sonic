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

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
)

const (
	validatorChallenge = 4 * time.Second
)

func (em *Emitter) recountConfirmingIntervals(validators *pos.Validators) {
	// validators with lower stake should emit fewer events to reduce network load
	// confirmingEmitInterval = piecefunc(totalStakeBeforeMe / totalStake) * MinEmitInterval
	totalStakeBefore := pos.Weight(0)
	for i, stake := range validators.SortedWeights() {
		vid := validators.GetID(idx.Validator(i))
		// pos.Weight is uint32, so cast to uint64 to avoid an overflow
		stakeRatio := uint64(totalStakeBefore) * uint64(piecefunc.DecimalUnit) / uint64(validators.TotalWeight())
		if !em.offlineValidators[vid] {
			totalStakeBefore += stake
		}
		confirmingEmitIntervalRatio := confirmingEmitIntervalF(stakeRatio)
		em.stakeRatio[vid] = stakeRatio
		em.expectedEmitIntervals[vid] = time.Duration(
			piecefunc.Mul(em.globalConfirmingInterval.Load(), confirmingEmitIntervalRatio))
	}
	em.intervals.Confirming = em.expectedEmitIntervals[em.config.Validator.ID]
}

func (em *Emitter) recheckChallenges() {
	if time.Since(em.prevRecheckedChallenges) < validatorChallenge/10 {
		return
	}
	em.world.Lock()
	defer em.world.Unlock()
	now := time.Now()
	if !em.idle() {
		// give challenges to all the non-spare validators if network isn't idle
		for _, vid := range em.validators.Load().IDs() {
			if em.offlineValidators[vid] {
				continue
			}
			if _, ok := em.challenges[vid]; !ok {
				em.challenges[vid] = now.Add(validatorChallenge + em.expectedEmitIntervals[vid]*4)
			}
		}
	} else {
		// erase all the challenges if network is idle
		em.challenges = make(map[idx.ValidatorID]time.Time)
	}
	// check challenges
	recount := false
	for vid, challengeDeadline := range em.challenges {
		if now.After(challengeDeadline) {
			em.offlineValidators[vid] = true
			recount = true
		}
	}
	if recount {
		em.recountConfirmingIntervals(em.validators.Load())
	}
	em.prevRecheckedChallenges = now
}
