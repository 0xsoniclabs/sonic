// Copyright 2026 Sonic Operations Ltd
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

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/txtime"
)

var (
	effectiveBundleGasHistogram = utils.MetricsHistogram(utils.NewPrometheusHistogram(prometheus.HistogramOpts{
		Name:    "emitter_bundle_gas_effective",
		Help:    "Effective gas usage ratio for bundle transactions",
		Buckets: prometheus.LinearBuckets(0.0, 0.01, 100), // buckets: [0.0, 0.01, ..., 0.99, +inf]
	}))
)

const (
	txTurnPeriod        = 8 * time.Second
	txTurnPeriodLatency = 1 * time.Second
	txTurnNonces        = 32
)

func max64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func (em *Emitter) maxGasPowerToUse(e *inter.MutableEventPayload) uint64 {
	rules := em.world.GetRules()
	maxGasToUse := rules.Economy.Gas.MaxEventGas
	if maxGasToUse > e.GasPowerLeft().Min() {
		maxGasToUse = e.GasPowerLeft().Min()
	}
	// Smooth TPS if power isn't big
	if em.config.LimitedTpsThreshold > em.config.NoTxsThreshold {
		upperThreshold := em.config.LimitedTpsThreshold
		downThreshold := em.config.NoTxsThreshold

		estimatedAlloc := gaspowercheck.CalcValidatorGasPower(e, e.CreationTime(), e.MedianTime(), 0, em.validators.Load(), gaspowercheck.Config{
			Idx:                inter.LongTermGas,
			AllocPerSec:        rules.Economy.LongGasPower.AllocPerSec * 4 / 5,
			MaxAllocPeriod:     inter.Timestamp(time.Minute),
			MinEnsuredAlloc:    0,
			StartupAllocPeriod: 0,
			MinStartupGas:      0,
		})

		gasPowerLeft := e.GasPowerLeft().Min() + estimatedAlloc
		if gasPowerLeft < downThreshold {
			return 0
		}
		newGasPowerLeft := uint64(0)
		if gasPowerLeft > maxGasToUse {
			newGasPowerLeft = gasPowerLeft - maxGasToUse
		}

		var x1, x2 = newGasPowerLeft, gasPowerLeft
		if x1 < downThreshold {
			x1 = downThreshold
		}
		if x2 > upperThreshold {
			x2 = upperThreshold
		}
		trespassingPart := uint64(0)
		if x2 > x1 {
			trespassingPart = x2 - x1
		}
		healthyPart := uint64(0)
		if gasPowerLeft > x2 {
			healthyPart = gasPowerLeft - x2
		}

		smoothGasToUse := healthyPart + trespassingPart/2
		if maxGasToUse > smoothGasToUse {
			maxGasToUse = smoothGasToUse
		}
	}
	// pendingGas should be below MaxBlockGas
	{
		maxPendingGas := max64(max64(rules.Blocks.MaxBlockGas/3, rules.Economy.Gas.MaxEventGas), 15000000)
		if maxPendingGas <= em.pendingGas {
			return 0
		}
		if maxPendingGas < em.pendingGas+maxGasToUse {
			maxGasToUse = maxPendingGas - em.pendingGas
		}
	}
	// No txs if power is low
	{
		threshold := em.config.NoTxsThreshold
		if e.GasPowerLeft().Min() <= threshold {
			return 0
		} else if e.GasPowerLeft().Min() < threshold+maxGasToUse {
			maxGasToUse = e.GasPowerLeft().Min() - threshold
		}
	}
	return maxGasToUse
}

func getTxRoundIndex(now, txTime time.Time, validatorsNum idx.Validator) int {
	passed := now.Sub(txTime)
	if passed < 0 {
		passed = 0
	}
	return int((passed / txTurnPeriod) % time.Duration(validatorsNum))
}

