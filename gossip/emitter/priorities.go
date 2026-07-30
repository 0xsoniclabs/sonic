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

// priorityContext holds the state required to classify and rate-limit
// prioritized transactions against the current head block. It is owned by the
// emitter for the duration of one ordering build, which is the only user of the
// acquired state db. The cache it classifies into outlives the context, see
// PriorityCache.
type priorityContext struct {
	cache      *evmcore.PriorityCache
	classifier priorities.Classifier
	config     priorities.Config
	releaseDB  func()
}

// newPriorityContext builds the priority context for the current head block, or
// returns nil if priorities are disabled or required state is not available.
// The caller must invoke release() (only when non-nil) when done.
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
	snapshot := statedb.InterTxSnapshot()
	config := priorities.GetConfigOrFallback(rules.Upgrades, evm, priorityConfigFailures)
	statedb.RevertToInterTxSnapshot(snapshot)
	return &priorityContext{
		cache:      em.priorityCache,
		classifier: priorities.NewEvmClassifier(rules.Upgrades, evm, em.world.TransactionSigner, statedb, priorityTxFailures),
		config:     config,
		releaseDB:  statedb.Release,
	}
}

// release releases the state db when the context is not nil.
func (c *priorityContext) release() {
	if c != nil {
		c.releaseDB()
	}
}

// priorityOf classifies a candidate transaction against the head state held by
// the context. If the context is nil or classification fails, the transaction
// is non-prioritized.
func (c *priorityContext) priorityOf(tx *types.Transaction) priorities.Priority {
	if c == nil {
		return priorities.Priority{}
	}
	return c.cache.GetOrClassify(tx, c.classifier)
}

// getConfig returns the rate limits read along with the context, or the
// zero-valued configuration — prioritizing nothing — if the context is nil.
func (c *priorityContext) getConfig() priorities.Config {
	if c == nil {
		return priorities.Config{}
	}
	return c.config
}
