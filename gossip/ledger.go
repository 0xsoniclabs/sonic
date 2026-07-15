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
// store, and the new-block feed). Separating block processing from the consensus
// behind this interface is the goal of the extraction: it lets the consensus
// implementation evolve — or be replaced — without changing block processing.
//
// The ledger owns a stack of speculative blocks: blocks applied to the live state
// but not yet made permanent. A consensus that certifies a block before finalizing
// it must execute a block to obtain the hash a quorum certifies, and discard it
// again when the round fails to certify, so several blocks are in flight at once.
// A block is assembled with BeginBlock, then Run, then Finalize; it is later made
// permanent with Commit or discarded with RevertTo.
//
// The stack enforces the ordering rules structurally rather than by convention:
// Commit always takes the OLDEST speculative block, so the committed chain can only
// advance in order and a block that may still be discarded never reaches the
// append-only archive; RevertTo only discards from the top, so a committed block
// can never be reverted and a mispredicted suffix is rolled back newest first.
//
// Each block chains onto the one below it, which is why the stack lives here rather
// than above: the parent hash, height and pre-state of a speculative block are the
// speculative tip's, and the tip is not in the store — only Commit puts a block
// there. Callers therefore do not supply them; the ledger derives them.
type Ledger interface {
	// BeginBlock starts assembling a new block, pushing it onto the speculative
	// stack. It chains onto the speculative tip — or onto the committed head, when
	// the stack is empty — deriving the parent hash and opening the live state at
	// the parent's state root. Logs emitted during execution are forwarded to
	// onNewLog. It reports an error if params.Number is not the tip's successor.
	BeginBlock(params BlockParams, onNewLog func(*core_types.Log)) error

	// StateDB returns the live state opened for the block being assembled, used by
	// the consensus callback to build the internal-transaction nonce source.
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

	// Finalize applies the block being assembled to the live state, assembles and
	// hashes it, and returns the resulting candidate. The block is live but not yet
	// permanent: it stays on the speculative stack until Commit or RevertTo.
	Finalize() *BlockCandidate

	// Commit makes the oldest speculative block permanent and returns it, without
	// waiting for the archive write it starts and without notifying the RPC layer —
	// Publish does both. It reports an error if there is no finalized block to
	// commit.
	Commit() (*BlockCandidate, error)

	// Publish awaits the archive write started by Commit for the given candidate and
	// notifies the RPC layer of it. It is called after the consensus has advanced
	// its head pointer, so the RPC head is observable before the notification.
	Publish(candidate *BlockCandidate)

	// RevertTo discards speculative blocks from the top of the stack, newest first,
	// until keep of them remain, taking each back out of the live state. Reverting
	// to the current depth is a no-op; keep is out of range if it is negative or
	// exceeds the current depth.
	RevertTo(keep int) error

	// Depth is the number of speculative blocks currently in flight.
	Depth() int

	// Head returns the last committed block. It reads the store and does not
	// consult the live state, which holds the speculative tip rather than the head
	// whenever anything is in flight.
	Head() (*inter.Block, error)

	// Tip returns the block the next BeginBlock chains onto: the newest speculative
	// block, or the committed head when nothing is in flight. It reports an error
	// if the Carmen live state does not match that block's state root.
	Tip() (*inter.Block, error)
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
	_ Ledger             = (*ledger)(nil)
	_ evmcore.DummyChain = (*stagedChain)(nil)
)

