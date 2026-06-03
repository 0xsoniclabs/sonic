// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

//go:generate mockgen -source=state_processor.go -destination=state_processor_mock.go -package=evmcore

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config    *params.ChainConfig // Chain configuration options
	bc        DummyChain          // Canonical block chain
	upgrades  opera.Upgrades      // Enabled network upgrades
	forReplay bool                // Whether the state processor is used for replaying transactions or for head state processing.

	metrics BlockExecutionMetrics
}

// NewStateProcessorForHeadState initializes a new StateProcessor for head state processing.
func NewStateProcessorForHeadState(
	config *params.ChainConfig,
	bc DummyChain,
	upgrades opera.Upgrades,
	metrics BlockExecutionMetrics,
) *StateProcessor {
	return &StateProcessor{
		config:   config,
		bc:       bc,
		upgrades: upgrades,
		metrics:  metrics,
	}
}

// NewStateProcessorForReplay initializes a new StateProcessor for replaying transactions.
func NewStateProcessorForReplay(
	config *params.ChainConfig,
	bc DummyChain,
	upgrades opera.Upgrades,
) *StateProcessor {
	return &StateProcessor{
		config:    config,
		bc:        bc,
		upgrades:  upgrades,
		forReplay: true,
	}
}

// ProcessSummary contains the result of processing a list of transactions,
// including the list of processed transactions with their receipts and the
// total execution cost. The execution cost tracks the gas consumed during
// processing, including gas from rolled-back bundles, which distinguishes it
// from the usedGas counter that gets reverted on snapshot rollback.
type ProcessSummary struct {
	// ProcessedTransactions is a list of processed transactions
	// with their receipts (nil receipt for skipped transactions).
	ProcessedTransactions []ProcessedTransaction

	// ExecutionCost is the total execution cost of processing the transactions,
	// including gas from rolled-back bundles.
	ExecutionCost core_types.ExecutionCost

	// CausedBy is a map tx.Hash -> tx.Hash, where the key is the hash
	// of the transaction that got processed and the value is the hash
	// of the transaction that caused the processing of the key transaction.
	// - For regular transactions, the value is the same as the key.
	// - For sponsored transactions, the fees-payment transaction is mapped
	// to the sponsored transaction.
	// - For transactions included as part of a bundle, the value is the hash
	// of the envelope transaction of the bundle.
	CausedBy map[common.Hash]common.Hash
}

