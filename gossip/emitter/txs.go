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
	"maps"
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
func (em *Emitter) isMyTxTurn(txHash common.Hash, sender common.Address, accountNonce uint64, now time.Time, validators *pos.Validators, offlineValidators map[idx.ValidatorID]bool, me idx.ValidatorID, epoch idx.Epoch) bool {
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
		if !offlineValidators[chosenValidator] {
			return false // chosen validator is online - don't emit
		}
		// otherwise try next validator in the sequence
		skippedOfflineValidatorsCounter.Inc(1)
	}
	return false
}

// newTurnPolicy returns the per-transaction turn check of this validator, used
// to stage the transactions of the ordering set.
//
// The set of offline validators the check consults is guarded by the world lock,
// so it is snapshotted here. The returned policy reads only that snapshot and can
// therefore be evaluated without holding the lock.
func (em *Emitter) newTurnPolicy(upgrades opera.Upgrades) func(tx *txpool.LazyTransaction) bool {
	if upgrades.SingleProposerBlockFormation {
		return func(*txpool.LazyTransaction) bool { return true }
	}
	em.world.Lock()
	offlineValidators := maps.Clone(em.offlineValidators)
	em.world.Unlock()

	return func(tx *txpool.LazyTransaction) bool {
		resolvedTx := tx.Resolve()
		sender, _ := types.Sender(em.world.TransactionSigner, resolvedTx)
		return em.isMyTxTurn(tx.Hash, sender, resolvedTx.Nonce(), time.Now(), em.validators.Load(), offlineValidators, em.config.Validator.ID, idx.Epoch(em.epoch.Load()))
	}
}

// addTxs appends transactions from sorted to the event e within its gas-power
// and size budgets.
//
// A transaction is admitted on this validator's turn, or eagerly while it is
// another validator's turn so that prioritized transactions reach a block
// quickly; anything else is dropped. An eager admission requires a priority and
// counts against MaxPiggybackTxsPerEntityPerEvent per Priority.ID and half of the
// gas budget, since prioritized candidates are staged first and a few large ones
// would otherwise starve this validator's own. Eager admissions are rolled back
// if the event carries nothing of this validator's own, and never affect
// consensus: the authoritative priority ordering is re-derived during block
// formation.
func (em *Emitter) addTxs(e *inter.MutableEventPayload, sorted *transactionsByPriorityAndPriceAndNonce) {
	maxGasUsed := em.maxGasPowerToUse(e)
	if maxGasUsed <= e.GasPowerUsed() {
		return
	}

	totalTxSizeInBytes := uint64(0)
	rules := em.world.GetRules()

	// Eager admissions per entity, and the share of the gas budget they may
	// consume, leaving the rest for this validator's own transactions.
	maxPiggybackTxs := em.cache.priorityConfig.MaxPiggybackTxsPerEntityPerEvent
	piggybackTxs := map[priorities.PriorityID]uint64{}
	maxPiggybackGas := (maxGasUsed - e.GasPowerUsed()) / 2
	piggybackGas := uint64(0)

	// State to restore if the event ends up carrying foreign-priority
	// transactions only.
	txPrefixOnEntry := e.Transactions()
	gasPowerUsedOnEntry := e.GasPowerUsed()
	gasPowerLeftOnEntry := e.GasPowerLeft()
	ownTxs := false
	eagerTxs := int64(0)

	for entry, ok := sorted.Peek(); ok; entry, ok = sorted.Peek() {
		// Another validator's turn: admit eagerly only if the transaction is
		// prioritized and the eager limits are not yet exhausted.
		if !entry.myTurn {
			priority := entry.priority
			if !priority.IsPrioritized() {
				txsSkippedNotMyTurn.Inc(1)
				sorted.PopSequence()
				continue
			}
			if piggybackTxs[priority.ID] >= maxPiggybackTxs ||
				piggybackGas+entry.tx.Gas > maxPiggybackGas {
				txsSkippedPiggybackLimit.Inc(1)
				sorted.PopSequence()
				continue
			}
		}

		tx := entry.tx
		resolvedTx := tx.Resolve()

		// check transaction size limits
		txSize := resolvedTx.Size()
		if totalTxSizeInBytes+txSize > maxTotalTransactionsSizeInEventInBytes {
			txsSkippedSizeLimit.Inc(1)
			sorted.PopSequence()
			continue
		}

		sender, _ := types.Sender(em.world.TransactionSigner, resolvedTx)
		// check transaction epoch rules (tx type, gas price)
		if epochcheck.CheckTxs(types.Transactions{resolvedTx}, rules) != nil {
			txsSkippedEpochRules.Inc(1)
			sorted.PopSequence()
			continue
		}
		// check there's enough gas power to originate the transaction
		if tx.Gas >= e.GasPowerLeft().Min() || e.GasPowerUsed()+tx.Gas >= maxGasUsed {
			txsSkippedNoValidatorGas.Inc(1)
			if params.TxGas >= e.GasPowerLeft().Min() || e.GasPowerUsed()+params.TxGas >= maxGasUsed {
				// stop if cannot originate even an empty transaction
				break
			}
			sorted.PopSequence()
			continue
		}
		// check not conflicted with already originated txs (in any connected event)
		if em.originatedTxs.TotalOf(sender) != 0 {
			txsSkippedConflictingSender.Inc(1)
			sorted.PopSequence()
			continue
		}
		// check transaction is not outdated
		if !em.world.TxPool.Has(tx.Hash) {
			txsSkippedOutdated.Inc(1)
			sorted.PopSequence()
			continue
		}
		// check validity of bundled transactions
		if em.world.GetRules().Upgrades.Brio && bundle.IsEnvelope(resolvedTx) && !em.isValidBundleTx(resolvedTx) {
			sorted.PopSequence()
			continue
		}

		// add
		e.SetGasPowerUsed(e.GasPowerUsed() + tx.Gas)
		e.SetGasPowerLeft(e.GasPowerLeft().Sub(tx.Gas))
		e.SetTxs(append(e.Transactions(), resolvedTx))
		totalTxSizeInBytes += txSize
		if entry.myTurn {
			ownTxs = true
		} else {
			piggybackGas += tx.Gas
			piggybackTxs[entry.priority.ID]++
			eagerTxs++
		}
		sorted.Shift()
	}

	// If this validator contributed none of its own transactions, roll the
	// eagerly included ones back.
	if !ownTxs && eagerTxs > 0 {
		e.SetTxs(txPrefixOnEntry)
		e.SetGasPowerUsed(gasPowerUsedOnEntry)
		e.SetGasPowerLeft(gasPowerLeftOnEntry)
		txsSkippedPiggybackRollback.Inc(eagerTxs)
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
	_, plan, err := bundle.ValidateEnvelope(
		em.world.TransactionSigner, tx, em.world.GetRules().Upgrades,
	)
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