// BlockParams carries the inputs the ledger needs to execute and assemble a
// block. It holds no consensus concepts beyond what the block header records.
//
// It deliberately carries no parent hash and no parent state root: both are
// properties of the block below this one on the speculative stack, which the
// ledger owns and the caller does not. Deriving them there rather than accepting
// them here is what makes a block and the EVM header it executes against agree on
// their parent by construction.
type BlockParams struct {
	Number       uint64
	Time         inter.Timestamp // block timestamp
	Epoch        idx.Epoch       // consensus epoch the block belongs to (recorded on the block)
	PrevRandao   common.Hash
	GasLimit     uint64 // announced block gas limit (also the system batch budget)
	UserGasLimit uint64 // per-batch gas budget for user transactions
	Duration     time.Duration
	Rules        opera.Rules // block-start rules snapshot, used unchanged for the whole block
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

// ledger is the long-lived owner of the consensus-agnostic block-execution
// infrastructure and of the speculative block stack.
type ledger struct {
	evmModule blockproc.EVM
	store     *Store
	config    LedgerConfig

	// stack holds the speculative blocks, oldest first: applied to the live state,
	// not yet permanent. Its bottom is committed by Commit, its top discarded by
	// RevertTo, and its newest entry is the parent of the next block begun.
	stack []*blockProcessor

	// head is the last block this ledger committed, resolved from the store on
	// first use and advanced by Commit. It is not read back from the store,
	// because committing a block does not move the consensus head pointer: the
	// consensus advances that itself, after Commit and before Publish, so a ledger
	// reading it would see a head one block behind its own -- or, for a consensus
	// that finalizes several blocks ahead, several.
	head *inter.Block
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

// BeginBlock starts assembling a new block and pushes it onto the speculative
// stack: it derives the parent from the speculative tip (or the committed head,
// when the stack is empty), opens the live state at the parent's state root,
// starts the EVM processor against it, and starts the block builder. Logs emitted
// during execution are forwarded to onNewLog, reproducing the monolith's
// in-execution log delivery. The header gas limit is set to params.GasLimit here;
// the one historical net-146 exception is applied at Finalize. The opened live
// state is the same instance the EVM processor executes against, so the
// internal-tx nonce source built from StateDB observes the nonce increments of
// executed batches.
func (l *ledger) BeginBlock(
	params BlockParams,
	onNewLog func(*core_types.Log),
) error {
	parent, err := l.parentFor(params.Number)
	if err != nil {
		return err
	}
	// The EVM must reach the speculative parents too, not only the committed ones:
	// the BLOCKHASH opcode resolves by walking the chain backwards from the block
	// being executed, and a walk that reached a staged block through the store alone
	// would find nothing and silently report a zero hash for it and for every block
	// behind it. Ancestors below the stack come from the store as before.
	chain := &stagedChain{
		staged:   l.stagedHeaders(),
		fallback: &EvmStateReader{ServiceFeed: l.config.Feed, store: l.store},
	}

	parentStateRoot := common.Hash{}
	if parent != nil {
		parentStateRoot = parent.Root
	}
	statedb, err := l.store.evm.GetLiveStateDb(hash.Hash(parentStateRoot))
	if err != nil {
		return err
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

	parentHash := common.Hash{}
	if parent != nil {
		parentHash = parent.Hash
	}
	l.stack = append(l.stack, &blockProcessor{
		l:       l,
		params:  params,
		statedb: statedb,
		evmProcessor: l.evmModule.Start(
			idx.Block(params.Number),
			params.Time,
			params.Epoch,
			statedb,
			chain,
			parent,
			onNewLog,
			params.Rules,
			evmCfg,
			params.PrevRandao,
			l.config.ExecutionMetrics,
		),
		blockBuilder: inter.NewBlockBuilder().
			WithEpoch(params.Epoch).
			WithNumber(params.Number).
			WithParentHash(parentHash).
			WithTime(params.Time).
			WithPrevRandao(params.PrevRandao).
			WithGasLimit(params.GasLimit).
			WithDuration(params.Duration),
	})
	return nil
}

// StateDB returns the live state opened for the block being assembled.
func (l *ledger) StateDB() state.StateDB {
	return l.top().statedb
}

// Run executes a batch of user transactions in order, adding the accepted ones to
// the block while honoring the block's user gas limit and the block size limit,
// and returns the execution summary so the caller can react to the resulting
// receipts and logs.
func (l *ledger) Run(txs types.Transactions) evmcore.ProcessSummary {
	bp := l.top()
	return executeUserBatch(bp.evmProcessor, bp.blockBuilder, txs, bp.params.UserGasLimit, bp.params.Rules.Upgrades)
}

// runInternal executes a batch of system (consensus-internal) transactions with an
// unlimited gas and size budget, adds the accepted ones to the block, and returns
// the execution summary.
func (l *ledger) runInternal(txs types.Transactions) evmcore.ProcessSummary {
	bp := l.top()
	summary := bp.evmProcessor.Execute(txs, bp.params.GasLimit, math.MaxUint64)
	for _, processed := range summary.ProcessedTransactions {
		if processed.Receipt != nil {
			bp.blockBuilder.AddTransaction(processed.Transaction, processed.Receipt)
		}
	}
	return summary
}

// Depth is the number of speculative blocks currently in flight.
func (l *ledger) Depth() int {
	return len(l.stack)
}

// Head returns the last block this ledger committed. Until it has committed one,
// that is the head the store was opened at.
func (l *ledger) Head() (*inter.Block, error) {
	if l.head != nil {
		return l.head, nil
	}
	headIdx := l.store.GetBlockState().LastBlock.Idx
	block := l.store.GetBlock(headIdx)
	if block == nil {
		return nil, fmt.Errorf("head block %d not found in store", headIdx)
	}
	l.head = block
	return l.head, nil
}

// Tip returns the block the next BeginBlock chains onto and checks the Carmen
// live state against it. The live state holds the speculative tip -- every
// finalized block is applied to it -- so the tip, not the committed head, is what
// it must agree with.
func (l *ledger) Tip() (*inter.Block, error) {
	block, err := l.tip()
	if err != nil {
		return nil, err
	}
	if err := l.store.evm.CheckLiveStateHash(idx.Block(block.Number), hash.Hash(block.StateRoot)); err != nil {
		return nil, fmt.Errorf("ledger tip and Carmen live state do not match: %w", err)
	}
	return block, nil
}

// Finalize completes the block being assembled: it applies the net-146 header
// gas-limit exception, finalizes EVM execution to obtain the state root, assembles
// and hashes the block, stamps the block hash and time onto the receipts and their
// logs, and returns the resulting candidate. The block stays on the speculative
// stack: it is live but not permanent.
func (l *ledger) Finalize() *BlockCandidate {
	return l.top().finalize()
}

// Commit makes the oldest speculative block permanent and pops it off the bottom
// of the stack. Taking the oldest is what keeps the committed chain advancing in
// order, which the append-only archive requires.
func (l *ledger) Commit() (*BlockCandidate, error) {
	if len(l.stack) == 0 {
		return nil, fmt.Errorf("cannot commit: no block is in flight")
	}
	oldest := l.stack[0]
	if oldest.candidate == nil {
		return nil, fmt.Errorf("cannot commit block %d: it has not been finalized", oldest.params.Number)
	}
	oldest.commit()
	l.stack = append(l.stack[:0], l.stack[1:]...)
	l.head = oldest.candidate.Block
	return oldest.candidate, nil
}

// Publish awaits the archive write started by Commit and notifies the RPC layer of
// the committed block, feeding the new-block subscriptions. It is called after the
// consensus has advanced its head pointer.
func (l *ledger) Publish(candidate *BlockCandidate) {
	// Waiting happens before the feed is consulted: a ledger without a feed still
	// has to observe the outcome of its archive write, or the failure would go
	// unnoticed.
	waitForArchive(candidate)

	if l.config.Feed == nil {
		return
	}
	var logs []*types.Log
	for _, r := range candidate.Receipts {
		logs = append(logs, r.Logs...)
	}
	l.config.Feed.notifyAboutNewBlock(candidate.EvmBlock, logs)
}

// RevertTo takes speculative blocks back out of the live state from the top down
// until keep of them remain. Discarding from the top is what makes a committed
// block unreachable from here and rolls a mispredicted suffix back newest first,
// which is the order the live state can undo.
func (l *ledger) RevertTo(keep int) error {
	if keep < 0 || keep > len(l.stack) {
		return fmt.Errorf(
			"cannot revert to depth %d: %d blocks are in flight", keep, len(l.stack),
		)
	}
	for len(l.stack) > keep {
		top := l.stack[len(l.stack)-1]
		if err := top.rollback(); err != nil {
			return fmt.Errorf("failed to roll back block %d: %w", top.params.Number, err)
		}
		l.stack = l.stack[:len(l.stack)-1]
	}
	return nil
}

// top returns the block currently being assembled. It panics when no block has
// been begun: Run, StateDB and Finalize are meaningless without one, and returning
// a zero value would silently assemble a wrong block rather than report the
// programming error.
func (l *ledger) top() *blockProcessor {
	if len(l.stack) == 0 {
		panic("ledger: no block is in flight")
	}
	return l.stack[len(l.stack)-1]
}

// tip returns the block a new block chains onto: the newest speculative block, or
// the committed head when none is in flight.
func (l *ledger) tip() (*inter.Block, error) {
	if len(l.stack) == 0 {
		return l.Head()
	}
	top := l.stack[len(l.stack)-1]
	if top.candidate == nil {
		return nil, fmt.Errorf(
			"block %d is still being assembled and cannot be a parent", top.params.Number,
		)
	}
	return top.candidate.Block, nil
}

// parentFor returns the EVM header the block with the given number chains onto,
// and nil for the genesis block, which has none. It reports an error if the number
// is not the tip's successor -- the caller and the ledger disagreeing on the height
// means one of them is working from a stale head.
func (l *ledger) parentFor(number uint64) (*evmcore.EvmHeader, error) {
	if number == 0 {
		return nil, nil
	}
	if len(l.stack) > 0 {
		top := l.stack[len(l.stack)-1]
		if top.candidate == nil {
			return nil, fmt.Errorf(
				"cannot begin block %d: block %d is still being assembled",
				number, top.params.Number,
			)
		}
		if got := top.candidate.Block.Number; got+1 != number {
			return nil, fmt.Errorf(
				"cannot begin block %d on speculative tip %d", number, got,
			)
		}
		return top.candidate.EvmBlock.Header(), nil
	}

	head, err := l.Head()
	if err != nil {
		return nil, err
	}
	if head.Number+1 != number {
		return nil, fmt.Errorf("cannot begin block %d on committed head %d", number, head.Number)
	}
	return l.headerFor(head)
}

// headerFor builds the EVM header of a committed block, reading the rules of its
// epoch for the upgrade-gated fields.
func (l *ledger) headerFor(block *inter.Block) (*evmcore.EvmHeader, error) {
	var rules opera.Rules
	if es := l.store.GetHistoryEpochState(block.Epoch); es != nil {
		rules = es.Rules
	} else if block.Epoch == 0 {
		// There is no epoch state for epoch 0 comprising block 0. For this epoch,
		// London and Sonic upgrades are enabled. See EvmStateReader.getBlock, which
		// makes the same exception, and issue #72.
		rules.Upgrades.London = true
		rules.Upgrades.Sonic = true
	} else {
		return nil, fmt.Errorf("no epoch state for epoch %d of block %d", block.Epoch, block.Number)
	}
	return evmcore.ToEvmHeader(block, block.ParentHash, rules), nil
}

// stagedHeaders returns the headers of the speculative blocks, oldest first. They
// are complete parents needing no store read: Finalize stamps the block hash and
// duration onto the candidate's EVM block, and execution filled the rest. A block
// still being assembled has no header yet and is skipped -- it is above every block
// that could ask for it.
func (l *ledger) stagedHeaders() []*evmcore.EvmHeader {
	headers := make([]*evmcore.EvmHeader, 0, len(l.stack))
	for _, bp := range l.stack {
		if bp.candidate == nil {
			continue
		}
		headers = append(headers, bp.candidate.EvmBlock.Header())
	}
	return headers
}

// net146Block8054923GasLimit is the canonical header gas limit (0x12a05f200) of
// the single net-146 epoch-sealing block (8054923) that adopted the post-seal
// MaxBlockGas in its own header. It is a fixed historical value, deliberately not
// derived from any rule parameter, so historical replay stays stable regardless
// of future rule defaults.
const net146Block8054923GasLimit = 0x12a05f200 // 5,000,000,000

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

// finalize completes the block: it applies the net-146 header gas-limit
// exception, finalizes EVM execution to obtain the state root, assembles and
// hashes the block, stamps the block hash and time onto the receipts and their
// logs, and returns the resulting candidate.
func (bp *blockProcessor) finalize() *BlockCandidate {
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

// commit persists the finalized block: it promotes the block's content from the
// live state into the archive, and writes the receipt and log indexes (when
// transaction indexing is enabled), the block's transactions, the block and its
// hash index, and the cached EVM block. It does not wait for the archive write --
// Publish does -- so that write proceeds alongside the store writes and the
// consensus advancing its head pointer. It does not notify the RPC layer either;
// the consensus advances its head pointer between Commit and Publish.
func (bp *blockProcessor) commit() {
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

// rollback takes the finalized block back out of the live state, discarding the
// candidate. It is the counterpart of commit for a block that turned out not to be
// wanted -- a consensus that certifies a block before finalizing it must execute
// blocks it may still have to discard.
//
// Only a block that has not been committed can be rolled back, and blocks must be
// rolled back newest first. ledger.RevertTo is what enforces both.
func (bp *blockProcessor) rollback() error {
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

// stagedChain serves the ledger's speculative blocks as chain headers, falling
// back to the store for the committed history below them. It is what the EVM
// resolves BLOCKHASH against.
//
// The walk that resolves BLOCKHASH (evmcore.GetHashFn) steps from a block to its
// parent by asking for each ancestor in turn, and stops at the first one it cannot
// find. Since only Commit writes a block to the store, a walk served by the store
// alone would stop at the newest speculative block and report a zero hash for every
// block behind it -- silently, and only on a node that happens to have that block in
// flight, so two nodes executing the same block would disagree on its state root.
type stagedChain struct {
	staged   []*evmcore.EvmHeader // oldest first; the blocks not yet in the store
	fallback evmcore.DummyChain   // the committed history
}

// Header returns the header of the block with the given number, preferring a
// speculative block over the store. It returns nil if no such block is known, or if
// verificationHash is non-zero and names a different block.
func (c *stagedChain) Header(verificationHash common.Hash, number uint64) *evmcore.EvmHeader {
	for _, header := range c.staged {
		if header.Number.Uint64() != number {
			continue
		}
		if (verificationHash != common.Hash{}) && verificationHash != header.Hash {
			return nil
		}
		return header
	}
	return c.fallback.Header(verificationHash, number)
}

// waitForArchive awaits the archive write of the committed block. Blocks older
// than an hour are left to complete asynchronously, which keeps a node catching up
// with history from being paced by its archive; a recent block is waited for, so
// the latest state is available in both the live and the archive database once the
// block is published.
func waitForArchive(c *BlockCandidate) {
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
		log.Error("Failed to finalize block %v: %v", c.Block.Number, err)
	}
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
