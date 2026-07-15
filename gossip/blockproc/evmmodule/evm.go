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

package evmmodule

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

//go:generate mockgen -source=evm.go -destination=evm_mock.go -package=evmmodule

type EVMModule struct {
	// forReplay selects the replaying state processor over the head-state one.
	// When replaying historical blocks, the block already contains the
	// post-execution transactions that subsidies and bundles generate, so they
	// must not be generated again; the replaying processor skips them.
	forReplay bool
}

// New creates an EVM module for head-state block production, as used by the live
// node.
func New() *EVMModule {
	return &EVMModule{}
}

// NewForReplay creates an EVM module that replays historical blocks: it executes
// the recorded transactions without re-generating the post-execution
// transactions of subsidies and bundles. Use it to reproduce already-produced
// blocks, not to produce new ones.
func NewForReplay() *EVMModule {
	return &EVMModule{forReplay: true}
}

// Start begins executing a block against the given live state. The parent is the
// header of the block this one chains onto, or nil for the genesis block, which has
// none; it supplies the parent hash the block records and the base fee this block
// charges. It is passed in rather than read out of reader because the parent may be
// speculative -- applied to the live state but not yet written to any store, so
// there is nothing to read it from.
func (p *EVMModule) Start(
	blockNumber idx.Block,
	blockTime inter.Timestamp,
	epoch idx.Epoch,
	statedb state.StateDB,
	reader evmcore.DummyChain,
	parent *evmcore.EvmHeader,
	onNewLog func(*core_types.Log),
	rules opera.Rules,
	evmCfg *params.ChainConfig,
	prevrandao common.Hash,
	metrics evmcore.BlockExecutionMetrics,
) blockproc.EVMProcessor {
	var prevBlockHash common.Hash
	var baseFee *big.Int
	if parent == nil {
		baseFee = gasprice.GetInitialBaseFee(rules.Economy)
	} else {
		prevBlockHash = parent.Hash
		baseFee = gasprice.GetBaseFeeForNextBlock(gasprice.ParentBlockInfo{
			BaseFee:  parent.BaseFee,
			Duration: parent.Duration,
			GasUsed:  parent.GasUsed,
		}, rules.Economy)
	}

	// Start block
	statedb.BeginBlock(uint64(blockNumber))

	return &OperaEVMProcessor{
		blockTime:        blockTime,
		epoch:            epoch,
		reader:           reader,
		statedb:          statedb,
		onNewLog:         onNewLog,
		rules:            rules,
		evmCfg:           evmCfg,
		blockIdx:         uint64(blockNumber),
		prevBlockHash:    prevBlockHash,
		prevRandao:       prevrandao,
		gasBaseFee:       baseFee,
		processorFactory: stateProcessorFactory{},
		metrics:          metrics,
		forReplay:        p.forReplay,
	}
}

type OperaEVMProcessor struct {
	blockTime inter.Timestamp
	epoch     idx.Epoch
	reader    evmcore.DummyChain
	statedb   state.StateDB
	onNewLog  func(*core_types.Log)
	rules     opera.Rules
	evmCfg    *params.ChainConfig

	blockIdx      uint64
	prevBlockHash common.Hash
	gasBaseFee    *big.Int

	gasUsed uint64

	processedTxs []evmcore.ProcessedTransaction
	prevRandao   common.Hash

	processorFactory _stateProcessorFactory

	metrics evmcore.BlockExecutionMetrics

	// forReplay selects the replaying state processor in Execute (see EVMModule).
	forReplay bool
}

func (p *OperaEVMProcessor) evmBlockWith(txs types.Transactions) *evmcore.EvmBlock {
	baseFee := p.rules.Economy.MinGasPrice
	if !p.rules.Upgrades.London {
		baseFee = nil
	} else if p.rules.Upgrades.Sonic {
		baseFee = p.gasBaseFee
	}

	prevRandao := common.Hash{}
	// This condition must be kept, otherwise Sonic will not be able to synchronize
	if p.rules.Upgrades.Sonic {
		prevRandao = p.prevRandao
	}

	var withdrawalsHash *common.Hash = nil
	if p.rules.Upgrades.Sonic {
		withdrawalsHash = &types.EmptyWithdrawalsHash
	}

	blobBaseFee := evmcore.GetBlobBaseFee()
	h := &evmcore.EvmHeader{
		Number:          new(big.Int).SetUint64(p.blockIdx),
		ParentHash:      p.prevBlockHash,
		Root:            common.Hash{}, // state root is added later
		Time:            p.blockTime,
		Coinbase:        evmcore.GetCoinbase(),
		GasLimit:        p.rules.Blocks.MaxBlockGas,
		GasUsed:         p.gasUsed,
		BaseFee:         baseFee,
		BlobBaseFee:     blobBaseFee.ToBig(),
		PrevRandao:      prevRandao,
		WithdrawalsHash: withdrawalsHash,
		Epoch:           p.epoch,
	}

	return evmcore.NewEvmBlock(h, txs)
}

