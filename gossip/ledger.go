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
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
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
	// It panics if another block is still in progress. A successful call
	// consumes the head; any later BeginBlock panics.
	BeginBlock(params BlockParams, onNewLog func(*core_types.Log)) (ExecutingBlock, error)
}

// ExecutingBlock is a block being executed. It owns the live state.
type ExecutingBlock interface {
	// stateDB returns the block's live state. It is unexported so that only
	// the gossip package can reach it, e.g. for the consensus callback.
	stateDB() state.StateDB

	// Run executes a batch of user transactions against the block being
	// assembled, honoring the UserTransactionsGasLimit shared by all calls on
	// the block and the block size limit, appends the accepted ones, and
	// returns the execution summary.
	Run(txs types.Transactions) evmcore.ProcessSummary

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

	// Commit persists the block without notifying the RPC layer. It consumes
	// the finalized block; any later Commit panics.
	Commit() CommittedBlock
}

// CommittedBlock is a persisted block that is not yet published.
type CommittedBlock interface {
	// Publish notifies the RPC layer of the block, logs it, and releases the
	// ledger so the next block can begin. Before calling it, the caller has to
	// make the block the store's latest block (BlockState.LastBlock), which
	// Head reads; otherwise it panics. It consumes the committed block; any
	// later call panics.
	Publish()
}

var (
	_ Ledger         = (*ledger)(nil)
	_ Head           = (*head)(nil)
	_ ExecutingBlock = (*executingBlock)(nil)
	_ FinalizedBlock = (*finalizedBlock)(nil)
	_ CommittedBlock = (*committedBlock)(nil)
)

// BlockParams holds the inputs for executing the next block.
type BlockParams struct {
	Time       inter.Timestamp
	Epoch      idx.Epoch
	PrevRandao common.Hash
	// GasLimit is the block's header gas limit; internal transactions run
	// within it.
	GasLimit uint64
	// UserTransactionsGasLimit is the gas available to the user transactions
	// of all Run calls on the block. It must be at most GasLimit.
	UserTransactionsGasLimit uint64
	Rules                    opera.Rules
}

// ledger is the main implementation of Ledger.
type ledger struct {
	evmModule         blockproc.EVM
	store             *Store
	feed              *ServiceFeed
	indexTransactions bool
	log               log.Logger

	// blockInProgress is set from BeginBlock until Publish. It is atomic
	// because endBlock calls Publish from a worker goroutine, not the one that
	// called BeginBlock.
	blockInProgress atomic.Bool
}

// NewLedger creates a Ledger over the given store, executing blocks with the
// given EVM module. The new-block feed may be nil. If indexTransactions is set,
// committed blocks also store their receipts, logs and transaction positions.
func NewLedger(store *Store, evm blockproc.EVM, feed *ServiceFeed, indexTransactions bool) *ledger {
	return &ledger{evmModule: evm, store: store, feed: feed, indexTransactions: indexTransactions, log: log.Root()}
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
) (_ ExecutingBlock, err error) {
	if h.consumed {
		panic("head already consumed")
	}
	if !h.ledger.blockInProgress.CompareAndSwap(false, true) {
		panic("another block is in progress")
	}
	defer func() {
		if err != nil {
			h.ledger.blockInProgress.Store(false)
		}
	}()

	start := time.Now()

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
		ledger:       h.ledger,
		statedb:      statedb,
		number:       number,
		params:       params,
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		start:        start,
	}, nil
}

// executingBlock is the main implementation of ExecutingBlock.
type executingBlock struct {
	ledger                  *ledger
	statedb                 state.StateDB
	number                  uint64
	params                  BlockParams
	evmProcessor            blockproc.EVMProcessor
	blockBuilder            *inter.BlockBuilder
	userTransactionsGasUsed uint64
	finalized               bool

	// start is the wall-clock time the block processing started, used to
	// measure and log the block processing latency.
	start time.Time
}

func (eb *executingBlock) stateDB() state.StateDB {
	if eb.finalized {
		panic("executing block already consumed")
	}
	return eb.statedb
}

