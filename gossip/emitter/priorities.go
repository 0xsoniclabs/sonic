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
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/vm"
)

// priorityContext holds the state required to classify and rate-limit
// prioritized transactions against the current head block. It is owned by the
// emitter for the duration of one ordering build, which is the only user of the
// acquired statedb.
type priorityContext struct {
	classifier priorities.Classifier
	config     priorities.Config
	release    func()
}

// newPriorityContext builds the priority state for the current head block, or
// returns nil if priorities are disabled or the head state is unavailable. The
// caller must invoke release() (only when non-nil) when done.
func (em *Emitter) newPriorityContext() *priorityContext {
	rules := em.world.GetRules()
	if !rules.Upgrades.TransactionPriorities {
		return nil
	}
	lastBlock := em.world.GetLatestBlock()
	if lastBlock == nil {
		return nil
	}
	header := em.world.Header(lastBlock.Hash(), lastBlock.Number)
	if header == nil {
		return nil
	}
	statedb := em.world.StateDB()
	if statedb == nil {
		return nil
	}
	chainCfg := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		em.world.GetUpgradeHeights(),
		em.world.GetLatestBlockIndex(),
	)
	evm := vm.NewEVM(
		evmcore.NewEVMBlockContext(header, em.world, nil),
		statedb,
		chainCfg,
		opera.GetVmConfig(rules),
	)
	config := priorities.GetConfigOrFallback(rules.Upgrades, evm, priorityConfigFailures)
	return &priorityContext{
		classifier: priorities.NewEvmClassifier(rules.Upgrades, evm, em.world.TransactionSigner, statedb, priorityTxFailures),
		config:     config,
		release:    statedb.Release,
	}
}

// getConfig returns the rate limits read along with the context, or the
// zero-valued configuration — prioritizing nothing — if there is no context.
func (c *priorityContext) getConfig() priorities.Config {
	if c == nil {
		return priorities.Config{}
	}
	return c.config
}

// priorityHinter provides best-effort transaction-priority classification for
// the emitter. It is used to eagerly include prioritized transactions in an
// emitted event regardless of the per-transaction "turn", so that prioritized
// transactions reach the DAG (and thus a block) as quickly as possible.
//
// This is a hint only: it is evaluated against the current head state and never
// affects consensus. The authoritative priority ordering is re-derived during
// block formation (see gossip/c_block_callbacks_priorities.go).
type priorityHinter struct {
	config priorities.Config
	counts map[priorities.PriorityID]uint64
}

// newPriorityHinter builds a per-event hinter from the rate-limit configuration
// read while the current ordering was built, or returns nil if priorities are
// disabled.
func (em *Emitter) newPriorityHinter() *priorityHinter {
	if !em.world.GetRules().Upgrades.TransactionPriorities {
		return nil
	}
	return &priorityHinter{
		config: em.cache.priorityConfig,
		counts: map[priorities.PriorityID]uint64{},
	}
}

// eligible reports whether a transaction with the given priority should be
// eagerly included in the event despite not being this validator's turn: it
// must be prioritized and the per-entity per-event cap must not be exhausted.
// The priority has already been resolved when the transaction was placed on the
// ordering heap, so no registry query is issued here. The caller is responsible
// for enforcing the "do not emit an event solely for foreign priorities"
// invariant. It does not modify any state; call record after the transaction
// has actually been added.
func (h *priorityHinter) eligible(p priorities.Priority) (bool, priorities.PriorityID) {
	if h == nil {
		return false, priorities.PriorityID{}
	}
	if !p.IsPrioritized() {
		return false, priorities.PriorityID{}
	}
	if h.counts[p.ID] >= h.config.MaxPiggybackTxsPerEntityPerEvent {
		return false, priorities.PriorityID{}
	}
	return true, p.ID
}

// record accounts for a prioritized transaction that has been added to the
// event, counting it against the per-entity per-event cap.
func (h *priorityHinter) record(id priorities.PriorityID) {
	h.counts[id]++
}