// safe for concurrent use
func (em *Emitter) isMyTxTurn(txHash common.Hash, sender common.Address, accountNonce uint64, now time.Time, validators *pos.Validators, me idx.ValidatorID, epoch idx.Epoch) bool {
	txTime := txtime.Of(txHash)

	roundIndex := getTxRoundIndex(now, txTime, validators.Len())
	if roundIndex != getTxRoundIndex(now.Add(txTurnPeriodLatency), txTime, validators.Len()) {
		// round is about to change, avoid originating the transaction to avoid racing with another validator
		return false
	}

	// generate seed for generating the validators sequence for the tx
	roundsHash := hash.Of(sender.Bytes(), bigendian.Uint64ToBytes(accountNonce/txTurnNonces), epoch.Bytes())

	// generate the validators sequence for the tx
	rounds := utils.WeightedPermutation(int(validators.Len()), validators.SortedWeights(), roundsHash)

	// take a validator from the sequence, skip offline validators
	for ; roundIndex < len(rounds); roundIndex++ {
		chosenValidator := validators.GetID(idx.Validator(rounds[roundIndex]))
		if chosenValidator == me {
			return true // current validator is the chosen - emit
		}
		if !em.offlineValidators[chosenValidator] {
			return false // chosen validator is online - don't emit
		}
		// otherwise try next validator in the sequence
		skippedOfflineValidatorsCounter.Inc(1)
	}
	return false
}

func (em *Emitter) addTxs(e *inter.MutableEventPayload, sorted *transactionsByPriorityAndPriceAndNonce, classifier priorities.Classifier) {
	// Best-effort priority hinter: lets prioritized transactions be eagerly
	// included regardless of the per-transaction turn. Nil while the feature is
	// disabled, keeping behavior unchanged.
	hinter := em.newPriorityHinter()
	em.addTxsWithHinter(e, sorted, classifier, hinter)
}

