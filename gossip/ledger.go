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
	"errors"
	"fmt"
	"math"
	"sync"
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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

// BlockParams holds the inputs for executing the next block.
type BlockParams struct {
	// Must follow the head's number.
	Number uint64
	// Block timestamp.
	Time inter.Timestamp
	// Consensus epoch the block belongs to (recorded on the block).
	Epoch      idx.Epoch
	PrevRandao common.Hash
	// Announced block gas limit (also the system batch budget).
	GasLimit uint64
	// Per-batch gas budget for user transactions.
	UserGasLimit uint64
	// Block-start rules snapshot, used unchanged for the whole block.
	Rules opera.Rules
}

// Ledger is the consensus-agnostic block processing component. Of the blocks
// it processes, it only tracks whether one is open; the per-block state lives
// on the handles.
type Ledger struct {
	evmModule blockproc.EVM
	store     *Store
	config    LedgerConfig

	mu   sync.Mutex
	open bool // a block has begun and is not published yet
}

// LedgerConfig groups the configuration a Ledger needs beyond the store: the
// new-block feed (publish target and reader input, may be nil), whether to
// index transactions, and the block-execution metrics sink (may be nil, in
// which case NewLedger uses the default sink).
type LedgerConfig struct {
	Feed              *ServiceFeed
	IndexTransactions bool
	ExecutionMetrics  evmcore.BlockExecutionMetrics
}

// NewLedger creates a Ledger over the given store, backed by the production EVM
// implementation. A nil config.ExecutionMetrics defaults to the shared
// block-execution metrics sink.
func NewLedger(store *Store, config LedgerConfig) *Ledger {
	if config.ExecutionMetrics == nil {
		config.ExecutionMetrics = sonicFeaturesMetrics
	}
	return newLedgerInternal(store, config, evmmodule.New())
}

// newLedgerInternal creates a ledger with an explicit EVM module. It exists so
// tests can inject an EVM mock; production code uses NewLedger.
func newLedgerInternal(
	store *Store,
	config LedgerConfig,
	evm blockproc.EVM,
) *Ledger {
	return &Ledger{
		evmModule: evm,
		store:     store,
		config:    config,
	}
}

// Head returns a handle to the block the consensus head pointer points to,
// through which the next block is begun. It fails if that block is not in
// the store.
func (l *Ledger) Head() (*Head, error) {
	headIdx := l.store.GetLatestBlockIndex()
	block := l.store.GetBlock(headIdx)
	if block == nil {
		return nil, fmt.Errorf("head block %d not found in store", headIdx)
	}
	return &Head{ledger: l, block: block}, nil
}

// Head is a handle to the head block and the only way to begin the next one,
// so the next block is built on the same parent the consensus validated
// against.
type Head struct {
	ledger   *Ledger
	block    *inter.Block
	consumed bool
}

// Block returns the head block.
func (h *Head) Block() *inter.Block {
	return h.block
}