// ProcessedTransaction represents a transaction that was considered for
// inclusion in a block by the state processor. It contains the transaction
// itself and the receipt either confirming its execution, or nil if the
// transaction was skipped.
type ProcessedTransaction struct {
	Transaction *types.Transaction
	Receipt     *types.Receipt
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the StateDB, collecting receipts for applied
// transactions, nil-receipts for skipped transactions, and the used gas via an
// output parameter. The resulting list of receipts matches the order of the
// transactions in the block.
//
// A transaction is skipped if for some reason its execution in the given order
// is not possible. Skipped transactions do not consume any gas and do not affect
// the usedGas counter. The receipts for skipped transactions are nil. Processing
// continues with the next transaction in the block.
//
// Some reasons leading to issues during the execution of a transaction can lead
// to a general fail of the Process step. Among those are, for instance, the
// inability of restoring the sender from a transactions signature. In such a
// case, the corresponding transaction is skipped, but the processing of the
// block continues. The error is logged, but not returned to the caller.
//
// Note that these rules are part of the replicated state machine and must be
// consistent among all nodes on the network. The encoded rules have been
// inherited from the Fantom network and are active in the Sonic network.
// Future hard-forks may be used to clean up the rules and make them more
// consistent.
func (p *StateProcessor) Process(
	block *EvmBlock, statedb state.StateDB, cfg vm.Config, gasLimit uint64,
	usedGas *uint64, trueTxOffset int, onNewLog func(*core_types.Log), remainingSize uint64,
) ProcessSummary {
	sonicDifficulty := big.NewInt(1)
	return p.ProcessWithDifficulty(block, statedb, cfg, gasLimit, usedGas, trueTxOffset, onNewLog, sonicDifficulty, remainingSize)
}

// ProcessWithDifficulty is the same as Process, but allows specifying a custom
// difficulty for the block context. This is mainly intended for testing
// purposes, when processing blocks from non-Sonic networks exhibiting different
// difficulty values than Sonic's constant difficulty of 1.
func (p *StateProcessor) ProcessWithDifficulty(
	block *EvmBlock, statedb state.StateDB, cfg vm.Config, gasLimit uint64,
	usedGas *uint64, trueTxOffset int, onNewLog func(*core_types.Log),
	difficulty *big.Int, remainingSize uint64,
) ProcessSummary {
	var (
		gp           = core.NewGasPool(gasLimit)
		header       = block.Header()
		time         = uint64(block.Time.Unix())
		blockContext = NewEVMBlockContextWithDifficulty(header, p.bc, nil, difficulty)
		vmenv        = vm.NewEVM(blockContext, statedb, p.config, cfg)
		blockNumber  = block.Number
		signer       = types.LatestSignerForChainID(p.config.ChainID)
	)

	// execute EIP-2935 HistoryStorage contract.
	if p.config.IsPrague(blockNumber, time) {
		ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
	}

	// Iterate over and process the individual transactions
	return runTransactions(newRunContext(
		signer, header.BaseFee, statedb, gp, blockNumber, block.Time, usedGas,
		onNewLog, p.upgrades, &transactionRunner{evm{vmenv}}, p.forReplay, p.metrics,
	),
		block.Transactions, trueTxOffset, remainingSize)
}

// runContext bundles the parameters required for processing transactions in a
// block. It is used as input to the runTransactions helper function and passed
// along the processing layers to make the parameters available where needed.
type runContext struct {
	signer      types.Signer
	baseFee     *big.Int
	statedb     state.StateDB
	gasPool     *core.GasPool
	blockNumber *big.Int
	blockTime   inter.Timestamp
	usedGas     *uint64
	onNewLog    func(*core_types.Log)
	upgrades    opera.Upgrades
	runner      _transactionRunner
	forReplay   bool // Whether the context is used for replaying transactions or for head state processing.
	metrics     BlockExecutionMetrics
}

// newRunContext creates a new runContext instance bundling the given parameters
// required for processing transactions in a block. In productive code this
// function should be used instead of directly creating a runContext instance to
// ensure that all required parameters are provided.
func newRunContext(
	signer types.Signer,
	baseFee *big.Int,
	statedb state.StateDB,
	gasPool *core.GasPool,
	blockNumber *big.Int,
	blockTime inter.Timestamp,
	usedGas *uint64,
	onNewLog func(*core_types.Log),
	upgrades opera.Upgrades,
	runner _transactionRunner,
	isReplay bool,
	metrics BlockExecutionMetrics,
) *runContext {
	return &runContext{
		signer:      signer,
		baseFee:     baseFee,
		statedb:     statedb,
		gasPool:     gasPool,
		blockNumber: blockNumber,
		blockTime:   blockTime,
		usedGas:     usedGas,
		onNewLog:    onNewLog,
		upgrades:    upgrades,
		runner:      runner,
		forReplay:   isReplay,
		metrics:     metrics,
	}
}

// runTransaction is a helper function to process a list of transactions. It
// returns a list of ProcessedTransaction, containing the transaction and its
// receipt (or nil if the transaction was skipped).
//
// The function is intended to be used by both the Process function and the
// incremental transaction processor (BeginBlock/TransactionProcessor).
func runTransactions(
	context *runContext,
	transactions types.Transactions,
	trueTxIndexOffset int,
	remainingSize uint64,
) ProcessSummary {
	processedTxs := make([]ProcessedTransaction, 0, len(transactions))
	var execCosts core_types.ExecutionCost
	causedBy := make(map[common.Hash]common.Hash)
	for _, tx := range transactions {
		// Bundle-only transactions are only valid within a bundle and must
		// be rejected at the top level when processing the head state.
		if !context.forReplay && context.upgrades.Brio && bundle.IsBundleOnly(tx) {
			processedTxs = append(processedTxs, ProcessedTransaction{Transaction: tx})
			continue
		}

		txs, txResult, execCost := runTransaction(context, tx, trueTxIndexOffset, remainingSize)
		execCosts += execCost

		gasUsed := uint64(0)
		for _, processedTx := range txs {
			if processedTx.Receipt != nil { // < only transactions included in the block
				trueTxIndexOffset++
				causedBy[processedTx.Transaction.Hash()] = tx.Hash()

				if remainingSize < processedTx.Transaction.Size() {
					log.Debug("Block size limit exceeded,", "tx", processedTx.Transaction.Hash().Hex(),
						"txSize", processedTx.Transaction.Size(), "remainingSize", remainingSize)
					remainingSize = 0
				} else {
					remainingSize -= processedTx.Transaction.Size()
				}

				gasUsed += processedTx.Receipt.GasUsed
			}

			if context.upgrades.GasSubsidies && subsidies.IsSponsorshipRequest(processedTx.Transaction) && context.metrics != nil {
				if processedTx.Receipt == nil {
					context.metrics.IncSkippedSponsoredTx()
				} else {
					context.metrics.IncSponsoredTx()
				}
			}
		}
		processedTxs = append(processedTxs, txs...)

		if context.upgrades.Brio && bundle.IsEnvelope(tx) && context.metrics != nil {
			// update metrics for bundles
			switch txResult {
			case core_types.TransactionResultSuccessful:
				context.metrics.IncExecutedBundle()
			case core_types.TransactionResultFailed:
				context.metrics.IncRolledBackBundle()
			case core_types.TransactionResultInvalid:
				context.metrics.IncInvalidBundle()
			}

			// update efficiency histogram (guard against division by zero)
			context.metrics.ObserveBundleEfficiency(gasUsed, uint64(execCost))
		}
	}

	return ProcessSummary{
		ProcessedTransactions: processedTxs,
		ExecutionCost:         execCosts,
		CausedBy:              causedBy,
	}
}

// runTransaction processes the given transaction and returns a list of all
// processed transactions (transactions and receipts), and the result of
// processing the transaction. The only exception is for invalid bundles, where
// the envelope transaction itself is returned as a processed transaction, but
// without a receipt, to signal that the bundle transaction was skipped.
func runTransaction(
	context *runContext,
	tx *types.Transaction,
	trueTxIndexOffset int,
	sizeLimit uint64,
) ([]ProcessedTransaction, core_types.TransactionResult, core_types.ExecutionCost) {
	// Since a transaction bundle has a gas-price of 0 it would be considered a
	// sponsorship request. Thus, we need to check for bundles first.
	if context.upgrades.Brio && bundle.IsEnvelope(tx) {
		if context.upgrades.TransactionBundles {
			return context.runner.runTransactionBundle(context, tx, trueTxIndexOffset, sizeLimit)
		} else {
			return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
		}
	}
	if context.upgrades.GasSubsidies && subsidies.IsSponsorshipRequest(tx) {
		res, result := context.runner.runSponsoredTransaction(context, tx, trueTxIndexOffset, sizeLimit)
		execCost := core_types.ExecutionCost(0)
		for _, r := range res {
			if r.Receipt != nil {
				execCost += core_types.ExecutionCost(r.Receipt.GasUsed)
			}
		}
		return res, result, execCost
	} else {
		res, result := context.runner.runRegularTransaction(context, tx, trueTxIndexOffset, sizeLimit)
		execCost := core_types.ExecutionCost(0)
		if res.Receipt != nil {
			execCost = core_types.ExecutionCost(res.Receipt.GasUsed)
		}
		return []ProcessedTransaction{res}, result, execCost
	}
}

// _transactionRunner is an interface for components implementing the logic
// required for running transactions with various rules, e.g. regular or
// sponsored transactions.
type _transactionRunner interface {
	runRegularTransaction(
		ctxt *runContext,
		tx *types.Transaction,
		trueTxIndexOffset int,
		sizeLimit uint64,
	) (
		ProcessedTransaction,
		core_types.TransactionResult,
	)

	runSponsoredTransaction(
		ctxt *runContext,
		tx *types.Transaction,
		trueTxIndexOffset int,
		sizeLimit uint64,
	) (
		[]ProcessedTransaction,
		core_types.TransactionResult,
	)

	runTransactionBundle(
		ctxt *runContext,
		tx *types.Transaction,
		trueTxIndexOffset int,
		sizeLimit uint64,
	) (
		[]ProcessedTransaction,
		core_types.TransactionResult,
		core_types.ExecutionCost,
	)
}

// transactionRunner implements the _transactionRunner interface by using an
// _evm instance to run transactions.
type transactionRunner struct {
	evm _evm
}

func (r *transactionRunner) runRegularTransaction(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
	sizeLimit uint64,
) (ProcessedTransaction, core_types.TransactionResult) {
	if size := tx.Size(); size > sizeLimit {
		log.Debug("Transaction skipped due to block size limit", "tx", tx.Hash().Hex(), "txSize", size, "sizeLimit", sizeLimit)
		return ProcessedTransaction{Transaction: tx}, core_types.TransactionResultInvalid
	}

	res := r.evm.runWithBaseFeeCheck(ctxt, tx, txIndex)
	if res.Receipt != nil {
		if res.Receipt.Status == types.ReceiptStatusSuccessful {
			return res, core_types.TransactionResultSuccessful
		} else {
			return res, core_types.TransactionResultFailed
		}
	}
	return res, core_types.TransactionResultInvalid
}

func (r *transactionRunner) runSponsoredTransaction(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
	sizeLimit uint64,
) ([]ProcessedTransaction, core_types.TransactionResult) {
	return r.runSponsoredTransactionInternal(ctxt, tx, txIndex, sizeLimit, subsidies.Sponsorship.GetPostTransactions)
}

func (r *transactionRunner) runSponsoredTransactionInternal(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
	sizeLimit uint64,
	getPostTransactions func(subsidies.Sponsorship, subsidies.NonceSource, uint64, *big.Int) ([]*types.Transaction, error),
) ([]ProcessedTransaction, core_types.TransactionResult) {
	// Run the IsCovered query in a snapshot to avoid spilling any side-effects
	// like warm storage slots or refunds into the actual transaction.
	snapshot := ctxt.statedb.Snapshot()
	sponsorship, err := subsidies.IsCovered(
		ctxt.upgrades, r.evm, ctxt.signer, tx, ctxt.baseFee,
	)
	ctxt.statedb.RevertToSnapshot(snapshot)
	if err != nil {
		log.Warn("Failed to query subsidies registry", "tx", tx.Hash().Hex(), "err", err)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid
	}
	if !sponsorship.IsSponsored() {
		log.Debug("Transaction is not covered by a subsidy", "tx", tx.Hash().Hex())
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid
	}

	overhead := sponsorship.Overhead()
	if tx.Size()+overhead.Size > sizeLimit {
		log.Debug("Transaction skipped due to block size limit",
			"tx", tx.Hash().Hex(), "txSize", tx.Size(), "estimatedPaymentTxSize", overhead.Size, "sizeLimit", sizeLimit)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid
	}

	available := ctxt.gasPool.Gas()
	needed := tx.Gas() + overhead.Gas
	if available < needed {
		log.Debug("Not enough gas left in block for sponsored transaction",
			"tx", tx.Hash().Hex(), "available", available, "needed", needed,
		)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid
	}

	// Run the sponsored transaction.
	processed := r.evm.runWithoutBaseFeeCheck(ctxt, tx, txIndex)
	if processed.Receipt == nil {
		log.Debug("Sponsored transaction skipped", "tx", tx.Hash().Hex())
		return []ProcessedTransaction{processed}, core_types.TransactionResultInvalid
	}

	status := core_types.TransactionResultSuccessful
	if processed.Receipt.Status == types.ReceiptStatusFailed {
		log.Debug("Sponsored transaction failed", "tx", tx.Hash().Hex())
		status = core_types.TransactionResultFailed
	}

	gasUsed := processed.Receipt.GasUsed

	// Skip post-execution transactions when replaying.
	if ctxt.forReplay {
		return []ProcessedTransaction{processed}, status
	}

	postTxs, err := getPostTransactions(sponsorship, ctxt.statedb, gasUsed, ctxt.baseFee)
	if err != nil {
		// Note: at this point the sponsored transaction has been executed, but we
		// are not able to build the post-execution transaction. We can not undo
		// the sponsored transaction, and we can not abort the block formation.
		log.Warn("Failed to create post-execution transaction", "sponsored-tx", tx.Hash().Hex(), "err", err)
		return []ProcessedTransaction{processed}, status
	}

	txIndex++
	out := []ProcessedTransaction{processed}
	for _, postTx := range postTxs {
		processedPost := r.evm.runWithoutBaseFeeCheck(ctxt, postTx, txIndex)
		if processedPost.Receipt == nil {
			log.Warn("Post-execution transaction was skipped", "sponsored-tx", tx.Hash().Hex())
		} else if processedPost.Receipt.Status == types.ReceiptStatusFailed {
			log.Warn("Post-execution transaction failed", "sponsored-tx", tx.Hash().Hex())
		}
		out = append(out, processedPost)
		if processedPost.Receipt != nil {
			txIndex++
		}
	}
	return out, status
}

// runTransactionBundle processes the bundle-only transactions in the given
// envelope transaction as a bundle. It returns the list of processed
// transactions (transactions and receipts), and the result of processing the
// bundle transaction. If the bundle is invalid, the envelope transaction itself
// is returned as a single processed transaction without a receipt. This is
// needed to signal skipped bundles.
func (r *transactionRunner) runTransactionBundle(
	ctxt *runContext,
	tx *types.Transaction,
	trueTxIndexOffset int,
	sizeLimit uint64,
) ([]ProcessedTransaction, core_types.TransactionResult, core_types.ExecutionCost) {
	return r.runTransactionBundleInternal(ctxt, tx, trueTxIndexOffset, log.Root(), sizeLimit)
}

func (r *transactionRunner) runTransactionBundleInternal(
	ctxt *runContext,
	tx *types.Transaction,
	trueTxOffset int,
	log logger,
	sizeLimit uint64,
) ([]ProcessedTransaction, core_types.TransactionResult, core_types.ExecutionCost) {
	if !ctxt.upgrades.TransactionBundles {
		log.Warn("Transaction bundles are not enabled, bundle transaction skipped", "tx", tx.Hash().Hex())
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
	}

	txBundle, plan, err := bundle.ValidateEnvelope(ctxt.signer, tx)
	if err != nil {
		log.Warn("Invalid bundle skipped", "tx", tx.Hash().Hex(), "err", err)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
	}

	if !plan.Range.IsInRange(ctxt.blockNumber.Uint64()) {
		log.Warn("Bundle skipped due to out-of-range execution plan", "tx", tx.Hash().Hex(), "planRange", plan.Range, "blockNumber", ctxt.blockNumber.Uint64())
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
	}

	if !plan.Period.IsInPeriod(ctxt.blockTime) {
		log.Warn("Bundle skipped due to out-of-time execution plan", "tx", tx.Hash().Hex(), "planPeriod", plan.Period, "blockTime", ctxt.blockTime)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
	}

	planHash := plan.Hash()
	if ctxt.statedb.HasBundleRecentlyBeenProcessed(planHash) {
		log.Warn("Rescheduled bundle skipped", "exec_plan_hash", planHash)
		return []ProcessedTransaction{{Transaction: tx}}, core_types.TransactionResultInvalid, 0
	}

	positionInBlock := bundle.PositionInBlock{
		Offset: uint32(trueTxOffset),
	}

	// Run the bundle and collect the processed transactions.
	runner := bundleTransactionRunner{
		ctxt:         ctxt,
		trueTxOffset: trueTxOffset,
		sizeLimit:    sizeLimit,
	}
	if !bundle.RunBundle(txBundle, &runner) {
		// Mark the execution plan as processed in the StateDB to prevent processing
		// another bundle with the same execution plan in the same block. Also keep
		// track of the position of the bundle in the block.
		// Note: it is sufficient to mark the execution plan of a bundle after the
		// execution of the bundle as used since nested bundles can not contain
		// copies of themselves without finding a hash-function collision.
		ctxt.statedb.AddProcessedBundle(planHash, positionInBlock)
		return []ProcessedTransaction{}, core_types.TransactionResultFailed, runner.executionCost
	}

	// Update the position-in-block struct to track the number of transactions
	// added to the block as part of this bundle execution.
	for _, processedTx := range runner.processedTransactions {
		if processedTx.Receipt != nil {
			positionInBlock.Count++
		}
	}

	// Mark the execution plan as processed in the StateDB to prevent processing
	// another bundle with the same execution plan in the same block. Also keep
	// track of the position of the bundle in the block.
	// Note: it is sufficient to mark the execution plan of a bundle after the
	// execution of the bundle as used since nested bundles can not contain
	// copies of themselves without finding a hash-function collision.
	ctxt.statedb.AddProcessedBundle(planHash, positionInBlock)

	return runner.processedTransactions, core_types.TransactionResultSuccessful, runner.executionCost
}

// bundleTransactionRunner is an adapter implementing the bundle.TransactionRunner
// interface to run transactions within a bundle and collect their results.
type bundleTransactionRunner struct {
	ctxt                  *runContext
	trueTxOffset          int
	processedTransactions []ProcessedTransaction
	snapshots             []bundleTransactionRunnerSnapshot
	executionCost         core_types.ExecutionCost // not included in snapshots
	sizeLimit             uint64
}

func (b *bundleTransactionRunner) Run(tx *types.Transaction) core_types.TransactionResult {

	snapshot := b.CreateSnapshot()

	processed, result, execCost := runTransaction(b.ctxt, tx, b.trueTxOffset, b.sizeLimit)
	b.executionCost += execCost
	b.processedTransactions = append(b.processedTransactions, processed...)

	for _, p := range processed {
		if p.Receipt != nil {
			b.trueTxOffset++
			if b.sizeLimit < p.Transaction.Size() {
				b.RevertToSnapshot(snapshot)
				return core_types.TransactionResultInvalid
			}
			b.sizeLimit -= p.Transaction.Size()
		}
	}

	return result
}

func (b *bundleTransactionRunner) CreateSnapshot() int {
	snapshot := bundleTransactionRunnerSnapshot{
		stateDbSnapshot:                b.ctxt.statedb.InterTxSnapshot(),
		trueTxOffset:                   b.trueTxOffset,
		processedTransactionListLength: len(b.processedTransactions),
		usedGas:                        *b.ctxt.usedGas,
		gasPool:                        b.ctxt.gasPool.Snapshot(),
		sizeLimit:                      b.sizeLimit,
	}
	b.snapshots = append(b.snapshots, snapshot)
	return len(b.snapshots) - 1
}

func (b *bundleTransactionRunner) RevertToSnapshot(id int) {
	if id < 0 || id >= len(b.snapshots) {
		// trigger an invalid revert in the StateDB to keep track of the invalid
		// revert, such that it can be handled when checking for errors in the
		// StateDB after the block processing.
		b.ctxt.statedb.RevertToInterTxSnapshot(state.InvalidSnapshotID)
		return
	}
	snapshot := b.snapshots[id]
	b.ctxt.statedb.RevertToInterTxSnapshot(snapshot.stateDbSnapshot)
	b.trueTxOffset = snapshot.trueTxOffset
	b.processedTransactions = b.processedTransactions[:snapshot.processedTransactionListLength]
	*b.ctxt.usedGas = snapshot.usedGas
	b.ctxt.gasPool.Set(snapshot.gasPool)
	b.sizeLimit = snapshot.sizeLimit
	b.snapshots = b.snapshots[:id]
}

type bundleTransactionRunnerSnapshot struct {
	stateDbSnapshot                int
	trueTxOffset                   int
	processedTransactionListLength int
	usedGas                        uint64
	gasPool                        *core.GasPool
	sizeLimit                      uint64
}

// _evm is an interface to an EVM instance that can be used to run a single
// transaction. It is used by the transactionRunner to decouple the transaction
// running logic from the actual EVM implementation, enabling easier testing.
type _evm interface {
	subsidies.VirtualMachine
	runWithBaseFeeCheck(*runContext, *types.Transaction, int) ProcessedTransaction
	runWithoutBaseFeeCheck(*runContext, *types.Transaction, int) ProcessedTransaction
}

type evm struct {
	*vm.EVM
}

func (e evm) runWithBaseFeeCheck(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
) ProcessedTransaction {
	return e._runTransaction(ctxt, tx, txIndex, true)
}

func (e evm) runWithoutBaseFeeCheck(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
) ProcessedTransaction {
	return e._runTransaction(ctxt, tx, txIndex, false)
}

func (e evm) _runTransaction(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
	checkBaseFee bool,
) ProcessedTransaction {
	msg, err := TxAsMessage(tx, ctxt.signer, ctxt.baseFee)
	if err != nil {
		log.Info("Failed to convert transaction to message", "tx", tx.Hash().Hex(), "err", err)
		return ProcessedTransaction{Transaction: tx}
	}

	e.Config.NoBaseFee = !checkBaseFee
	ctxt.statedb.SetTxContext(tx.Hash(), txIndex)
	receipt, _, err := applyTransaction(
		msg, ctxt.gasPool, ctxt.statedb, ctxt.blockNumber, tx,
		ctxt.usedGas, e.EVM, ctxt.onNewLog,
	)
	if err != nil {
		log.Debug("Failed to apply transaction", "tx", tx.Hash().Hex(), "err", err)
		return ProcessedTransaction{Transaction: tx}
	}
	return ProcessedTransaction{Transaction: tx, Receipt: receipt}
}

// ---

// ChainState provides access to the chain state retained by the client required
// for test-running transactions.
type ChainState interface {
	// DummyChain needs to be implemented in order to resolve past block hashes.
	// TODO: follow-up task - simplify this to a GetBlockHash(idx.Block) method.
	DummyChain

	// GetCurrentNetworkRules returns the current network rules for the EVM.
	GetCurrentNetworkRules() opera.Rules

	// GetCurrentChainConfig returns the current chain configuration, which
	// is needed to determine the active EVM rules and the block context
	// for transaction processing.
	GetCurrentChainConfig() *params.ChainConfig
}

// NewTransactionProcessorForBlock creates a new transaction processor to be used
// for trial-running transaction sequences (e.g. in the emitter) in the context
// of the given block.
func NewTransactionProcessorForBlock(
	chain ChainState,
	state state.StateDB,
	block *EvmBlock,
) *TransactionProcessor {
	// TODO: follow-up task - align this with c_block_callbacks.go
	// see https://github.com/0xsoniclabs/sonic-admin/issues/227
	rules := chain.GetCurrentNetworkRules()
	chainCfg := chain.GetCurrentChainConfig()
	vmConfig := opera.GetVmConfig(rules)

	// The gas limit for transactions is enforced on a per-transaction level
	// in the scheduler. See the scheduler.Schedule method for details. The
	// total gas used for attempting to schedule transactions is not limited.
	gasLimit := uint64(math.MaxUint64)
	stateProcessor := NewStateProcessorForHeadState(
		chainCfg,
		chain,
		rules.Upgrades,
		nil,
	)
	return stateProcessor.BeginBlock(block, state, vmConfig, gasLimit, nil)
}

// BeginBlock starts the processing of a new block and returns a function to
// process individual transactions in the block. It follows the same rules as
// the Process method, yet enables the incremental processing of transactions.
// This is required by the transaction scheduler in the emitter, which needs to
// probe individual transactions to determine their applicability and gas usage.
func (p *StateProcessor) BeginBlock(
	block *EvmBlock, stateDb state.StateDB, cfg vm.Config, gasLimit uint64,
	onNewLog func(*core_types.Log),
) *TransactionProcessor {
	var (
		gp            = core.NewGasPool(gasLimit)
		header        = block.Header()
		time          = uint64(block.Time.Unix())
		blockContext  = NewEVMBlockContext(header, p.bc, nil)
		vmEnvironment = vm.NewEVM(blockContext, stateDb, p.config, cfg)
		blockNumber   = block.Number
		signer        = types.LatestSignerForChainID(p.config.ChainID)
	)

	// execute EIP-2935 HistoryStorage contract.
	if p.config.IsPrague(blockNumber, time) {
		ProcessParentBlockHash(block.ParentHash, vmEnvironment, stateDb)
	}

	return &TransactionProcessor{
		blockNumber:   blockNumber,
		blockTime:     block.Time,
		gp:            gp,
		header:        header,
		onNewLog:      onNewLog,
		signer:        signer,
		stateDb:       stateDb,
		vmEnvironment: vmEnvironment,
		upgrades:      p.upgrades,
		metrics:       p.metrics,
	}
}

// TransactionProcessor is produced by the BeginBlock function and is used to
// process individual transactions in the block.
type TransactionProcessor struct {
	blockNumber   *big.Int
	blockTime     inter.Timestamp
	gp            *core.GasPool
	header        *EvmHeader
	onNewLog      func(*core_types.Log)
	signer        types.Signer
	stateDb       state.StateDB
	usedGas       uint64
	vmEnvironment *vm.EVM
	upgrades      opera.Upgrades
	metrics       BlockExecutionMetrics
}

// Run processes a single transaction in the block, where i is the index of
// the transaction in the block. It returns the list of all transactions that
// have been attempted to be processed to cover the given transaction as well as
// their receipts if they did not get skipped.
func (tp *TransactionProcessor) Run(i int, tx *types.Transaction) ProcessSummary {
	return runTransactions(newRunContext(
		tp.signer, tp.header.BaseFee, tp.stateDb, tp.gp, tp.blockNumber, tp.blockTime,
		&tp.usedGas, tp.onNewLog, tp.upgrades, &transactionRunner{evm{tp.vmEnvironment}},
		false, tp.metrics,
	), []*types.Transaction{tx}, i, math.MaxUint64)
}

// ApplyTransactionWithEVM attempts to apply a transaction to the given state database
// and uses the input parameters for its environment similar to ApplyTransaction. However,
// this method takes an already created EVM instance as input.
//
// The primary use case for this method is to apply transactions from RPC calls or
// traces.
func ApplyTransactionWithEVM(
	msg *core.Message,
	config *params.ChainConfig,
	gp *core.GasPool,
	statedb state.StateDB,
	blockNumber *big.Int,
	blockHash common.Hash,
	tx *types.Transaction,
	usedGas *uint64,
	evm *vm.EVM,
) (receipt *types.Receipt, err error) {
	if evm.Config.Tracer != nil && evm.Config.Tracer.OnTxStart != nil {
		evm.Config.Tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
		if evm.Config.Tracer.OnTxEnd != nil {
			defer func() {
				evm.Config.Tracer.OnTxEnd(receipt, err)
			}()
		}
	}
	// Create a new context to be used in the EVM environment.
	txContext, err := NewEVMTxContext(msg)
	if err != nil {
		statedb.EndTransaction()
		return nil, fmt.Errorf("failed to create EVM transaction context: %w", err)
	}
	evm.SetTxContext(txContext)

	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		statedb.EndTransaction()
		return nil, err
	}

	// Update the state with pending changes.
	statedb.EndTransaction()
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt = &types.Receipt{Type: tx.Type(), CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	if tx.Type() == types.BlobTxType {
		receipt.BlobGasUsed = uint64(len(tx.BlobHashes()) * params.BlobTxBlobGasPerBlob)
		receipt.BlobGasPrice = evm.Context.BlobBaseFee // TODO issue #147
	}

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.Origin, tx.Nonce())
	}

	// Tracing doesn't need logs and bloom.
	if evm.Config.Tracer == nil {
		// Set the receipt logs and create the bloom filter.
		receipt.Logs = statedb.GetLogs(tx.Hash(), blockHash) // don't store logs when tracing
		receipt.Bloom = types.CreateBloom(receipt)
	}
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

