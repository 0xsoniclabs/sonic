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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// priorityHinter provides best-effort transaction-priority classification for
// the emitter. It is used to eagerly include prioritized transactions in an
// emitted event regardless of the per-transaction "turn", so that prioritized
// transactions reach the DAG (and thus a block) as quickly as possible.
//
// This is a hint only: it is evaluated against the current head state and never
// affects consensus. The authoritative priority ordering is re-derived during
// block formation (see gossip/c_block_callbacks_priorities.go).
type priorityHinter struct {
	classifier priorities.Classifier
	config     priorities.Config
	release    func()
	counts     map[[32]byte]uint64
}

// newPriorityHinter builds a priorityHinter against the current head state, or
// returns nil if the feature is disabled or the head state is unavailable. The
// caller must invoke release() when done (only when the returned hinter is
// non-nil).
func (em *Emitter) newPriorityHinter() *priorityHinter {
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

	config, err := priorities.GetConfig(rules.Upgrades, evm)
	if err != nil {
		// Best-effort: fall back to the conservative default (which prioritizes
		// nothing). Correctness is unaffected because block formation is
		// authoritative.
		config = priorities.FallbackConfig
	}

	return &priorityHinter{
		classifier: priorities.NewEvmClassifier(rules.Upgrades, evm, em.world.TransactionSigner, statedb),
		config:     config,
		release:    statedb.Release,
		counts:     map[[32]byte]uint64{},
	}
}

// eligible reports whether the given transaction should be eagerly included in
// the event despite not being this validator's turn: it must be prioritized, the
// event must already carry at least one transaction (so events are never emitted
// solely for non-owned prioritized transactions), and the per-entity per-event
// cap must not be exhausted. It does not modify any state; call record after the
// transaction has actually been added.
func (h *priorityHinter) eligible(eventHasTxs bool, tx *types.Transaction) (bool, [32]byte) {
	if h == nil || !eventHasTxs {
		return false, [32]byte{}
	}
	p, err := h.classifier.Priority(tx)
	if err != nil || !p.IsPrioritized() {
		return false, [32]byte{}
	}
	if h.counts[p.Id] >= h.config.MaxTxsPerEntityPerEvent {
		return false, [32]byte{}
	}
	return true, p.Id
}

// record accounts for a prioritized transaction that has been added to the
// event, counting it against the per-entity per-event cap.
func (h *priorityHinter) record(id [32]byte) {
	h.counts[id]++
}
