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

package gossip

import (
	"fmt"
	"math"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

//go:generate go tool mockgen -source=ledger.go -package=gossip -destination=ledger_mock.go

// Ledger is the consensus-agnostic block processing component.
type Ledger interface {
	// Head returns a handle to the block the consensus head pointer points to,
	// through which the next block is begun. It fails if that block is not in
	// the store.
	Head() (Head, error)
}

// Head is a handle to the head block and the only way to begin the next one,
// so the next block is built on the same parent the consensus validated
// against.
type Head interface {
	// Block returns the head block.
	Block() *inter.Block

	// BeginBlock opens the live state at the head's state root and starts
	// executing the next block on it, forwarding emitted logs to onNewLog.
	// A successful call consumes the head; any later call panics.
	BeginBlock(params BlockParams, onNewLog func(*core_types.Log)) (ExecutingBlock, error)
}

// ExecutingBlock is a block being executed. It owns the live state.
type ExecutingBlock interface {
	// stateDB returns the block's live state. It is unexported so that only
	// the gossip package can reach it, e.g. for the consensus callback.
	stateDB() state.StateDB

	// Run executes a batch of user transactions against the block being
	// assembled, honoring UserTransactionsGasLimit and the block size limit,
	// appends the accepted ones, and maps each processed transaction to the
	// transaction that caused it.
	Run(txs types.Transactions) map[common.Hash]common.Hash

	// runInternal executes a batch of system (consensus-internal) transactions
	// within the block gas limit and without a size limit, appends the accepted
	// ones, and returns the execution summary. It is unexported so that only
	// the gossip package can reach it, e.g. for the consensus callback which
	// injects internal transactions.
	runInternal(txs types.Transactions) evmcore.ProcessSummary

	// Finalize commits the EVM state, assembles and hashes the block, and
	// returns the finalized block. It consumes the executing block; any later
	// call on it panics.
	Finalize() FinalizedBlock
}

// FinalizedBlock is an assembled and hashed block that is not yet persisted.
type FinalizedBlock interface {
	// Candidate returns the assembled block and its execution results.
	Candidate() *BlockCandidate
}

var (
	_ Ledger         = (*ledger)(nil)
	_ Head           = (*head)(nil)
	_ ExecutingBlock = (*executingBlock)(nil)
	_ FinalizedBlock = (*finalizedBlock)(nil)
)

// BlockParams holds the inputs for executing the next block.
type BlockParams struct {
	Time       inter.Timestamp
	Epoch      idx.Epoch
	PrevRandao common.Hash
	// GasLimit is the block's header gas limit; internal transactions run
	// within it.
	GasLimit uint64
	// UserTransactionsGasLimit is the gas available to user transactions
	// executed by Run. It must be at most GasLimit.
	UserTransactionsGasLimit uint64
	Rules                    opera.Rules
}

// ledger is the main implementation of Ledger.
type ledger struct {
	evmModule blockproc.EVM
	store     *Store
	feed      *ServiceFeed
}

// NewLedger creates a Ledger over the given store, executing blocks with the
// given EVM module. The new-block feed may be nil.
func NewLedger(store *Store, evm blockproc.EVM, feed *ServiceFeed) *ledger {
	return &ledger{evmModule: evm, store: store, feed: feed}
}

func (l *ledger) Head() (Head, error) {
	headIdx := l.store.GetLatestBlockIndex()
	block := l.store.GetBlock(headIdx)
	if block == nil {
		return nil, fmt.Errorf("head block %d not found in store", headIdx)
	}
	return &head{ledger: l, block: block}, nil
}

// head is the main implementation of Head.
type head struct {
	ledger   *ledger
	block    *inter.Block
	consumed bool
}

func (h *head) Block() *inter.Block {
	return h.block
}

func (h *head) BeginBlock(
	params BlockParams,
	onNewLog func(*core_types.Log),
) (ExecutingBlock, error) {
	if h.consumed {
		panic("head already consumed")
	}
	if params.Time <= h.block.Time {
		return nil, fmt.Errorf("block time %d is not after head time %d", params.Time, h.block.Time)
	}

	latest := h.ledger.store.GetLatestBlock()
	if latest == nil || latest.Hash() != h.block.Hash() {
		return nil, fmt.Errorf("block %d is no longer the ledger's head",
			h.block.Number)
	}
	statedb, err := h.ledger.store.evm.GetLiveStateDb(hash.Hash(h.block.StateRoot))
	if err != nil {
		return nil, err
	}
	h.consumed = true
	number := h.block.Number + 1
	evmCfg := opera.CreateTransientEvmChainConfig(
		params.Rules.NetworkID,
		h.ledger.store.GetUpgradeHeights(),
		idx.Block(number),
	)
	evmProcessor := h.ledger.evmModule.Start(
		idx.Block(number),
		params.Time,
		params.Epoch,
		statedb,
		&EvmStateReader{ServiceFeed: h.ledger.feed, store: h.ledger.store},
		onNewLog,
		params.Rules,
		evmCfg,
		params.PrevRandao,
		sonicFeaturesMetrics,
	)
	blockBuilder := inter.NewBlockBuilder().
		WithEpoch(params.Epoch).
		WithNumber(number).
		WithParentHash(h.block.Hash()).
		WithTime(params.Time).
		WithPrevRandao(params.PrevRandao).
		WithGasLimit(params.GasLimit).
		WithDuration(time.Duration(params.Time - h.block.Time))
	return &executingBlock{
		statedb:      statedb,
		number:       number,
		params:       params,
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
	}, nil
}

// executingBlock is the main implementation of ExecutingBlock.
type executingBlock struct {
	statedb      state.StateDB
	number       uint64
	params       BlockParams
	evmProcessor blockproc.EVMProcessor
	blockBuilder *inter.BlockBuilder
	finalized    bool
}

func (eb *executingBlock) stateDB() state.StateDB {
	if eb.finalized {
		panic("executing block already consumed")
	}
	return eb.statedb
}

func (eb *executingBlock) Run(txs types.Transactions) map[common.Hash]common.Hash {
	if eb.finalized {
		panic("executing block already consumed")
	}
	return processUserTransactions(
		eb.evmProcessor,
		eb.blockBuilder,
		txs,
		eb.params.UserTransactionsGasLimit,
		eb.params.Rules.Upgrades,
	)
}

func (eb *executingBlock) runInternal(txs types.Transactions) evmcore.ProcessSummary {
	if eb.finalized {
		panic("executing block already consumed")
	}
	summary := eb.evmProcessor.Execute(txs, eb.params.GasLimit, math.MaxUint64)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil {
			eb.blockBuilder.AddTransaction(processed.Transaction, processed.Receipt)
		}
	}
	return summary
}

