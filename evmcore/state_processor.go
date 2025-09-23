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

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/utils/signers/gsignercache"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     DummyChain          // Canonical block chain
}

// ProcessedTransaction bundles a transaction with its receipt. It is produced
// by the StateProcessor's Process function for each non-skipped transaction.
// When produced by the Process function, neither field is nil.
type ProcessedTransaction struct {
	Transaction *types.Transaction
	Receipt     *types.Receipt
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc DummyChain) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the given block's transactions using the StateDB. The list of all executed
// transactions and their receipts is collected. The total used gas is reported
// via an output parameter.
//
// A transaction is skipped if for some reason its execution in the given order
// is not possible. Skipped transactions do not consume any gas and do not affect
// the usedGas counter. Processing continues with the next transaction in the
// block.
//
// Note that these rules are part of the replicated state machine and must be
// consistent among all nodes on the network. The encoded rules have been
// inherited from the Fantom network and are active in the Sonic network.
// Future hard-forks may be used to clean up the rules and make them more
// consistent.
func (p *StateProcessor) Process(
	block *EvmBlock, statedb state.StateDB, cfg vm.Config, gasLimit uint64,
	usedGas *uint64, onNewLog func(*types.Log),
) (
	[]ProcessedTransaction, int,
) {
	result := make([]ProcessedTransaction, 0, 2*len(block.Transactions))
	var (
		gp           = new(core.GasPool).AddGas(gasLimit)
		header       = block.Header()
		time         = uint64(block.Time.Unix())
		blockContext = NewEVMBlockContext(header, p.bc, nil)
		vmenv        = vm.NewEVM(blockContext, statedb, p.config, cfg)
		blockNumber  = block.Number
		signer       = gsignercache.Wrap(types.MakeSigner(p.config, header.Number, time))
		numSkipped   = 0
	)

	// execute EIP-2935 HistoryStorage contract.
	if p.config.IsPrague(blockNumber, time) {
		ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
	}

	// Iterate over and process the individual transactions
	txCounter := 0
	for _, tx := range block.Transactions {
		counterBefore := txCounter
		processed, err := runTransaction(tx, signer, header.BaseFee,
			func(msg *core.Message, tx *types.Transaction) (*types.Receipt, error) {
				statedb.SetTxContext(tx.Hash(), txCounter)
				txCounter++
				receipt, _, err := applyTransaction(
					msg, gp, statedb, blockNumber, tx,
					usedGas, vmenv, onNewLog,
				)
				return receipt, err
			},
		)
		if err != nil {
			log.Warn("Error processing transaction", "tx", tx.Hash().Hex(), "err", err)
		}

		if len(processed) == 0 {
			numSkipped++
		} else {
			result = append(result, processed...)
		}

		// Ensure that even if the current transaction was skipped for any
		// reason, the txCounter is still incremented. This means that the
		// transaction index in the receipts might not match the index of
		// the transaction in the block. This might actually be an issue in the
		// block processing semantic, but to remain compatible with the existing
		// behavior, we need to preserve this until we can safely change it.
		if txCounter == counterBefore {
			txCounter++
		}
	}
	return result, numSkipped
}

// runTransaction handles the processing of the given transaction in an execution
// environment provided by the run function. If enabled, it also handles sponsored
// transactions by checking for sufficient sponsorship funds before executing the
// sponsored transactions and charging the sponsorship fee after execution.
func runTransaction(
	tx *types.Transaction,
	signer types.Signer,
	baseFee *big.Int,
	run func(*core.Message, *types.Transaction) (*types.Receipt, error),
) ([]ProcessedTransaction, error) {
	// TODO: add support for sponsored transactions
	msg, err := TxAsMessage(tx, signer, baseFee)
	if err != nil {
		log.Info("Failed to convert transaction to message", "tx", tx.Hash().Hex(), "err", err)
		return nil, err
	}
	receipt, err := run(msg, tx)
	if err != nil {
		log.Debug("Failed to apply transaction", "tx", tx.Hash().Hex(), "err", err)
		return nil, err
	}
	return []ProcessedTransaction{{
		Transaction: tx,
		Receipt:     receipt,
	}}, nil
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
}

// Run processes a single transaction in the block, where i is the index of
// the transaction in the block. It returns the list of transactions processed
// as a consequence of running the given transaction, which may be more than
// one, and the associated receipts. An error is returned if the given
// transaction could not be processed.
func (tp *TransactionProcessor) Run(i int, tx *types.Transaction) (
	[]ProcessedTransaction,
	error,
) {
	return runTransaction(tx, tp.signer, tp.header.BaseFee,
		func(msg *core.Message, tx *types.Transaction) (*types.Receipt, error) {
			tp.stateDb.SetTxContext(tx.Hash(), i)
			i++
			receipt, _, err := applyTransaction(
				msg, tp.gp, tp.stateDb, tp.blockNumber, tx,
				&tp.usedGas, tp.vmEnvironment, tp.onNewLog,
			)
			return receipt, err
		},
	)
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
	evm.Config.NoBaseFee = msg.SkipNonceChecks

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