// addTxsWithHinter appends transactions from sorted to the event e, honoring
// the event's gas-power and size budgets and this validator's per-transaction
// turn policy.
//
// # Prioritized inclusion
//
// No admission is unconditionally guaranteed: every candidate must pass the
// per-tx checks in tryAdd (size, epoch rules, gas power, sender conflicts,
// tx-pool freshness, bundle validity) and the shared stop-on-empty check.
// Beyond those:
//
//   - A prioritized tx for which it is this validator's turn is admitted in
//     priority-then-tip order and does not consume the hinter cap.
//   - A prioritized tx for another validator's turn is admitted only via the
//     priority hinter's eager path, subject to the per-entity per-event cap
//     (priorityHinter.config.MaxTxsPerEntityPerEvent, keyed by Priority.Id).
//     If the hinter is nil (feature disabled), foreign-priority admissions are
//     never attempted.
//   - Between successful phase-2 admissions the sender's next nonce may flip
//     back to this validator's turn; those admissions take the my-turn branch
//     and, like phase 1, do not consume the hinter cap.
//
// # Limit accounting
//
// Gas power (event gas power / event MaxEventGas / block MaxBlockGas) and
// total transaction size (maxTotalTransactionsSizeInEventInBytes) form a
// shared budget deducted inside tryAdd for every successful admission from
// any phase. The hinter's MaxTxsPerEntityPerEvent counts only foreign-
// priority admissions; my-turn admissions never consume it — including
// my-turn admissions that reached the phase-2 heap because an earlier nonce
// of the same sender was not-my-turn.
//
// # "Emit only if my-turn tx included" is best-effort
//
// To keep events from consisting solely of other validators' prioritized
// transactions, foreign-priority admissions are skipped when it is already
// known that this validator will contribute nothing on its own. The check
// is a cheap forward-peek at the non-prioritized heap performed after
// phase 1: if phase 1 admitted nothing and no non-prioritized head is my
// turn, foreign-priority admissions are skipped entirely.
//
// This is best-effort: a my-turn candidate spotted by the peek may still
// fail tryAdd (e.g. outdated, conflicting sender, invalid bundle,
// insufficient gas). When that happens, foreign-priority txs that were
// already admitted before phase 3 discovered the failure remain in the
// event. Conversely, my-turn admissions that only surface *within* the
// phase-2 loop (nonce+1 promotions after a hinter admission) can never
// trigger the skip because they only exist once phase 2 runs.
func (em *Emitter) addTxsWithHinter(e *inter.MutableEventPayload, sorted *transactionsByPriorityAndPriceAndNonce, classifier priorities.Classifier, hinter *priorityHinter) {
	maxGasUsed := em.maxGasPowerToUse(e)
	if maxGasUsed <= e.GasPowerUsed() {
		return
	}

	totalTxSizeInBytes := uint64(0)
	rules := em.world.GetRules()

	// tryAdd runs every per-transaction inclusion check except the turn and
	// priority-hinter checks. On success it mutates the event to include the
	// transaction and calls onAdded; on skip it calls onSkip. It returns true
	// when the loop must stop entirely (no room for even an empty transaction).
	tryAdd := func(tx *txpool.LazyTransaction, onAdded, onSkip func()) bool {
		resolvedTx := tx.Resolve()

		// check transaction size limits
		txSize := resolvedTx.Size()
		if totalTxSizeInBytes+txSize > maxTotalTransactionsSizeInEventInBytes {
			txsSkippedSizeLimit.Inc(1)
			onSkip()
			return false
		}

		sender, _ := types.Sender(em.world.TransactionSigner, resolvedTx)
		// check transaction epoch rules (tx type, gas price)
		if epochcheck.CheckTxs(types.Transactions{resolvedTx}, rules) != nil {
			txsSkippedEpochRules.Inc(1)
			onSkip()
			return false
		}
		// check there's enough gas power to originate the transaction
		if tx.Gas >= e.GasPowerLeft().Min() || e.GasPowerUsed()+tx.Gas >= maxGasUsed {
			txsSkippedNoValidatorGas.Inc(1)
			if params.TxGas >= e.GasPowerLeft().Min() || e.GasPowerUsed()+params.TxGas >= maxGasUsed {
				// stop if cannot originate even an empty transaction
				return true
			}
			onSkip()
			return false
		}
		// check not conflicted with already originated txs (in any connected event)
		if em.originatedTxs.TotalOf(sender) != 0 {
			txsSkippedConflictingSender.Inc(1)
			onSkip()
			return false
		}
		// check transaction is not outdated
		if !em.world.TxPool.Has(tx.Hash) {
			txsSkippedOutdated.Inc(1)
			onSkip()
			return false
		}
		// check validity of bundled transactions
		if em.world.GetRules().Upgrades.Brio && bundle.IsEnvelope(resolvedTx) && !em.isValidBundleTx(resolvedTx) {
			onSkip()
			return false
		}

		// add
		e.SetGasPowerUsed(e.GasPowerUsed() + tx.Gas)
		e.SetGasPowerLeft(e.GasPowerLeft().Sub(tx.Gas))
		e.SetTxs(append(e.Transactions(), resolvedTx))
		totalTxSizeInBytes += txSize
		onAdded()
		return false
	}

	isMyTurn := func(tx *txpool.LazyTransaction) bool {
		resolvedTx := tx.Resolve()
		sender, _ := types.Sender(em.world.TransactionSigner, resolvedTx)
		return em.isMyTxTurn(tx.Hash, sender, resolvedTx.Nonce(), time.Now(), em.validators.Load(), e.Creator(), idx.Epoch(em.epoch.Load()))
	}

	// Phase 1: prioritized heads for which it is this validator's turn.
	// Heads that are prioritized but not my turn are demoted into the
	// prioritized not-my-turn heap for phase 2.
	phase1Added := false
	for entry := sorted.PeekPrioHead(); entry != nil; entry = sorted.PeekPrioHead() {
		if !isMyTurn(entry.tx) {
			sorted.DemotePrioHead()
			continue
		}
		if tryAdd(entry.tx, func() {
			phase1Added = true
			sorted.ShiftPrioHead(classifier)
		}, sorted.DiscardPrioHead) {
			return
		}
	}

	// The invariant "events are never emitted solely to carry other validators'
	// prioritized transactions" means phase 2 must only run if phase 1 already
	// contributed or phase 3 will contribute. If phase 1 added nothing, peek
	// non-prioritized heads to check whether phase 3 has at least one my-turn
	// candidate; drop non-my-turn heads along the way (phase 3 would drop them
	// anyway). If none remain, skip phase 2 entirely.
	if !phase1Added {
		for entry := sorted.PeekNonPrioHead(); entry != nil; entry = sorted.PeekNonPrioHead() {
			if isMyTurn(entry.tx) {
				break
			}
			txsSkippedNotMyTurn.Inc(1)
			sorted.DiscardNonPrioHead()
		}
		if sorted.PeekNonPrioHead() == nil {
			return
		}
	}

	// Phase 2: prioritized heads that were not my turn when first observed.
	// A head that has become my turn (a subsequent nonce from a sender whose
	// earlier nonce landed here) is admitted directly without consuming the
	// per-entity hinter cap. Heads that are still not my turn go through the
	// hinter eligibility path for eager inclusion.
	for entry := sorted.PeekPrioNotMyTurnHead(); entry != nil; entry = sorted.PeekPrioNotMyTurnHead() {
		if isMyTurn(entry.tx) {
			if tryAdd(entry.tx, func() {
				sorted.ShiftPrioNotMyTurnHead(classifier)
			}, sorted.DiscardPrioNotMyTurnHead) {
				return
			}
			continue
		}
		resolvedTx := entry.tx.Resolve()
		eagerPrio, prioId := hinter.eligible(resolvedTx)
		if !eagerPrio {
			txsSkippedNotMyTurn.Inc(1)
			sorted.DiscardPrioNotMyTurnHead()
			continue
		}
		if tryAdd(entry.tx, func() {
			hinter.record(prioId)
			sorted.ShiftPrioNotMyTurnHead(classifier)
		}, sorted.DiscardPrioNotMyTurnHead) {
			return
		}
	}

	// Phase 3: ordinary (non-prioritized) heads, my-turn only.
	for entry := sorted.PeekNonPrioHead(); entry != nil; entry = sorted.PeekNonPrioHead() {
		if !isMyTurn(entry.tx) {
			txsSkippedNotMyTurn.Inc(1)
			sorted.DiscardNonPrioHead()
			continue
		}
		if tryAdd(entry.tx, func() {
			sorted.ShiftNonPrioHead()
		}, sorted.DiscardNonPrioHead) {
			return
		}
	}
}

