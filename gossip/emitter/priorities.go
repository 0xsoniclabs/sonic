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

// priorityContext holds the per-head-block state required to classify and
// rate-limit prioritized transactions. A single instance is shared by both
// the hinter (eager inclusion) and the transaction-ordering heap, so only one
// statedb is acquired per head block.
type priorityContext struct {
	classifier priorities.Classifier
	config     priorities.Config
	release    func()
}

// newPriorityContext builds the per-block priority state, or returns nil if
// priorities are disabled or the head state is unavailable. The caller must
// invoke release() (only when non-nil) when done.
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
	statedb := em.world.StateDB() // TODO could be nil
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
	config := priorities.GetConfigOrFallback(rules.Upgrades, evm)
	return &priorityContext{
		classifier: priorities.NewEvmClassifier(rules.Upgrades, evm, em.world.TransactionSigner, statedb),
		config:     config,
		release:    statedb.Release,
	}
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
	counts map[[16]byte]uint64
}

// newPriorityHinter builds a per-event hinter from the cached priorityContext,
// or returns nil if priorities are disabled. Its lifetime is scoped to the
// event being built; the underlying statedb is owned by the cache.
func (em *Emitter) newPriorityHinter() *priorityHinter {
	ctx := em.cache.priorityCtx
	if ctx == nil {
		return nil
	}
	return &priorityHinter{
		config: ctx.config,
		counts: map[[16]byte]uint64{},
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
func (h *priorityHinter) eligible(p priorities.Priority) (bool, [16]byte) {
	if h == nil {
		return false, [16]byte{}
	}
	if !p.IsPrioritized() {
		return false, [16]byte{}
	}
	if h.counts[p.ID] >= h.config.MaxPiggybackTxsPerEntityPerEvent {
		return false, [16]byte{}
	}
	return true, p.ID
}

// record accounts for a prioritized transaction that has been added to the
// event, counting it against the per-entity per-event cap.
func (h *priorityHinter) record(id [16]byte) {
	h.counts[id]++
}