func (p *OperaEVMProcessor) Execute(txs types.Transactions, gasLimit uint64, sizeLimit uint64) evmcore.ProcessSummary {
	var evmProcessor _stateProcessor
	if p.forReplay {
		evmProcessor = p.processorFactory.NewStateProcessorForReplay(p.evmCfg, p.reader, p.rules.Upgrades)
	} else {
		evmProcessor = p.processorFactory.NewStateProcessorForHeadState(p.evmCfg, p.reader, p.rules.Upgrades, p.metrics)
	}
	trueTxsOffset := int(0)
	for _, tx := range p.processedTxs {
		if tx.Receipt != nil {
			trueTxsOffset++
		}
	}

	vmConfig := opera.GetVmConfig(p.rules)

	// Process txs
	evmBlock := p.evmBlockWith(txs)
	summary := evmProcessor.Process(evmBlock, p.statedb, vmConfig, gasLimit, &p.gasUsed, trueTxsOffset, p.onNewLog, sizeLimit)

	p.processedTxs = append(p.processedTxs, summary.ProcessedTransactions...)

	return summary
}

func (p *OperaEVMProcessor) Finalize() (evmBlock *evmcore.EvmBlock, numSkipped int, receipts types.Receipts, staged state.StagedBlock) {
	transactions := make(types.Transactions, 0, len(p.processedTxs))
	receipts = make(types.Receipts, 0, len(p.processedTxs))
	for _, tx := range p.processedTxs {
		if tx.Receipt != nil {
			transactions = append(transactions, tx.Transaction)
			receipts = append(receipts, tx.Receipt)
		} else {
			numSkipped++
		}
	}

	evmBlock = p.evmBlockWith(transactions)

	// Apply the block to the live state. It is only staged: the caller decides
	// whether it is committed or taken back again, which is what lets a consensus
	// execute ahead of the decision to keep a block. Notably, this does not wait for
	// the block to reach the archive -- committing it is what starts that.
	staged, err := p.statedb.EndBlock(evmBlock.Number.Uint64())
	if err == nil && staged == nil {
		err = fmt.Errorf("state database applied block %v without returning a staged block", evmBlock.Number)
	}
	if err != nil {
		// the underlying database has collected an error during finalize or
		// a previous operation. State consistency and its persistence my
		// have been compromised.
		log.Error("Failed to finalize block %v: %v", evmBlock.Number, err)
		return evmBlock, numSkipped, receipts, staged
	}

	// Get state root
	evmBlock.Root = common.Hash(staged.StateHash())

	return
}

// _stateProcessorFactory is an internal interface to allow introducing mocked
// state processors in tests.
type _stateProcessorFactory interface {
	NewStateProcessorForHeadState(
		evmCfg *params.ChainConfig,
		reader evmcore.DummyChain,
		upgrades opera.Upgrades,
		metrics evmcore.BlockExecutionMetrics,
	) _stateProcessor

	NewStateProcessorForReplay(
		evmCfg *params.ChainConfig,
		reader evmcore.DummyChain,
		upgrades opera.Upgrades,
	) _stateProcessor
}

// _stateProcessor is an internal interface to allow introducing mocked
// state processors in tests.
type _stateProcessor interface {
	Process(
		block *evmcore.EvmBlock,
		statedb state.StateDB,
		vmCfg vm.Config,
		gasLimit uint64,
		gasUsed *uint64,
		trueTxOffset int,
		onNewLog func(*core_types.Log),
		remainingSize uint64,
	) evmcore.ProcessSummary
}

// stateProcessorFactory is the production implementation of the
// _stateProcessorFactory using the real evmcore.StateProcessor.
type stateProcessorFactory struct{}

func (stateProcessorFactory) NewStateProcessorForHeadState(
	evmCfg *params.ChainConfig,
	reader evmcore.DummyChain,
	upgrades opera.Upgrades,
	metrics evmcore.BlockExecutionMetrics,
) _stateProcessor {
	return evmcore.NewStateProcessorForHeadState(evmCfg, reader, upgrades, metrics)
}

func (stateProcessorFactory) NewStateProcessorForReplay(
	evmCfg *params.ChainConfig,
	reader evmcore.DummyChain,
	upgrades opera.Upgrades,
) _stateProcessor {
	return evmcore.NewStateProcessorForReplay(evmCfg, reader, upgrades)
}