// isValidBundleTx checks whether the given transaction is a valid bundle that
// could be emitted by this emitter.
func (em *Emitter) isValidBundleTx(tx *types.Transaction) bool {
	return em.isRunnableBundleTxInternal(tx, em.bundleCache, effectiveBundleGasHistogram)
}

func (em *Emitter) isRunnableBundleTxInternal(
	tx *types.Transaction,
	evalBundle evmcore.BundleEvaluator,
	effectiveGasHistogram utils.MetricsHistogram,
) bool {
	// Ignore if bundled transactions are not enabled.
	if !em.world.GetRules().Upgrades.TransactionBundles {
		return false
	}

	// Ignore if not a bundle transaction.
	if !bundle.IsEnvelope(tx) {
		return false
	}

	// Ignore if it is not a valid bundle transaction.
	_, plan, err := bundle.ValidateEnvelope(em.world.TransactionSigner, tx)
	if err != nil {
		return false
	}

	// Ignore if the next block is no longer in the range. If it is just the
	// next block, it is likely anyway too late, since the DAG consensus is
	// pipelined, but it is fine to error on the safe side here.
	if !plan.Range.IsInRange(uint64(em.world.GetLatestBlockIndex()) + 1) {
		return false
	}

	stateDb := em.world.StateDB()
	defer stateDb.Release()

	// Ignore if the same bundle has already been processed.
	if stateDb.HasBundleRecentlyBeenProcessed(plan.Hash()) {
		return false
	}

	// Skip bundles that are not runnable in the current state.
	adapter := &preCheckChainStateAdapter{external: em.world}
	bundleState := evalBundle.GetBundleState(adapter, stateDb, tx)

	// Update the gas efficiency metric for the bundle.
	if bundleState.GasEfficiency != nil {
		effectiveGasHistogram.Observe(*bundleState.GasEfficiency)
	}
	return bundleState.Executable
}

type preCheckChainStateAdapter struct {
	external External
}

func (a *preCheckChainStateAdapter) GetCurrentNetworkRules() opera.Rules {
	return a.external.GetRules()
}

func (a *preCheckChainStateAdapter) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	return a.external.Header(hash, number)
}

func (a *preCheckChainStateAdapter) GetCurrentChainConfig() *params.ChainConfig {
	return opera.CreateTransientEvmChainConfig(
		a.external.GetRules().NetworkID,
		a.external.GetUpgradeHeights(),
		a.external.GetLatestBlockIndex(),
	)
}

func (a *preCheckChainStateAdapter) GetLatestHeader() *evmcore.EvmHeader {
	lastBlock := a.external.GetLatestBlock()
	return a.external.Header(lastBlock.Hash(), lastBlock.Number)
}