func (eb *executingBlock) Run(txs types.Transactions) evmcore.ProcessSummary {
	if eb.finalized {
		panic("executing block already consumed")
	}
	remainingSize := uint64(math.MaxUint64)
	if eb.params.Rules.Upgrades.Brio {
		remainingSize = uint64(params.MaxBlockSize - rlpEncodedMaxHeaderSizeInBytes)
		for _, tx := range eb.blockBuilder.GetTransactions() {
			txSize := tx.Size()
			if txSize > remainingSize {
				// Still call evmProcessor execute with 0 remaining size to
				// track skipped transactions correctly.
				eb.ledger.log.Warn("block filled with recorded transactions")
				remainingSize = 0
				break
			}
			remainingSize -= txSize
		}
	}

	summary := eb.evmProcessor.Execute(txs, eb.params.UserTransactionsGasLimit-eb.userTransactionsGasUsed, remainingSize)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil { // < nil if skipped
			eb.blockBuilder.AddTransaction(
				processed.Transaction,
				processed.Receipt,
			)
			eb.userTransactionsGasUsed += processed.Receipt.GasUsed
		}
	}

	return summary
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
		ledger: eb.ledger,
		start:  eb.start,
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
	ledger    *ledger
	start     time.Time
	candidate *BlockCandidate
	committed bool
}

func (fb *finalizedBlock) Candidate() *BlockCandidate {
	return fb.candidate
}

func (fb *finalizedBlock) Commit() CommittedBlock {
	if fb.committed {
		panic("block already committed")
	}
	fb.committed = true

	blockIdx := idx.Block(fb.candidate.Block.Number)

	// Everything is written before the indexes leading to it, so that a reader
	// following an index finds what it points to.
	for _, tx := range fb.candidate.EvmBlock.Transactions {
		fb.ledger.store.evm.SetTx(tx.Hash(), tx)
	}

	if fb.ledger.indexTransactions && fb.candidate.Receipts.Len() != 0 {
		fb.ledger.store.evm.SetReceipts(blockIdx, fb.candidate.Receipts)
		for _, r := range fb.candidate.Receipts {
			fb.ledger.store.evm.IndexLogs(r.Logs...)
		}
	}

	fb.ledger.store.SetBlock(blockIdx, fb.candidate.Block)
	fb.ledger.store.evm.SetCachedEvmBlock(blockIdx, fb.candidate.EvmBlock)
	fb.ledger.store.SetBlockIndex(fb.candidate.Block.Hash(), blockIdx)

	if fb.ledger.indexTransactions {
		for i, tx := range fb.candidate.EvmBlock.Transactions {
			fb.ledger.store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{
				Block:       blockIdx,
				BlockOffset: uint32(i),
			})
		}
	}

	return &committedBlock{ledger: fb.ledger, start: fb.start, candidate: fb.candidate}
}

// committedBlock is the main implementation of CommittedBlock.
type committedBlock struct {
	ledger    *ledger
	start     time.Time
	candidate *BlockCandidate
	published bool
}

func (cb *committedBlock) Publish() {
	if cb.published {
		panic("block already published")
	}
	if latest := cb.ledger.store.GetLatestBlock(); latest == nil || latest.Hash() != cb.candidate.Block.Hash() {
		panic("block was not made the latest block before publishing")
	}
	cb.published = true

	if cb.ledger.feed != nil {
		var logs []*types.Log
		for _, r := range cb.candidate.Receipts {
			logs = append(logs, r.Logs...)
		}
		cb.ledger.feed.notifyAboutNewBlock(cb.candidate.EvmBlock, logs)
	}

	blockDuration := time.Duration(cb.candidate.Block.Duration)
	now := time.Now()
	blockAge := now.Sub(cb.candidate.Block.Time.Time())
	cb.ledger.log.Info("New block",
		"index", cb.candidate.Block.Number,
		"id", cb.candidate.Block.Hash(),
		"gas_used", cb.candidate.EvmBlock.GasUsed,
		"gas_rate", float64(cb.candidate.EvmBlock.GasUsed)/blockDuration.Seconds(),
		"base_fee", cb.candidate.EvmBlock.BaseFee.String(),
		"txs", fmt.Sprintf("%d/%d", len(cb.candidate.EvmBlock.Transactions), cb.candidate.NumSkipped),
		"age", utils.PrettyDuration(blockAge),
		"t", utils.PrettyDuration(now.Sub(cb.start)),
		"epoch", cb.candidate.EvmBlock.Epoch,
	)

	cb.ledger.blockInProgress.Store(false)
}
