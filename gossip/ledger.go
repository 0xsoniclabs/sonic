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

//go:generate mockgen -source=ledger.go -destination=ledger_mock.go -package=gossip

import (
	"fmt"
	"math"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"

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

// Ledger is the long-lived block-processing component the consensus is handed to
// execute, assemble, persist, and publish blocks. It owns the consensus-agnostic
// block-execution infrastructure (the EVM module, the execution metrics, the
// store, and the new-block feed) and produces a fresh BlockProcessor for each
// block. Separating block processing from the consensus behind this interface is
// the goal of the extraction: it lets the consensus implementation evolve — or be
// replaced — without changing block processing.
type Ledger interface {
	// BeginBlock starts assembling a new block: it opens the live state for
	// params.ParentStateRoot, starts the EVM processor against it, and starts the
	// block builder. Logs emitted during execution are forwarded to onNewLog. The
	// returned BlockProcessor owns the opened live state and carries the per-block
	// state through the rest of the pipeline.
	BeginBlock(params BlockParams, onNewLog func(*core_types.Log)) (BlockProcessor, error)

	// GetHeadBlock returns the ledger's current head: the last committed block. Its
	// StateRoot field is the finalized live-state root the next block must be opened
	// against (as BlockParams.ParentStateRoot). It returns an error if that state
	// root does not match the Carmen live state.
	GetHeadBlock() (*inter.Block, error)
}

// BlockProcessor drives a single block through the state machine
// Run* -> Finalize -> Commit -> Publish, or Run* -> Finalize -> Rollback for a
// block that turns out not to be wanted. It is produced by Ledger.BeginBlock and
// holds all the per-block state; the long-lived dependencies live on the Ledger.
//
// Finalize applies the block to the live state without making it permanent, so
// several blocks may be in flight at once: a consensus that certifies a block
// before finalizing it has to execute a block to obtain the hash a quorum
// certifies, and discard it again when the round fails to certify. Commit makes a
// block permanent, Rollback takes it back; blocks must be committed oldest first
// and rolled back newest first.
type BlockProcessor interface {
	// StateDB returns the live state opened for this block, used by the consensus
	// callback to build the internal-transaction nonce source.
	StateDB() state.StateDB

	// Run executes a batch of user transactions against the block being assembled,
	// honoring the block's user gas limit and the block size limit, appends the
	// accepted ones, and returns the execution summary.
	Run(txs types.Transactions) evmcore.ProcessSummary

	// runInternal executes a batch of system (consensus-internal) transactions with
	// an unlimited per-block gas and size budget, appends the accepted ones, and
	// returns the execution summary. It is unexported because only the consensus
	// callback injects internal transactions.
	runInternal(txs types.Transactions) evmcore.ProcessSummary

	// Finalize applies the block to the live state, assembles and hashes the block,
	// and returns the resulting candidate. The block is live but not yet permanent:
	// it can still be taken back with Rollback.
	Finalize() *BlockCandidate

	// Commit makes the finalized block permanent, without waiting for the archive
	// write it starts and without notifying the RPC layer.
	Commit()

	// Publish awaits the archive write started by Commit and notifies the RPC layer
	// of the committed block.
	Publish()

	// Rollback takes the finalized block back out of the live state, discarding the
	// candidate. It reports an error if the block cannot be taken back -- because it
	// was already committed, or because a newer block is still in flight.
	Rollback() error
}

var (
	sonicFeaturesMetrics = &evmcore.SonicBlockExecutionMetrics{
		SponsoredTxs:        utils.MetricsCounter(metrics.GetOrRegisterCounter("chain/sponsored", nil)),
		SkippedSponsoredTxs: utils.MetricsCounter(metrics.GetOrRegisterCounter("chain/sponsored/skipped", nil)),
		ExecutedBundles:     utils.MetricsCounter(metrics.GetOrRegisterCounter("chain/bundles", nil)),
		RolledBackBundles:   utils.MetricsCounter(metrics.GetOrRegisterCounter("chain/bundles/rolledback", nil)),
		InvalidBundles:      utils.MetricsCounter(metrics.GetOrRegisterCounter("chain/bundles/invalid", nil)),
		BundleEfficiency: utils.MetricsHistogram(utils.NewPrometheusHistogram(prometheus.HistogramOpts{
			Name:    "chain_bundle_gas_effective",
			Help:    "Effective gas usage ratio for a bundle transaction",
			Buckets: prometheus.LinearBuckets(0.00, 0.01, 100), // Buckets [0.00, 0.01, ..., 0.99, +inf]
		})),
	}
)

var (
	_ Ledger         = (*ledger)(nil)
	_ BlockProcessor = (*blockProcessor)(nil)
)

// BlockParams carries the inputs the ledger needs to execute and assemble a
// block. It holds no consensus concepts beyond what the block header records.
type BlockParams struct {
	Number          uint64
	Time            inter.Timestamp // block timestamp
	Epoch           idx.Epoch       // consensus epoch the block belongs to (recorded on the block)
	ParentHash      common.Hash
	PrevRandao      common.Hash
	ParentStateRoot common.Hash // finalized state root the live state is opened against
	GasLimit        uint64      // announced block gas limit (also the system batch budget)
	UserGasLimit    uint64      // per-batch gas budget for user transactions
	Duration        time.Duration
	Rules           opera.Rules // block-start rules snapshot, used unchanged for the whole block
}

// ledger is the long-lived owner of the consensus-agnostic block-execution
// infrastructure. It holds no per-block state; each block is processed by a
// fresh blockProcessor created by BeginBlock.
type ledger struct {
	evmModule blockproc.EVM
	store     *Store
	config    LedgerConfig
}

// LedgerConfig groups the configuration a Ledger needs beyond the store: the
// new-block feed (publish target and reader input, may be nil), whether to index
// transactions, the block-execution metrics sink (may be nil, in which case
// NewLedger uses the default sink), and whether the ledger replays historical
// blocks rather than producing new ones.
type LedgerConfig struct {
	Feed              *ServiceFeed
	IndexTransactions bool
	ExecutionMetrics  evmcore.BlockExecutionMetrics

	// Replay selects an EVM module that reproduces already-produced blocks
	// instead of producing new ones. It must be false for the live node: a
	// replaying ledger does not re-generate the post-execution transactions of
	// subsidies and bundles (which a historical block already contains), so it
	// reproduces such blocks but would produce incorrect new ones.
	Replay bool
}

// NewLedger creates a Ledger over the given store, backed by the production EVM
// implementation. A nil config.ExecutionMetrics defaults to the shared
// block-execution metrics sink. When config.Replay is set, the ledger uses the
// replaying EVM module (see LedgerConfig.Replay).
func NewLedger(store *Store, config LedgerConfig) Ledger {
	if config.ExecutionMetrics == nil {
		config.ExecutionMetrics = sonicFeaturesMetrics
	}
	evm := evmmodule.New()
	if config.Replay {
		evm = evmmodule.NewForReplay()
	}
	return newLedgerInternal(store, config, evm)
}

// newLedgerInternal creates a ledger with an explicit EVM module. It exists so
// tests can inject an EVM mock; production code uses NewLedger.
func newLedgerInternal(
	store *Store,
	config LedgerConfig,
	evm blockproc.EVM,
) *ledger {
	return &ledger{
		evmModule: evm,
		store:     store,
		config:    config,
	}
}

// blockProcessor owns block execution and assembly for a single block. It is
// created by ledger.BeginBlock and carries the per-block state through the rest
// of the pipeline; the long-lived dependencies are reached through l.
type blockProcessor struct {
	l            *ledger
	params       BlockParams
	statedb      state.StateDB
	evmProcessor blockproc.EVMProcessor
	blockBuilder *inter.BlockBuilder
	candidate    *BlockCandidate
}

// BeginBlock starts assembling a new block: it opens the live state for
// params.ParentStateRoot, starts the EVM processor against it, and starts the
// block builder. Logs emitted during execution are forwarded to onNewLog, reproducing
// the monolith's in-execution log delivery. The header gas limit is set to
// params.GasLimit here; the one historical net-146 exception is applied at
// Finalize. The opened live state is owned by the returned BlockProcessor and is
// the same instance the EVM processor executes against, so the internal-tx nonce
// source built from StateDB observes the nonce increments of executed batches.
func (l *ledger) BeginBlock(
	params BlockParams,
	onNewLog func(*core_types.Log),
) (BlockProcessor, error) {
	statedb, err := l.store.evm.GetLiveStateDb(hash.Hash(params.ParentStateRoot))
	if err != nil {
		return nil, err
	}
	// Derive the transient EVM chain config from the block-start rules snapshot
	// and the store's upgrade-height history. This mirrors the construction the
	// consensus callback used to do before passing it in, so the config is always
	// consistent with the ledger's own rules.
	evmCfg := opera.CreateTransientEvmChainConfig(
		params.Rules.NetworkID,
		l.store.GetUpgradeHeights(),
		idx.Block(params.Number),
	)
	return &blockProcessor{
		l:       l,
		params:  params,
		statedb: statedb,
		evmProcessor: l.evmModule.Start(
			idx.Block(params.Number),
			params.Time,
			params.Epoch,
			statedb,
			&EvmStateReader{ServiceFeed: l.config.Feed, store: l.store},
			onNewLog,
			params.Rules,
			evmCfg,
			params.PrevRandao,
			l.config.ExecutionMetrics,
		),
		blockBuilder: inter.NewBlockBuilder().
			WithEpoch(params.Epoch).
			WithNumber(params.Number).
			WithParentHash(params.ParentHash).
			WithTime(params.Time).
			WithPrevRandao(params.PrevRandao).
			WithGasLimit(params.GasLimit).
			WithDuration(params.Duration),
	}, nil
}

// GetHeadBlock returns the ledger's current head: the last committed block. It
// checks that the head block's state root matches the Carmen live state and
// returns an error if it does not.
func (l *ledger) GetHeadBlock() (*inter.Block, error) {
	headIdx := l.store.GetBlockState().LastBlock.Idx
	block := l.store.GetBlock(headIdx)
	if block == nil {
		return nil, fmt.Errorf("head block %d not found in store", headIdx)
	}
	if err := l.store.evm.CheckLiveStateHash(headIdx, hash.Hash(block.StateRoot)); err != nil {
		return nil, fmt.Errorf("ledger store head and Carmen live state do not match: %w", err)
	}
	return block, nil
}

// StateDB returns the live state opened for this block.
func (bp *blockProcessor) StateDB() state.StateDB {
	return bp.statedb
}

// Run executes a batch of user transactions in order, adding the accepted ones to
// the block while honoring the block's user gas limit and the block size limit,
// and returns the execution summary so the caller can react to the resulting
// receipts and logs.
func (bp *blockProcessor) Run(txs types.Transactions) evmcore.ProcessSummary {
	return executeUserBatch(bp.evmProcessor, bp.blockBuilder, txs, bp.params.UserGasLimit, bp.params.Rules.Upgrades)
}

// runInternal executes a batch of system (consensus-internal) transactions with an
// unlimited gas and size budget, adds the accepted ones to the block, and returns
// the execution summary.
func (bp *blockProcessor) runInternal(txs types.Transactions) evmcore.ProcessSummary {
	summary := bp.evmProcessor.Execute(txs, bp.params.GasLimit, math.MaxUint64)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil {
			bp.blockBuilder.AddTransaction(processed.Transaction, processed.Receipt)
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

	// staged is the candidate's content in the live state. It is applied but not
	// yet part of the archive: Commit promotes it, Rollback takes it back.
	staged state.StagedBlock
}

// net146Block8054923GasLimit is the canonical header gas limit (0x12a05f200) of
// the single net-146 epoch-sealing block (8054923) that adopted the post-seal
// MaxBlockGas in its own header. It is a fixed historical value, deliberately not
// derived from any rule parameter, so historical replay stays stable regardless
// of future rule defaults.
const net146Block8054923GasLimit = 0x12a05f200 // 5,000,000,000

// Finalize completes the block: it applies the net-146 header gas-limit
// exception, finalizes EVM execution to obtain the state root, assembles and
// hashes the block, stamps the block hash and time onto the receipts and their
// logs, and returns the resulting candidate.
func (bp *blockProcessor) Finalize() *BlockCandidate {
	// In the past, there was one epoch-sealing block (8054923 on network 146) in
	// which the block gas limit was adapted in the sealing block itself rather
	// than in the following block. Every other block keeps the block-start gas
	// limit, so this one-time exception is reproduced by stamping the canonical
	// header gas limit of that block as a fixed historical constant. The value is
	// hard-coded (rather than read from the post-seal rules) so the ledger never
	// needs to observe the epoch's sealed rules.
	if bp.params.Rules.NetworkID == 146 && bp.params.Number == 8054923 {
		bp.blockBuilder.WithGasLimit(net146Block8054923GasLimit)
	}

	evmBlock, numSkipped, receipts, staged := bp.evmProcessor.Finalize()

	// Add results of the transaction processing to the block.
	bp.blockBuilder.
		WithStateRoot(common.Hash(evmBlock.Root)).
		WithGasUsed(evmBlock.GasUsed).
		WithBaseFee(evmBlock.BaseFee)

	// Complete the block.
	block := bp.blockBuilder.Build()
	evmBlock.Hash = block.Hash()
	evmBlock.Duration = bp.params.Duration

	// Update block-hash and -time values in receipts and logs.
	for i := range receipts {
		receipts[i].BlockHash = block.Hash()
		for j := range receipts[i].Logs {
			receipts[i].Logs[j].BlockHash = block.Hash()
			receipts[i].Logs[j].BlockTimestamp = uint64(block.Time.Unix())
		}
	}

	bp.candidate = &BlockCandidate{
		Block:      block,
		EvmBlock:   evmBlock,
		Receipts:   receipts,
		NumSkipped: numSkipped,
		staged:     staged,
	}
	return bp.candidate
}

// Commit persists the finalized block: it promotes the block's content from the
// live state into the archive, and writes the receipt and log indexes (when
// transaction indexing is enabled), the block's transactions, the block and its
// hash index, and the cached EVM block. It does not wait for the archive write --
// Publish does -- so that write proceeds alongside the store writes and the
// consensus advancing its head pointer. It does not notify the RPC layer either;
// the consensus advances its head pointer between Commit and Publish.
func (bp *blockProcessor) Commit() {
	c := bp.candidate
	blockIdx := idx.Block(bp.params.Number)

	if c.staged != nil {
		if err := c.staged.Commit(); err != nil {
			log.Error("Failed to commit block %d: %v", bp.params.Number, err)
		}
	}

	if bp.l.config.IndexTransactions && c.Receipts.Len() != 0 {
		// Note: it's possible for receipts to get indexed twice by BR and block processing.
		bp.l.store.evm.SetReceipts(blockIdx, c.Receipts)
		for _, r := range c.Receipts {
			bp.l.store.evm.IndexLogs(r.Logs...)
		}
	}

	for i, tx := range bp.blockBuilder.GetTransactions() {
		bp.l.store.evm.SetTx(tx.Hash(), tx)
		if bp.l.config.IndexTransactions {
			bp.l.store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{
				Block:       blockIdx,
				BlockOffset: uint32(i),
			})
		}
	}

	bp.l.store.SetBlock(blockIdx, c.Block)
	bp.l.store.SetBlockIndex(c.Block.Hash(), blockIdx)
	bp.l.store.EvmStore().SetCachedEvmBlock(blockIdx, c.EvmBlock)
}

// Publish awaits the archive write started by Commit and notifies the RPC layer of
// the committed block, feeding the new-block subscriptions. It is called after the
// consensus has advanced its head pointer.
func (bp *blockProcessor) Publish() {
	c := bp.candidate

	// Waiting happens before the feed is consulted: a ledger without a feed still
	// has to observe the outcome of its archive write, or the failure would go
	// unnoticed.
	bp.waitForArchive()

	if bp.l.config.Feed == nil {
		return
	}
	var logs []*types.Log
	for _, r := range c.Receipts {
		logs = append(logs, r.Logs...)
	}
	bp.l.config.Feed.notifyAboutNewBlock(c.EvmBlock, logs)
}

// waitForArchive awaits the archive write of the committed block. Blocks older
// than an hour are left to complete asynchronously, which keeps a node catching up
// with history from being paced by its archive; a recent block is waited for, so
// the latest state is available in both the live and the archive database once the
// block is published.
func (bp *blockProcessor) waitForArchive() {
	c := bp.candidate
	if c.staged == nil {
		return
	}
	if time.Since(c.Block.Time.Time()) >= 1*time.Hour {
		return
	}
	if err := c.staged.Wait(); err != nil {
		// the underlying database has collected an error during finalize or a
		// previous operation. State consistency and its persistence may have been
		// compromised.
		log.Error("Failed to finalize block %v: %v", bp.params.Number, err)
	}
}

// Rollback takes the finalized block back out of the live state, discarding the
// candidate. It is the counterpart of Commit for a block that turned out not to be
// wanted -- a consensus that certifies a block before finalizing it must execute
// blocks it may still have to discard.
//
// Only a block that has not been committed can be rolled back, and blocks must be
// rolled back newest first.
func (bp *blockProcessor) Rollback() error {
	c := bp.candidate
	if c == nil {
		return fmt.Errorf("cannot roll back block %d: it has not been finalized", bp.params.Number)
	}
	bp.candidate = nil
	if c.staged == nil {
		return nil
	}
	return c.staged.Rollback()
}

// executeUserBatch executes user transactions in order, adding them to the block
// until all transactions are processed or the gas/block size limit is reached,
// and returns the execution summary.
func executeUserBatch(
	evmProcessor blockproc.EVMProcessor,
	blockBuilder *inter.BlockBuilder,
	orderedTxs []*types.Transaction,
	userTransactionGasLimit uint64,
	upgrades opera.Upgrades,
) evmcore.ProcessSummary {
	remainingSize := uint64(math.MaxUint64)
	if upgrades.Brio {
		remainingSize = uint64(params.MaxBlockSize - rlpEncodedMaxHeaderSizeInBytes)
		for _, tx := range blockBuilder.GetTransactions() {
			txSize := tx.Size()
			if txSize > remainingSize {
				// Still call evmProcessor execute with 0 remaining size to track skipped transactions correctly.
				log.Warn("block filled with only internal transactions")
				remainingSize = 0
				break
			}
			remainingSize -= txSize
		}
	}

	summary := evmProcessor.Execute(orderedTxs, userTransactionGasLimit, remainingSize)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil { // < nil if skipped
			blockBuilder.AddTransaction(
				processed.Transaction,
				processed.Receipt,
			)
		}
	}

	return summary
}
