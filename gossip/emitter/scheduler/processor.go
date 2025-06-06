package scheduler

import (
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
	// and returns the result of the execution. The gas limit is the maximum
	// amount of gas that can be used by the transaction.
	run(tx *types.Transaction, gasLimit uint64) (success bool, gasUsed uint64)

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
	vmConfig := opera.DefaultVMConfig
	state := p.chain.StateDB()

	stateProcessor := evmcore.NewStateProcessor(chainCfg, p.chain)
	return &evmProcessor{
		processor: stateProcessor.BeginBlock(block, state, vmConfig, nil),
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

func (p *evmProcessor) run(tx *types.Transaction, gasLimit uint64) (
	result bool, gasUsed uint64,
) {
	// Note: the index can be set to 0 since code running inside the EVM can not
	// obtain the position of a transaction in the block. It has thus no effect
	// on the scheduling of the transactions.
	receipt, skipped, err := p.processor.Run(0, tx)
	if skipped || err != nil || receipt == nil {
		return false, 0
	}
	return receipt.GasUsed < gasLimit, receipt.GasUsed
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
	Run(index int, tx *types.Transaction) (*types.Receipt, bool, error)
}
