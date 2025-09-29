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

package scheduler

import (
	"math"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

//go:generate mockgen -source=processor.go -destination=processor_mock.go -package=scheduler

// processorFactory is an internal interface for a component that creates a
// transaction processor capable of test-running transactions in a block to
// be scheduled.
type processorFactory interface {
	beginBlock(*evmcore.EvmBlock) processor
}

// processor is an internal interface for a component that can process
// individual transactions to be scheduled in a block.
type processor interface {
	// run runs the given transaction in the context of the current block
	// and returns the result of the execution.
	run(tx *types.Transaction) (success bool, gasUsed uint64)

	// release releases the resources used by the processor. In particular, it
	// allows implementations to release temporary database state.
	release()
}

// ------ Adapters for the EVM ------

// Chain provides access to the chain state retained by the client required for
// test-running transactions.
type Chain interface {
	// DummyChain needs to be implemented in order to resolve past block hashes.
	// TODO: follow-up task - simplify this to a GetBlockHash(idx.Block) method.
	evmcore.DummyChain

	// GetCurrentNetworkRules returns the current network rules for the EVM.
	GetCurrentNetworkRules() opera.Rules

	// GetEvmChainConfig returns the chain configuration for the EVM at the
	// given block height
	GetEvmChainConfig(blockHeight idx.Block) *params.ChainConfig

	// StateDB returns a context for running transactions on the head state of
	// the chain. A non-committable state-DB instance is sufficient.
	StateDB() state.StateDB
}

// evmProcessorFactory is an implementation of the processorFactory that wraps
// the EVM state processor implementation provided by the evmcore package.
type evmProcessorFactory struct {
	// chain provides access to the chain state retained by the client,
	// including the current chain configuration and the state database.
	chain Chain
}

func (p *evmProcessorFactory) beginBlock(
	block *evmcore.EvmBlock,
) processor {
	// TODO: follow-up task - align this with c_block_callbacks.go
	chainCfg := p.chain.GetEvmChainConfig(idx.Block(block.Header().Number.Uint64()))
	vmConfig := opera.GetVmConfig(p.chain.GetCurrentNetworkRules())
	state := p.chain.StateDB()

	// The gas limit for transactions is enforced on a per-transaction level
	// in the scheduler. See the scheduler.Schedule method for details. The
	// total gas used for attempting to schedule transactions is not limited.
	gasLimit := uint64(math.MaxUint64)
	stateProcessor := evmcore.NewStateProcessor(
		chainCfg, p.chain, p.chain.GetCurrentNetworkRules().Upgrades,
	)
	return &evmProcessor{
		processor: stateProcessor.BeginBlock(block, state, vmConfig, gasLimit, nil),
		stateDb:   state,
	}
}

// evmProcessor is the implementation of the processor interface produced by the
// evmProcessorFactory. It retains an instance of the evmcore's
// TransactionProcessor, abstracted through the evmProcessorRunner interface, and
// the stateDb instance holding the temporary state of the EVM accumulating
// changes during the transaction execution. These changes are discarded when
// the processor is released.
type evmProcessor struct {
	processor evmProcessorRunner
	stateDb   state.StateDB
}

func (p *evmProcessor) run(tx *types.Transaction) (
	result bool, gasUsed uint64,
) {
	// Note: the index can be set to 0 since code running inside the EVM can not
	// obtain the position of a transaction in the block. It has thus no effect
	// on the scheduling of the transactions.
	processed := p.processor.Run(0, tx)

	// A single input transaction can lead to multiple processed transactions.
	// For instance, a sponsored transaction may be accompanied by a fee
	// charging transaction. We consider the transaction successful if the
	// provided transaction was executed, and we sum up the gas used by all
	// non-skipped transactions, as this is the total gas cost of running the
	// provided transaction.
	txWasProcessed := false
	for _, pt := range processed {
		if pt.Receipt != nil {
			gasUsed += pt.Receipt.GasUsed
			if pt.Transaction == tx {
				txWasProcessed = true
			}
		}
	}
	return txWasProcessed, gasUsed
}

func (p *evmProcessor) release() {
	p.stateDb.Release()
}

// evmProcessorRunner is an interface implemented by the evmcore's
// TransactionProcessor. The interface is defined instead of a direct dependency
// to avoid unnecessary dependencies and to facilitate mocking of the evmcore.
type evmProcessorRunner interface {
	// Run runs the given transaction in the context of the current block
	// where the index is the position of the transaction in the block.
	Run(index int, tx *types.Transaction) []evmcore.ProcessedTransaction
}