// BeginBlock opens the live state at the head's state root and starts
// executing the next block on it, forwarding emitted logs to onNewLog.
func (h *Head) BeginBlock(
	params BlockParams,
	onNewLog func(*core_types.Log),
) (*ExecutingBlock, error) {
	start := time.Now()
	if h.consumed {
		return nil, errors.New("head already consumed")
	}
	if params.Number != h.block.Number+1 {
		return nil, fmt.Errorf("block %d does not follow head %d",
			params.Number, h.block.Number)
	}
	if params.Time <= h.block.Time {
		return nil, fmt.Errorf("block time %d is not after head time %d", params.Time, h.block.Time)
	}

	h.ledger.mu.Lock()
	defer h.ledger.mu.Unlock()
	if h.ledger.open {
		return nil, errors.New("another block is open")
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
	evmCfg := opera.CreateTransientEvmChainConfig(
		params.Rules.NetworkID,
		h.ledger.store.GetUpgradeHeights(),
		idx.Block(params.Number),
	)
	h.ledger.open = true
	return &ExecutingBlock{
		ledger:  h.ledger,
		params:  params,
		statedb: statedb,
		evmProcessor: h.ledger.evmModule.Start(
			idx.Block(params.Number),
			params.Time,
			params.Epoch,
			statedb,
			&EvmStateReader{ServiceFeed: h.ledger.config.Feed, store: h.ledger.store},
			onNewLog,
			params.Rules,
			evmCfg,
			params.PrevRandao,
			h.ledger.config.ExecutionMetrics,
		),
		blockBuilder: inter.NewBlockBuilder().
			WithEpoch(params.Epoch).
			WithNumber(params.Number).
			WithParentHash(h.block.Hash()).
			WithTime(params.Time).
			WithPrevRandao(params.PrevRandao).
			WithGasLimit(params.GasLimit).
			WithDuration(time.Duration(params.Time - h.block.Time)),
		start: start,
	}, nil
}

// ExecutingBlock is a block being executed. It owns the live state.
type ExecutingBlock struct {
	ledger       *Ledger
	params       BlockParams
	statedb      state.StateDB
	evmProcessor blockproc.EVMProcessor
	blockBuilder *inter.BlockBuilder
	finalized    bool

	// start is the wall-clock time the block processing started, used to
	// measure and log the block processing latency.
	start time.Time
}

// stateDB returns the block's live state. It is unexported so that only
// the gossip package can reach it, e.g. for the consensus callback.
func (eb *ExecutingBlock) stateDB() state.StateDB {
	if eb.finalized {
		panic("stateDB called after Finalize")
	}
	return eb.statedb
}

// Run executes a batch of user transactions in order, adding the accepted ones
// to the block while honoring the block's user gas limit and the block size
// limit, and returns the execution summary so the caller can react to the
// resulting receipts and logs.
func (eb *ExecutingBlock) Run(txs types.Transactions) evmcore.ProcessSummary {
	if eb.finalized {
		panic("Run called after Finalize")
	}
	remainingSize := uint64(math.MaxUint64)
	if eb.params.Rules.Upgrades.Brio {
		remainingSize = uint64(params.MaxBlockSize - rlpEncodedMaxHeaderSizeInBytes)
		for _, tx := range eb.blockBuilder.GetTransactions() {
			txSize := tx.Size()
			if txSize > remainingSize {
				// Still call evmProcessor execute with 0 remaining size to
				// track skipped transactions correctly.
				log.Warn("block filled with only internal transactions")
				remainingSize = 0
				break
			}
			remainingSize -= txSize
		}
	}

	summary := eb.evmProcessor.Execute(txs, eb.params.UserGasLimit, remainingSize)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil { // < nil if skipped
			eb.blockBuilder.AddTransaction(
				processed.Transaction,
				processed.Receipt,
			)
		}
	}

	return summary
}

