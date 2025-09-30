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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/signers/gsignercache"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

//go:generate mockgen -source=state_processor.go -destination=state_processor_mock.go -package=evmcore

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config   *params.ChainConfig // Chain configuration options
	bc       DummyChain          // Canonical block chain
	upgrades opera.Upgrades      // Enabled network upgrades
}

// NewStateProcessor initializes a new StateProcessor.
func NewStateProcessor(
	config *params.ChainConfig,
	bc DummyChain,
	upgrades opera.Upgrades,
) *StateProcessor {
	return &StateProcessor{
		config:   config,
		bc:       bc,
		upgrades: upgrades,
	}
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
	usedGas *uint64, onNewLog func(*types.Log),
) []ProcessedTransaction {
	var (
		gp           = new(core.GasPool).AddGas(gasLimit)
		header       = block.Header()
		time         = uint64(block.Time.Unix())
		blockContext = NewEVMBlockContext(header, p.bc, nil)
		vmenv        = vm.NewEVM(blockContext, statedb, p.config, cfg)
		blockNumber  = block.Number
		signer       = gsignercache.Wrap(types.MakeSigner(p.config, header.Number, time))
	)

	// execute EIP-2935 HistoryStorage contract.
	if p.config.IsPrague(blockNumber, time) {
		ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
	}

	// Iterate over and process the individual transactions
	return runTransactions(newRunContext(
		signer, header.BaseFee, statedb, gp, blockNumber, usedGas,
		onNewLog, p.upgrades, &transactionRunner{evm{vmenv}},
	), block.Transactions, 0)
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
	usedGas     *uint64
	onNewLog    func(*types.Log)
	upgrades    opera.Upgrades
	runner      _transactionRunner
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
	usedGas *uint64,
	onNewLog func(*types.Log),
	upgrades opera.Upgrades,
	runner _transactionRunner,
) *runContext {
	return &runContext{
		signer:      signer,
		baseFee:     baseFee,
		statedb:     statedb,
		gasPool:     gasPool,
		blockNumber: blockNumber,
		usedGas:     usedGas,
		onNewLog:    onNewLog,
		upgrades:    upgrades,
		runner:      runner,
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
	txIndexOffset int,
) []ProcessedTransaction {
	processed := make([]ProcessedTransaction, 0, len(transactions))
	for _, tx := range transactions {
		nextId := len(processed) + txIndexOffset
		if context.upgrades.GasSubsidies && subsidies.IsSponsorshipRequest(tx) {
			processed = append(processed,
				context.runner.runSponsoredTransaction(context, tx, nextId)...,
			)
		} else {
			processed = append(processed,
				context.runner.runRegularTransaction(context, tx, nextId),
			)
		}
	}
	return processed
}

// _transactionRunner is an interface for components implementing the logic
// required for running transactions with various rules, e.g. regular or
// sponsored transactions.
type _transactionRunner interface {
	runRegularTransaction(ctxt *runContext, tx *types.Transaction, txIndex int) ProcessedTransaction
	runSponsoredTransaction(ctxt *runContext, tx *types.Transaction, txIndex int) []ProcessedTransaction
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
) ProcessedTransaction {
	return r.evm.runWithBaseFeeCheck(ctxt, tx, txIndex)
}

func (r *transactionRunner) runSponsoredTransaction(
	ctxt *runContext,
	tx *types.Transaction,
	txIndex int,
) []ProcessedTransaction {
	// Check the remaining available gas to be used in this block.
	available := ctxt.gasPool.Gas()
	needed := tx.Gas() + subsidies.SponsorshipOverheadGasCost
	if available < needed {
		log.Debug("Not enough gas left in block for sponsored transaction",
			"tx", tx.Hash().Hex(), "available", available, "needed", needed,
		)
		return []ProcessedTransaction{{Transaction: tx}}
	}

	// Run the IsCovered query in a snapshot to avoid spilling any side-effects
	// like warm storage slots or refunds into the actual transaction.
	snapshot := ctxt.statedb.Snapshot()
	covered, fundId, err := subsidies.IsCovered(
		ctxt.upgrades, r.evm, ctxt.signer, tx, ctxt.baseFee,
	)
	ctxt.statedb.RevertToSnapshot(snapshot)
	if err != nil {
		log.Warn("Failed to query subsidies registry", "tx", tx.Hash().Hex(), "err", err)
		return []ProcessedTransaction{{Transaction: tx}}
	}
	if !covered {
		log.Debug("Transaction is not covered by a subsidy", "tx", tx.Hash().Hex())
		return []ProcessedTransaction{{Transaction: tx}}
	}

	// Run the sponsored transaction.
	processed := r.evm.runWithoutBaseFeeCheck(ctxt, tx, txIndex)
	if processed.Receipt == nil {
		log.Debug("Sponsored transaction skipped", "tx", tx.Hash().Hex())
		return []ProcessedTransaction{processed}
	}

	// Charge the fee for the sponsored transaction to the subsidy fund.
	gasUsed := processed.Receipt.GasUsed
	feeChargingTx, err := subsidies.GetFeeChargeTransaction(
		ctxt.statedb, fundId, gasUsed, ctxt.baseFee,
	)
	if err != nil {
		// Note: at this point the sponsored transaction has been executed, but
		// we are not able to charge the fee to the subsidy fund. At this point
		// we can not undo the sponsored transaction, and we can not abort the
		// block formation. So we have to let this go. This sponsored
		// transaction was on the house (meaning on the network).
		log.Warn("Failed to create fee charging transaction", "sponsored-tx", tx.Hash().Hex(), "err", err)
		return []ProcessedTransaction{processed}
	}
	processedDeduction := r.evm.runWithoutBaseFeeCheck(ctxt, feeChargingTx, txIndex+1)
	if processedDeduction.Receipt == nil {
		// Note: at this point, the deduction transaction was skipped, meaning
		// the subsidy fund was not charged. We can not abort the block
		// formation, so we have to let this go.
		log.Warn("Fee charging transaction was skipped", "sponsored-tx", tx.Hash().Hex())
	}
	if processedDeduction.Receipt != nil && processedDeduction.Receipt.Status == types.ReceiptStatusFailed {
		// Note: at this point, the deduction transaction failed, meaning the
		// subsidy fund was not charged.
		log.Warn("Fee charging transaction failed", "sponsored-tx", tx.Hash().Hex())
	}
	return []ProcessedTransaction{processed, processedDeduction}
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

// BeginBlock starts the processing of a new block and returns a function to
// process individual transactions in the block. It follows the same rules as
// the Process method, yet enables the incremental processing of transactions.
// This is required by the transaction scheduler in the emitter, which needs to
// probe individual transactions to determine their applicability and gas usage.
func (p *StateProcessor) BeginBlock(
	block *EvmBlock, stateDb state.StateDB, cfg vm.Config, gasLimit uint64,
	onNewLog func(*types.Log),
) *TransactionProcessor {
	var (
		gp            = new(core.GasPool).AddGas(gasLimit)
		header        = block.Header()
		time          = uint64(block.Time.Unix())
		blockContext  = NewEVMBlockContext(header, p.bc, nil)
		vmEnvironment = vm.NewEVM(blockContext, stateDb, p.config, cfg)
		blockNumber   = block.Number
		signer        = gsignercache.Wrap(types.MakeSigner(p.config, header.Number, time))
	)

	// execute EIP-2935 HistoryStorage contract.
	if p.config.IsPrague(blockNumber, time) {
		ProcessParentBlockHash(block.ParentHash, vmEnvironment, stateDb)
	}

	return &TransactionProcessor{
		blockNumber:   blockNumber,
		gp:            gp,
		header:        header,
		onNewLog:      onNewLog,
		signer:        signer,
		stateDb:       stateDb,
		vmEnvironment: vmEnvironment,
		upgrades:      p.upgrades,
	}
}

// TransactionProcessor is produced by the BeginBlock function and is used to
// process individual transactions in the block.
type TransactionProcessor struct {
	blockNumber   *big.Int
	gp            *core.GasPool
	header        *EvmHeader
	onNewLog      func(*types.Log)
	signer        types.Signer
	stateDb       state.StateDB
	usedGas       uint64
	vmEnvironment *vm.EVM
	upgrades      opera.Upgrades
}

// Run processes a single transaction in the block, where i is the index of
// the transaction in the block. It returns the list of all transactions that
// have been attempted to be processed to cover the given transaction as well as
// their receipts if they did not get skipped.
func (tp *TransactionProcessor) Run(i int, tx *types.Transaction) []ProcessedTransaction {
	return runTransactions(newRunContext(
		tp.signer, tp.header.BaseFee, tp.stateDb, tp.gp, tp.blockNumber,
		&tp.usedGas, tp.onNewLog, tp.upgrades, &transactionRunner{evm{tp.vmEnvironment}},
	), []*types.Transaction{tx}, i)
}

// ApplyTransactionWithEVM attempts to apply a transaction to the given state database
// and uses the input parameters for its environment similar to ApplyTransaction. However,
// this method takes an already created EVM instance as input.
func ApplyTransactionWithEVM(msg *core.Message, config *params.ChainConfig, gp *core.GasPool, statedb state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (receipt *types.Receipt, err error) {
	if evm.Config.Tracer != nil && evm.Config.Tracer.OnTxStart != nil {
		evm.Config.Tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
		if evm.Config.Tracer.OnTxEnd != nil {
			defer func() {
				evm.Config.Tracer.OnTxEnd(receipt, err)
			}()
		}
	}
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.SetTxContext(txContext)

	// For now, Sonic only supports Blob transactions without blob data.
	if msg.BlobHashes != nil {
		if len(msg.BlobHashes) > 0 {
			statedb.EndTransaction()
			return nil, fmt.Errorf("blob data is not supported")
		}
		// PreCheck requires non-nil blobHashes not to be empty
		msg.BlobHashes = nil
	}

	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
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

	txContext := NewEVMTxContext(msg)
	evm.SetTxContext(txContext)

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
	onNewLog func(*types.Log),
) (
	*types.Receipt,
	uint64,
	error,
) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.SetTxContext(txContext)

	// Skip checking of base fee limits for internal transactions.
	evm.Config.NoBaseFee = evm.Config.NoBaseFee || msg.SkipNonceChecks

	// For now, Sonic only supports Blob transactions without blob data.
	if msg.BlobHashes != nil {
		if len(msg.BlobHashes) > 0 {
			statedb.EndTransaction()
			return nil, 0, fmt.Errorf("blob data is not supported")
		}
		// PreCheck requires non-nil blobHashes not to be empty
		msg.BlobHashes = nil
	}

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
			onNewLog(l)
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
	return receipt, result.UsedGas, nil
}

func TxAsMessage(tx *types.Transaction, signer types.Signer, baseFee *big.Int) (*core.Message, error) {
	if !internaltx.IsInternal(tx) {
		return core.TransactionToMessage(tx, signer, baseFee)
	} else {
		return &core.Message{ // internal tx - no signature checking
			From:             internaltx.InternalSender(tx),
			To:               tx.To(),
			Nonce:            tx.Nonce(),
			Value:            tx.Value(),
			GasLimit:         tx.Gas(),
			GasPrice:         tx.GasPrice(),
			GasFeeCap:        tx.GasFeeCap(),
			GasTipCap:        tx.GasTipCap(),
			Data:             tx.Data(),
			AccessList:       tx.AccessList(),
			BlobGasFeeCap:    tx.BlobGasFeeCap(),
			BlobHashes:       tx.BlobHashes(),
			SkipNonceChecks:  true, // don't check sender nonce and being EOA
			SkipFromEOACheck: true,
		}, nil
	}
}