// BlockCandidate is a processed-but-unpublished block produced by Finalize.
type BlockCandidate struct {
	Block      *inter.Block
	EvmBlock   *evmcore.EvmBlock
	Receipts   types.Receipts
	NumSkipped int
}

func (eb *executingBlock) Finalize() FinalizedBlock {
	if eb.finalized {
		panic("executing block already consumed")
	}
	eb.finalized = true

	// In the past, there was one epoch-sealing block (8054923 on Sonic mainnet)
	// in which the block gas limit was adapted in the sealing block itself
	// rather than in the following block. Every other block keeps the
	// block-start gas limit, so this one-time exception is reproduced by
	// stamping the canonical header gas limit of that block as a fixed
	// historical constant. The value is hard-coded (rather than read from the
	// post-seal rules) so the ledger never needs to observe the epoch's sealed
	// rules.
	if eb.params.Rules.NetworkID == 146 && eb.number == 8054923 {
		eb.blockBuilder.WithGasLimit(0x12a05f200)
	}

	evmBlock, numSkipped, receipts := eb.evmProcessor.Finalize()

	// Add results of the transaction processing to the block.
	eb.blockBuilder.
		WithStateRoot(common.Hash(evmBlock.Root)).
		WithGasUsed(evmBlock.GasUsed).
		WithBaseFee(evmBlock.BaseFee)

	// Complete the block.
	block := eb.blockBuilder.Build()
	evmBlock.Hash = block.Hash()
	evmBlock.Duration = time.Duration(block.Duration)

	// Update block-hash and -time values in receipts and logs.
	for i := range receipts {
		receipts[i].BlockHash = block.Hash()
		for j := range receipts[i].Logs {
			receipts[i].Logs[j].BlockHash = block.Hash()
			receipts[i].Logs[j].BlockTimestamp = uint64(block.Time.Unix())
		}
	}

	return &finalizedBlock{
		candidate: &BlockCandidate{
			Block:      block,
			EvmBlock:   evmBlock,
			Receipts:   receipts,
			NumSkipped: numSkipped,
		},
	}
}

// finalizedBlock is the main implementation of FinalizedBlock.
type finalizedBlock struct {
	candidate *BlockCandidate
}

func (fb *finalizedBlock) Candidate() *BlockCandidate {
	return fb.candidate
}