// runInternal executes a batch of system (consensus-internal) transactions with
// an unlimited gas and size budget, adds the accepted ones to the block, and
// returns the execution summary.
func (eb *ExecutingBlock) runInternal(txs types.Transactions) evmcore.ProcessSummary {
	if eb.finalized {
		panic("runInternal called after Finalize")
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

// sonicMainnetBlock8054923GasLimit is the canonical header gas limit
// (0x12a05f200) of the single Sonic mainnet epoch-sealing block (8054923, hash
// 0x14a024348f821220ac90dc28422ca367ed7e47cf0c491d97e9a4834925cb4c90) that
// adopted the post-seal MaxBlockGas in its own header. It is a fixed historical
// value, deliberately not derived from any rule parameter, so historical replay
// stays stable regardless of future rule defaults.
const sonicMainnetBlock8054923GasLimit = 0x12a05f200 // 5,000,000,000

// Finalize completes the block: it applies the Sonic mainnet header gas-limit
// exception, finalizes EVM execution to obtain the state root, assembles and
// hashes the block, stamps the block hash and time onto the receipts and their
// logs, and returns the finalized block.
func (eb *ExecutingBlock) Finalize() *FinalizedBlock {
	if eb.finalized {
		panic("Finalize called twice")
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
	if eb.params.Rules.NetworkID == 146 && eb.params.Number == 8054923 {
		eb.blockBuilder.WithGasLimit(sonicMainnetBlock8054923GasLimit)
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

	return &FinalizedBlock{
		block: eb,
		candidate: &BlockCandidate{
			Block:      block,
			EvmBlock:   evmBlock,
			Receipts:   receipts,
			NumSkipped: numSkipped,
		},
	}
}

// FinalizedBlock is an assembled and hashed block that is not yet persisted.
type FinalizedBlock struct {
	block     *ExecutingBlock
	candidate *BlockCandidate
	committed bool
}

// Candidate returns the assembled block and its execution results.
func (fb *FinalizedBlock) Candidate() *BlockCandidate {
	return fb.candidate
}

// Commit persists the finalized block: the receipt and log indexes and the
// transaction positions (when transaction indexing is enabled), the transaction
// bodies, the block and its hash index, and the cached EVM block. It does not
// notify the RPC layer; the consensus advances its head pointer between Commit
// and Publish.
func (fb *FinalizedBlock) Commit() *CommittedBlock {
	if fb.committed {
		panic("Commit called twice")
	}
	fb.committed = true

	c := fb.candidate
	l := fb.block.ledger
	blockIdx := idx.Block(fb.block.params.Number)

	if l.config.IndexTransactions && c.Receipts.Len() != 0 {
		l.store.evm.SetReceipts(blockIdx, c.Receipts)
		for _, r := range c.Receipts {
			l.store.evm.IndexLogs(r.Logs...)
		}
	}

	for i, tx := range fb.block.blockBuilder.GetTransactions() {
		l.store.evm.SetTx(tx.Hash(), tx)
		if l.config.IndexTransactions {
			l.store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{
				Block:       blockIdx,
				BlockOffset: uint32(i),
			})
		}
	}

	l.store.SetBlock(blockIdx, c.Block)
	l.store.SetBlockIndex(c.Block.Hash(), blockIdx)
	l.store.EvmStore().SetCachedEvmBlock(blockIdx, c.EvmBlock)
	return &CommittedBlock{finalized: fb}
}

// CommittedBlock is a persisted block whose head pointer the consensus has to
// advance before publishing it.
type CommittedBlock struct {
	finalized *FinalizedBlock
	published bool
}

// Publish logs the committed block and notifies the RPC layer of it, feeding
// the new-block subscriptions, and then closes the block so the next one can
// begin. It is called after the consensus has advanced its head pointer.
func (cb *CommittedBlock) Publish() {
	if cb.published {
		panic("Publish called twice")
	}
	cb.published = true

	c := cb.finalized.candidate
	l := cb.finalized.block.ledger
	blockDuration := time.Duration(c.Block.Duration)
	now := time.Now()
	blockAge := now.Sub(c.Block.Time.Time())
	log.Info("New block",
		"index", c.Block.Number,
		"id", c.Block.Hash(),
		"gas_used", c.EvmBlock.GasUsed,
		"gas_rate", float64(c.EvmBlock.GasUsed)/blockDuration.Seconds(),
		"base_fee", c.EvmBlock.BaseFee.String(),
		"txs", fmt.Sprintf("%d/%d", len(c.EvmBlock.Transactions), c.NumSkipped),
		"age", utils.PrettyDuration(blockAge),
		"t", utils.PrettyDuration(now.Sub(cb.finalized.block.start)),
		"epoch", c.EvmBlock.Epoch,
	)

	if l.config.Feed != nil {
		var logs []*types.Log
		for _, r := range c.Receipts {
			logs = append(logs, r.Logs...)
		}
		l.config.Feed.notifyAboutNewBlock(c.EvmBlock, logs)
	}

	l.mu.Lock()
	l.open = false
	l.mu.Unlock()
}