// ProcessParentBlockHash stores the parent block hash in the history storage contract
// as per EIP-2935.
func ProcessParentBlockHash(prevHash common.Hash, evm *vm.EVM, stateDb state.StateDB) {
	msg := &core.Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &params.HistoryStorageAddress,
		Data:      prevHash.Bytes(),
	}

	// Must is fine, since all inputs are practically hardcoded above.
	evm.SetTxContext(MustNewEVMTxContext(msg))

	stateDb.AddAddressToAccessList(params.HistoryStorageAddress)
	_, _, _ = evm.Call(msg.From, *msg.To, msg.Data, 30_000_000, common.U2560)
	stateDb.Finalise(true)
	stateDb.EndTransaction()
}

// applyTransaction attempts to apply a transaction defined by the given message
// to the provided EVM environment. If successful, a non-nil receipt and the
// used gas is returned. If it fails, an error is returned and the receipt is
// guaranteed to be nil.
func applyTransaction(
	msg *core.Message,
	gp *core.GasPool,
	statedb state.StateDB,
	blockNumber *big.Int,
	tx *types.Transaction,
	usedGas *uint64,
	evm *vm.EVM,
	onNewLog func(*core_types.Log),
) (
	*types.Receipt,
	uint64,
	error,
) {
	// Create a new context to be used in the EVM environment.
	txContext, err := NewEVMTxContext(msg)
	if err != nil {
		statedb.EndTransaction()
		return nil, 0, fmt.Errorf("failed to create EVM transaction context: %w", err)
	}
	evm.SetTxContext(txContext)

	// Skip checking of base fee limits for internal transactions.
	evm.Config.NoBaseFee = evm.Config.NoBaseFee || msg.SkipNonceChecks

	isAllegro := evm.ChainConfig().IsPrague(blockNumber, evm.Context.Time)
	var snapshot int
	if isAllegro {
		snapshot = statedb.Snapshot()
	}
	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		if isAllegro {
			statedb.RevertToSnapshot(snapshot)
		}
		statedb.EndTransaction()
		return nil, 0, err
	}
	// Notify about logs with potential state changes.
	// At this point the final block hash is not yet known, so we pass an empty
	// hash. For the consumers of the log messages, as for instance the driver
	// contract listener, only the sender, topics, and the data are relevant.
	// The block hash is not used.
	logs := statedb.GetLogs(tx.Hash(), common.Hash{})
	if onNewLog != nil {
		for _, l := range logs {
			onNewLog(core_types.CoreLogFromGethLog(l))
		}
	}

	// Update the state with pending changes.
	statedb.EndTransaction()
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.Origin, tx.Nonce())
	}

	// Set the receipt logs.
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(receipt)
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())

	// Set the effective gas price in the receipt. By registering it here, at
	// the source, down-stream consumers of the receipts do not have to
	// replicate the code for computing effective gas prices.
	receipt.EffectiveGasPrice = msg.GasPrice

	return receipt, result.UsedGas, nil
}

func TxAsMessage(tx *types.Transaction, signer types.Signer, baseFee *big.Int) (*core.Message, error) {
	if internaltx.IsInternal(tx) {

		return &core.Message{ // internal tx - no signature checking
			From:                  internaltx.InternalSender(tx),
			To:                    tx.To(),
			Nonce:                 tx.Nonce(),
			Value:                 tx.Value(),
			GasLimit:              tx.Gas(),
			GasPrice:              tx.GasPrice(),
			GasFeeCap:             tx.GasFeeCap(),
			GasTipCap:             tx.GasTipCap(),
			Data:                  tx.Data(),
			AccessList:            tx.AccessList(),
			BlobGasFeeCap:         tx.BlobGasFeeCap(),
			BlobHashes:            tx.BlobHashes(),
			SkipNonceChecks:       true, // don't check sender nonce and being EOA
			SkipTransactionChecks: true,
		}, nil
	}

	msg, err := core.TransactionToMessage(tx, signer, baseFee)
	if err != nil {
		return nil, err
	}

	// Patch BlobHashes to allow execution of ethereum history and sonic semantics:
	// - ethereum does not allow blob txs without BlobHashes, but preChecks will not
	// check it if BlobHashes is nil
	// - Sonic only allows blob txs if BlobHashes is empty
	if len(msg.BlobHashes) == 0 {
		msg.BlobHashes = nil
	}

	return msg, nil
}

// logger is an internal interface to enable the mocking of logging in tests.
// This is in particular useful to make sure tests that trigger failing
// conditions are actually triggering the correct condition.
type logger interface {
	Debug(msg string, ctx ...any)
	Info(msg string, ctx ...any)
	Warn(msg string, ctx ...any)
}
