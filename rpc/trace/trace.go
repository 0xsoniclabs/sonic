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

// Package trace implements the trace_ namespace: OpenEthereum style traces of
// transactions, blocks and calls.
package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/txtrace"
)

const (
	TraceTypeTrace     = "trace"
	TraceTypeStateDiff = "stateDiff"
	TraceTypeVmTrace   = "vmTrace"

	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second
)

type api struct {
	d   *dispatch.Dispatcher
	env *exec.Env
}

// Register registers the trace_ methods.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &api{d: d, env: env}
	dispatch.Add1(d, "trace_transaction", a.transaction, "hash")
	dispatch.Add4(d, "trace_call", a.call, "args", "traceTypes", "block", "config?")
	dispatch.Add1(d, "trace_block", a.block, "blockNrOrHash")
	dispatch.Add2(d, "trace_get", a.get, "hash", "traceIndex")
	dispatch.Add3(d, "trace_callMany", a.callMany, "calls", "block", "config?")
	dispatch.Add1(d, "trace_filter", a.filter, "args")
}

// TraceCallConfig is the config of the call tracing methods. It holds fields to
// override the state for tracing.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *exec.StateOverride
	BlockOverrides *exec.BlockOverrides
	TxIndex        *hexutil.Uint
}

// evmSetup bundles an EVM instance with the tracing loggers and active state
// prepared for a single traced transaction execution.
type evmSetup struct {
	vmenv           *vm.EVM
	tracer          *tracing.Hooks // cfg.Tracer at time of EVM creation
	txTracer        *txtrace.TraceStructLogger
	stateDiffLogger *txtrace.StateDiffLogger
	vmTraceLogger   *txtrace.VmTraceLogger
	activeState     state.StateDB // may be wrapped with stateDiff/vmTrace hooks
}

// setupTracedEVM creates an EVM configured with the requested tracing hooks and
// returns it together with the associated loggers and a cancel function.
// The caller is responsible for calling cancel when execution is complete.
// noBaseFee should be true for simulated calls, false for historical replay.
func (a *api) setupTracedEVM(
	ctx context.Context,
	head dispatch.Head,
	block *evmcore.EvmBlock,
	statedb state.StateDB,
	index uint64,
	tracers TraceOptions,
	noBaseFee bool,
) (*evmSetup, context.CancelFunc, error) {
	header := block.Header()
	cfg, err := a.env.VMConfig(header.Number.Uint64())
	if err != nil {
		return nil, func() {}, fmt.Errorf("cannot get vm config for block %d, error: %w", header.Number.Uint64(), err)
	}
	if noBaseFee {
		cfg.NoBaseFee = true
	}

	setup := &evmSetup{activeState: statedb}

	if tracers.Trace {
		setup.txTracer = txtrace.NewTraceStructLogger(block, uint(index))
		cfg.Tracer = setup.txTracer.Hooks()
	}

	if tracers.StateDiff {
		setup.stateDiffLogger = txtrace.NewStateDiffLogger()
		setup.activeState = exec.WrapStateDbWithLogger(setup.activeState, setup.stateDiffLogger.Hooks())

		sdOnEnter := setup.stateDiffLogger.OnEnter
		if cfg.Tracer == nil {
			cfg.Tracer = &tracing.Hooks{OnEnter: sdOnEnter}
		} else {
			existing := cfg.Tracer.OnEnter
			cfg.Tracer.OnEnter = func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
				if existing != nil {
					existing(depth, typ, from, to, input, gas, value)
				}
				sdOnEnter(depth, typ, from, to, input, gas, value)
			}
		}
	}

	if tracers.VmTrace {
		setup.vmTraceLogger = txtrace.NewVmTraceLogger()
		// Wrap activeState so OnStorageChange fires when storage is written.
		setup.activeState = exec.WrapStateDbWithLogger(setup.activeState, setup.vmTraceLogger.Hooks())
		// Merge VM level hooks into cfg.Tracer.
		cfg.Tracer = mergeVMHooks(cfg.Tracer, setup.vmTraceLogger.Hooks())
	}

	setup.tracer = cfg.Tracer

	timeout := defaultTraceTimeout
	if a.env.Cfg.EVMTimeout > 0 {
		timeout = a.env.Cfg.EVMTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)

	vmenv, err := a.env.NewEVM(head, setup.activeState, header, &cfg, nil)
	if err != nil {
		cancel()
		return nil, func() {}, fmt.Errorf("cannot initialize vm for block %d, error: %s", header.Number.Uint64(), err.Error())
	}

	stop := exec.Cancel(ctx, vmenv)
	setup.vmenv = vmenv
	return setup, func() {
		stop()
		cancel()
	}, nil
}

// transaction returns the inner traces of a transaction.
func (a *api) transaction(ctx context.Context, hash common.Hash) (*[]txtrace.ActionTrace, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_transaction call finished", "txHash", hash.String(), "runtime", time.Since(start))
	}(time.Now())
	return a.traceTxHash(ctx, hash, nil)
}

// call returns the inner traces of a transaction for non historical
// transactions. It supports the "trace", "stateDiff" and "vmTrace" trace types.
func (a *api) call(ctx context.Context, args exec.CallArgs, traceTypes []string, ref dispatch.BlockRef, config *TraceCallConfig) (*txtrace.TraceCallResult, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_Call call finished", "txArgs", args, "runtime", time.Since(start))
	}(time.Now())

	traceOptions, err := parseTraceOptions(traceTypes)
	if err != nil {
		return nil, err
	}

	head := a.env.Chain.Head()
	block, err := a.env.BlockByRef(head, ref)
	if err != nil {
		return nil, err
	}
	var txIndex uint
	if config != nil && config.TxIndex != nil {
		txIndex = uint(*config.TxIndex)
	}

	// Get state
	_, statedb, err := exec.StateAtTransaction(ctx, a.env, head, block, int(txIndex))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	// Apply state overrides
	if config != nil {
		if err := config.StateOverrides.Apply(statedb); err != nil {
			return nil, err
		}
	}

	tx, msg, err := exec.TxAndMessage(&args, a.env.Cfg.GasCap, &block.EvmHeader)
	if err != nil {
		return nil, err
	}

	return a.traceCallExec(ctx, head, block, msg, statedb, tx, uint64(txIndex), traceOptions)
}

// TraceOptions are the kinds of trace a request asks for.
type TraceOptions struct {
	Trace     bool
	StateDiff bool
	VmTrace   bool
}

// parseTraceOptions parses the requested trace types and returns the corresponding booleans.
func parseTraceOptions(traceTypes []string) (trace TraceOptions, err error) {
	for _, traceType := range traceTypes {
		switch traceType {
		case TraceTypeTrace:
			trace.Trace = true
		case TraceTypeStateDiff:
			trace.StateDiff = true
		case TraceTypeVmTrace:
			trace.VmTrace = true
		default:
			return TraceOptions{}, fmt.Errorf("unrecognized trace type: %s", traceType)
		}
	}
	return trace, nil
}

// mergeVMHooks returns a new Hooks that invokes 'orig' hook first, then 'newHooks' hook,
// for each hook that 'newHooks' defines. Hooks only in 'orig' are preserved unchanged.
// Either argument may be nil.
func mergeVMHooks(orig, newHooks *tracing.Hooks) *tracing.Hooks {
	if orig == nil {
		return newHooks
	}
	if newHooks == nil {
		return orig
	}
	merged := *orig // shallow copy, only overwrite fields that newHooks contributes

	if newHooks.OnTxStart != nil {
		origFn, newFn := orig.OnTxStart, newHooks.OnTxStart
		merged.OnTxStart = func(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
			if origFn != nil {
				origFn(vm, tx, from)
			}
			newFn(vm, tx, from)
		}
	}
	if newHooks.OnTxEnd != nil {
		origFn, newFn := orig.OnTxEnd, newHooks.OnTxEnd
		merged.OnTxEnd = func(receipt *types.Receipt, err error) {
			if origFn != nil {
				origFn(receipt, err)
			}
			newFn(receipt, err)
		}
	}
	if newHooks.OnEnter != nil {
		origFn, newFn := orig.OnEnter, newHooks.OnEnter
		merged.OnEnter = func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			if origFn != nil {
				origFn(depth, typ, from, to, input, gas, value)
			}
			newFn(depth, typ, from, to, input, gas, value)
		}
	}
	if newHooks.OnExit != nil {
		origFn, newFn := orig.OnExit, newHooks.OnExit
		merged.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if origFn != nil {
				origFn(depth, output, gasUsed, err, reverted)
			}
			newFn(depth, output, gasUsed, err, reverted)
		}
	}
	if newHooks.OnOpcode != nil {
		origFn, newFn := orig.OnOpcode, newHooks.OnOpcode
		merged.OnOpcode = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if origFn != nil {
				origFn(pc, op, gas, cost, scope, rData, depth, err)
			}
			newFn(pc, op, gas, cost, scope, rData, depth, err)
		}
	}
	if newHooks.OnFault != nil {
		origFn, newFn := orig.OnFault, newHooks.OnFault
		merged.OnFault = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if origFn != nil {
				origFn(pc, op, gas, cost, scope, depth, err)
			}
			newFn(pc, op, gas, cost, scope, depth, err)
		}
	}
	return &merged
}

// traceCallExec executes a simulated transaction and returns a TraceCallResult
// containing optional trace actions, state diff, and/or vmTrace based on the requested types.
func (a *api) traceCallExec(
	ctx context.Context,
	head dispatch.Head,
	block *evmcore.EvmBlock,
	msg *gethcore.Message,
	statedb state.StateDB,
	tx *types.Transaction,
	index uint64,
	traceOptions TraceOptions,
) (*txtrace.TraceCallResult, error) {
	tracedEVM, cancel, err := a.setupTracedEVM(ctx, head, block, statedb, index, traceOptions, true)
	if err != nil {
		return nil, err
	}
	defer cancel()

	tracedEVM.activeState.SetTxContext(tx.Hash(), int(index))

	// Call OnTxStart manually before execution so the trace logger can initialize.
	if tracedEVM.tracer != nil && tracedEVM.tracer.OnTxStart != nil {
		tracedEVM.tracer.OnTxStart(tracedEVM.vmenv.GetVMContext(), tx, msg.From)
	}

	// Set tx context in EVM
	vmContext, err := evmcore.NewEVMTxContext(msg)
	if err != nil {
		return nil, err
	}
	tracedEVM.vmenv.SetTxContext(vmContext)

	// Execute the transaction using core.ApplyMessage to capture raw return data.
	result, applyErr := gethcore.ApplyMessage(tracedEVM.vmenv, msg, gethcore.NewGasPool(msg.GasLimit))

	// Finalize the transaction state regardless of execution outcome.
	tracedEVM.activeState.EndTransaction()

	// Build a synthetic receipt and call OnTxEnd to finalize the trace.
	var receipt *types.Receipt
	if applyErr == nil && result != nil {
		if result.Failed() {
			receipt = &types.Receipt{Status: types.ReceiptStatusFailed, GasUsed: result.UsedGas}
		} else {
			receipt = &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: result.UsedGas}
		}
	}
	if tracedEVM.tracer != nil && tracedEVM.tracer.OnTxEnd != nil {
		tracedEVM.tracer.OnTxEnd(receipt, applyErr)
	}

	if tracedEVM.vmenv.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when tracing call")
	}

	callResult := &txtrace.TraceCallResult{VmTrace: nil}

	if applyErr != nil {
		// Pre-EVM error (e.g., insufficient funds, insufficient gas for the
		// intrinsic cost, nonce mismatch): the result holds a trace of the
		// error, not a failure of the request.
		errTrace := txtrace.GetErrorTraceFromMsg(msg, block.Hash, *block.Number, tx.Hash(), index, applyErr)
		if traceOptions.Trace {
			callResult.Trace = []txtrace.ActionTrace{*errTrace}
		}
		if traceOptions.StateDiff {
			callResult.StateDiff = make(txtrace.StateDiff)
		}
		return callResult, nil
	}

	callResult.Output = hexutil.Bytes(result.ReturnData)

	if traceOptions.Trace {
		callResult.Trace = *tracedEVM.txTracer.GetResult()
	}
	if traceOptions.StateDiff {
		callResult.StateDiff = tracedEVM.stateDiffLogger.GetResult()
	}
	if traceOptions.VmTrace {
		if err := tracedEVM.vmTraceLogger.Err(); err != nil {
			return nil, err
		}
		callResult.VmTrace = tracedEVM.vmTraceLogger.GetResult()
	}

	return callResult, nil
}

// block returns the transaction traces of a block. The history storage
// contract is not written before the replay.
func (a *api) block(ctx context.Context, ref dispatch.BlockRef) (*[]txtrace.ActionTrace, error) {
	head := a.env.Chain.Head()
	var blockNumber uint64
	switch {
	case ref.Number != nil:
		n, err := a.env.Chain.Number(head, *ref.Number)
		if err != nil {
			return nil, err
		}
		blockNumber = n
	case ref.Hash != nil:
		n, ok := a.env.Core.BlockNumberByHash(*ref.Hash)
		if !ok {
			return nil, errors.New("block not found")
		}
		blockNumber = n
	default:
		return nil, errors.New("unknown header selector")
	}

	defer func(start time.Time) {
		log.Debug("Executing trace_block call finished", "block", blockNumber, "runtime", time.Since(start))
	}(time.Now())

	block, err := a.env.BlockAt(head, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get block %v from db got %v", blockNumber, err.Error())
	}

	traces, err := a.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot trace block %v got %v", blockNumber, err.Error())
	}
	return traces, nil
}

// get returns the transaction traces on the specified index position of the
// traces. If the index is empty, just the root trace is returned.
func (a *api) get(ctx context.Context, hash common.Hash, traceIndex []hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_get call finished", "txHash", hash.String(), "index", traceIndex, "runtime", time.Since(start))
	}(time.Now())
	return a.traceTxHash(ctx, hash, &traceIndex)
}

// traceTxHash looks for the block of this transaction hash and traces it.
func (a *api) traceTxHash(ctx context.Context, hash common.Hash, traceIndex *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	head := a.env.Chain.Head()
	tx, blockNumber, _, err := a.env.Core.TxByHash(hash)
	if err != nil && !errors.Is(err, core.ErrNotFound) {
		return nil, fmt.Errorf("cannot get transaction %s: %v", hash.String(), err)
	}
	if err != nil || tx == nil || blockNumber > head.Number {
		return nil, fmt.Errorf("transaction %s not found", hash.String())
	}
	block, err := a.env.BlockAt(head, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", blockNumber, err.Error())
	}
	return a.replayBlock(ctx, head, block, &hash, traceIndex)
}

// replayBlock replays a block and returns traces according to parameters.
//
// txHash
//   - if nil, all transaction traces in the block are collected
//   - if a value, only the trace of that transaction is returned
//
// traceIndex - when specified, only the trace on that index is returned
func (a *api) replayBlock(ctx context.Context, head dispatch.Head, block *evmcore.EvmBlock, txHash *common.Hash, traceIndex *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	if block == nil {
		return nil, fmt.Errorf("invalid block for tracing")
	}

	if block.NumberU64() == 0 {
		return nil, fmt.Errorf("genesis block is not traceable")
	}

	blockNumber := block.NumberU64()
	callTrace := txtrace.CallTrace{
		Actions: make([]txtrace.ActionTrace, 0),
	}

	chainConfig := a.env.ChainConfig(block.NumberU64())
	signer := types.LatestSignerForChainID(chainConfig.ChainID)

	statedb, _, err := a.env.StateAt(head, dispatch.RefNumber(dispatch.BlockNumber(blockNumber-1)))
	if err != nil {
		return nil, fmt.Errorf("cannot get state for block %v, error: %v", block.NumberU64(), err.Error())
	}
	defer statedb.Release()

	receipts, err := a.env.Core.Receipts(blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get receipts for block %v, error: %v", block.NumberU64(), err.Error())
	}

	// loop through all transactions in the block and process them
	for i, tx := range block.Transactions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// replay only the needed transaction if specified
		if txHash == nil || *txHash == tx.Hash() {

			msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
			if err != nil {
				return nil, fmt.Errorf("cannot get message from transaction %s, error %s", tx.Hash().String(), err)
			}

			if len(receipts) <= i || receipts[i] == nil {
				return nil, fmt.Errorf("no receipt found for transaction %s", tx.Hash().String())
			}

			txTraces, err := a.traceTx(ctx, head, msg, statedb, block, tx, uint64(receipts[i].TransactionIndex), receipts[i].Status)
			if err != nil {
				return nil, fmt.Errorf("cannot get transaction trace for transaction %s, error %s", tx.Hash().String(), err)
			}
			callTrace.AddTraces(txTraces, traceIndex)

			// already replayed the specified transaction so end loop
			if txHash != nil {
				break
			}

		} else {

			// Replay the transaction without tracing to prepare the state for the next transaction.
			log.Debug("Replaying transaction without trace", "txHash", tx.Hash().String())
			msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
			if err != nil {
				return nil, fmt.Errorf("cannot get message from transaction %s, error %s", tx.Hash().String(), err)
			}

			if len(receipts) <= i || receipts[i] == nil {
				return nil, fmt.Errorf("no receipt found for transaction %s", tx.Hash().String())
			}

			statedb.SetTxContext(tx.Hash(), i)
			vmConfig, err := a.env.VMConfig(block.NumberU64())
			if err != nil {
				return nil, fmt.Errorf("cannot get vm config for block %d, error: %w", block.NumberU64(), err)
			}
			vmConfig.NoBaseFee = true
			vmConfig.Tracer = nil

			vmenv, err := a.env.NewEVM(head, statedb, block.Header(), &vmConfig, nil)
			if err != nil {
				return nil, fmt.Errorf("cannot initialize vm for transaction %s, error: %s", tx.Hash().String(), err.Error())
			}

			if vmenv.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
				evmcore.ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
			}

			res, err := gethcore.ApplyMessage(vmenv, msg, gethcore.NewGasPool(msg.GasLimit))
			failed := false
			if err != nil {
				failed = true
				log.Error("Cannot replay transaction", "txHash", tx.Hash().String(), "err", err.Error())
			}

			if res != nil && res.Err != nil {
				failed = true
				log.Debug("Error replaying transaction", "txHash", tx.Hash().String(), "err", res.Err.Error())
			}

			statedb.EndTransaction()

			// Check correct replay status according to receipt data
			if (failed && receipts[i].Status == 1) || (!failed && receipts[i].Status == 0) {
				return nil, fmt.Errorf("invalid transaction replay state at %s", tx.Hash().String())
			}
		}
	}

	// In case of empty result create empty trace for empty block
	if len(callTrace.Actions) == 0 {
		if traceIndex != nil || txHash != nil {
			return nil, nil
		}
		return getEmptyBlockTrace(block.Hash, *block.Number), nil
	}

	return &callTrace.Actions, nil
}

// traceTx replays a historical transaction with full tracing and returns the
// action traces. It validates the replayed execution status against the
// on-chain receipt status.
func (a *api) traceTx(
	ctx context.Context,
	head dispatch.Head,
	msg *gethcore.Message,
	statedb state.StateDB,
	block *evmcore.EvmBlock,
	tx *types.Transaction,
	index uint64,
	status uint64,
) (*[]txtrace.ActionTrace, error) {
	tracedEVM, cancel, err := a.setupTracedEVM(ctx, head, block, statedb, index, TraceOptions{Trace: true}, true)
	if err != nil {
		return nil, err
	}
	defer cancel()

	statedb.SetTxContext(tx.Hash(), int(index))
	chainConfig := a.env.ChainConfig(block.Number.Uint64())
	resultReceipt, err := evmcore.ApplyTransactionWithEVM(msg, chainConfig, gethcore.NewGasPool(msg.GasLimit), statedb, block.Number, block.Hash, tx, &index, tracedEVM.vmenv)

	traceActions := tracedEVM.txTracer.GetResult()
	statedb.EndTransaction()

	if err != nil {
		errTrace := txtrace.GetErrorTraceFromMsg(msg, block.Hash, *block.Number, tx.Hash(), index, err)
		at := []txtrace.ActionTrace{*errTrace}
		if status == 1 {
			return nil, fmt.Errorf("invalid transaction replay state at %s, error: %s", tx.Hash().String(), err.Error())
		}
		return &at, nil
	}

	if tracedEVM.vmenv.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when replaying tx")
	}

	if status != resultReceipt.Status {
		return nil, fmt.Errorf("invalid transaction replay state at %s, want %v but got %v", tx.Hash().String(), status, resultReceipt.Status)
	}
	return traceActions, nil
}

// getEmptyBlockTrace returns the trace for an empty block.
func getEmptyBlockTrace(blockHash common.Hash, blockNumber big.Int) *[]txtrace.ActionTrace {
	emptyTrace := txtrace.CallTrace{
		Actions: make([]txtrace.ActionTrace, 0),
	}
	blockTrace := txtrace.CreateActionTrace(blockHash, blockNumber, common.Hash{}, 0, "empty")
	txAction := txtrace.NewAddressAction(common.Address{}, 0, []byte{}, nil, hexutil.Big{}, nil)
	blockTrace.Action = txAction
	blockTrace.Error = "Empty block"
	emptyTrace.AddTrace(blockTrace)
	return &emptyTrace.Actions
}

// CallRequest represents a single [transaction, traceTypes] pair in a trace_callMany request.
type CallRequest struct {
	Args       exec.CallArgs
	TraceTypes []string
}

// UnmarshalJSON parses the [tx, traceTypes] arguments format expected by trace_callMany.
func (r *CallRequest) UnmarshalJSON(data []byte) error {
	var callRequestArgs [2]json.RawMessage
	if err := json.Unmarshal(data, &callRequestArgs); err != nil {
		return fmt.Errorf("each call must be [tx, traceTypes]: %w", err)
	}
	if err := json.Unmarshal(callRequestArgs[0], &r.Args); err != nil {
		return fmt.Errorf("cannot parse transaction args: %w", err)
	}
	if err := json.Unmarshal(callRequestArgs[1], &r.TraceTypes); err != nil {
		return fmt.Errorf("cannot parse trace types: %w", err)
	}
	return nil
}

// callMany executes multiple call traces on top of the same block. Call n is
// executed on top of the state produced by all previous calls n-1, so state
// changes accumulate across the list. The optional config allows state
// overrides to be applied once to the base state before any call is executed.
func (a *api) callMany(ctx context.Context, calls []CallRequest, ref dispatch.BlockRef, config *TraceCallConfig) ([]*txtrace.TraceCallResult, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_callMany call finished", "numCalls", len(calls), "runtime", time.Since(start))
	}(time.Now())

	head := a.env.Chain.Head()
	block, err := a.env.BlockByRef(head, ref)
	if err != nil {
		return nil, err
	}

	_, statedb, err := exec.StateAtTransaction(ctx, a.env, head, block, 0)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	// Apply state overrides to the base state before executing any call.
	if config != nil {
		if err := config.StateOverrides.Apply(statedb); err != nil {
			return nil, err
		}
	}

	results := make([]*txtrace.TraceCallResult, 0, len(calls))
	for i, call := range calls {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		traceOptions, err := parseTraceOptions(call.TraceTypes)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}

		tx, msg, err := exec.TxAndMessage(&call.Args, a.env.Cfg.GasCap, &block.EvmHeader)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}

		result, err := a.traceCallExec(ctx, head, block, msg, statedb, tx, uint64(i), traceOptions)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// FilterArgs represents the arguments for specifying trace targets
type FilterArgs struct {
	FromAddress *[]common.Address  `json:"fromAddress"`
	ToAddress   *[]common.Address  `json:"toAddress"`
	FromBlock   *dispatch.BlockRef `json:"fromBlock"`
	ToBlock     *dispatch.BlockRef `json:"toBlock"`
	After       uint               `json:"after"`
	Count       uint               `json:"count"`
}

// filter is the function of the trace_filter call.
func (a *api) filter(ctx context.Context, args FilterArgs) (json.RawMessage, error) {
	if args.Count == 0 && args.After == 0 {
		// count and order of traces doesn't matter so filter blocks in parallel
		return a.filterBlocksInParallel(ctx, args)
	}
	// filter blocks in series
	return a.filterBlocks(ctx, args)
}

// filterBlocks filters the specified block range in series.
func (a *api) filterBlocks(ctx context.Context, args FilterArgs) (json.RawMessage, error) {
	var traceAdded, traceCount uint

	// resultBuffer is the buffer for collecting result traces
	resultBuffer, err := newJSONResultBuffer(a.env.Cfg.MaxResponseSize)
	if err != nil {
		return nil, err
	}

	head := a.env.Chain.Head()
	// parse arguments
	fromBlock, toBlock, fromAddresses, toAddresses, err := a.parseFilterArguments(head, args)
	if err != nil {
		return nil, err
	}

	// loop through all blocks
	for i := fromBlock; i <= toBlock; i++ {
		traces, err := a.getTracesForBlock(ctx, head, i, fromAddresses, toAddresses)
		if err != nil {
			return nil, err
		}

		// check if traces have to be added
		for _, trace := range traces {
			if traceCount >= args.After {
				if err := resultBuffer.AddObject(&trace); err != nil {
					return nil, err
				}
				traceAdded++
			}
			if traceAdded >= args.Count {
				return resultBuffer.GetResult()
			}
			traceCount++
		}

		// when the context ended return an error
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if i == ^uint64(0) {
			break
		}
	}
	return resultBuffer.GetResult()
}

type traceWorkerResult struct {
	trace []txtrace.ActionTrace
	err   error
}

// filterBlocksInParallel filters the specified block range in parallel. The
// children run in a group of the dispatcher: a panic in one fails the request
// and cancels the others.
func (a *api) filterBlocksInParallel(ctx context.Context, args FilterArgs) (json.RawMessage, error) {
	// resultBuffer is the buffer for collecting result traces
	resultBuffer, err := newJSONResultBuffer(a.env.Cfg.MaxResponseSize)
	if err != nil {
		return nil, err
	}
	head := a.env.Chain.Head()
	// parse arguments
	fromBlock, toBlock, fromAddresses, toAddresses, err := a.parseFilterArguments(head, args)
	if err != nil {
		return nil, err
	}

	group, gctx := a.d.Group(ctx)

	// number of workers
	workerCount := runtime.NumCPU()

	blocks := make(chan uint64, 1)
	results := make(chan traceWorkerResult, 1)

	// make workers to process blocks
	var workers sync.WaitGroup
	for range workerCount {
		workers.Add(1)
		group.Go(func(ctx context.Context) error {
			defer workers.Done()
			a.replayBlockWorker(ctx, head, blocks, results, fromAddresses, toAddresses)
			return nil
		})
	}
	// fill the blocks channel with blocks to process
	group.Go(func(ctx context.Context) error {
		addBlocksForProcessing(ctx, fromBlock, toBlock, blocks)
		return nil
	})
	// close the results channel when the workers are done
	group.Go(func(ctx context.Context) error {
		workers.Wait()
		close(results)
		return nil
	})
	// collect the results
	group.Go(func(ctx context.Context) error {
		for {
			select {
			case res, ok := <-results:
				if !ok {
					return nil
				}
				if res.err != nil {
					return res.err
				}
				for _, trace := range res.trace {
					if err := resultBuffer.AddObject(&trace); err != nil {
						return err
					}
				}
			case <-ctx.Done():
				return nil
			}
		}
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}
	// check if the context expired
	if gctx.Err() != nil && ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout when replaying tx")
		}
		return nil, context.Cause(ctx)
	}
	return resultBuffer.GetResult()
}

// addBlocksForProcessing fills the blocks into the channel for processing and
// closes the channel in the end or if the context was canceled.
func addBlocksForProcessing(ctx context.Context, fromBlock, toBlock uint64, blocks chan<- uint64) {
	defer close(blocks)
	for i := fromBlock; i <= toBlock; i++ {
		select {
		case blocks <- i:
		case <-ctx.Done():
			return
		}
		if i == ^uint64(0) {
			return
		}
	}
}

// parseFilterArguments parses the rpc call arguments and refuses a block span
// wider than the limit, before any block is read; both filter paths route
// through here.
func (a *api) parseFilterArguments(head dispatch.Head, args FilterArgs) (fromBlock, toBlock uint64, fromAddresses, toAddresses map[common.Address]struct{}, err error) {
	if args.FromBlock != nil {
		fromBlock, err = a.resolveBound(head, *args.FromBlock)
		if err != nil {
			return 0, 0, nil, nil, err
		}
	}

	if args.ToBlock != nil {
		toBlock, err = a.resolveBound(head, *args.ToBlock)
		if err != nil {
			return 0, 0, nil, nil, err
		}
	} else {
		toBlock = head.Number
	}

	if maxRange := a.env.Cfg.MaxTraceFilterRange; maxRange > 0 && toBlock > fromBlock && toBlock-fromBlock > maxRange {
		return 0, 0, nil, nil, fmt.Errorf("too wide blocks range, the limit is %d", maxRange)
	}

	if args.FromAddress != nil {
		fromAddresses = make(map[common.Address]struct{})
		for _, addr := range *args.FromAddress {
			fromAddresses[addr] = struct{}{}
		}
	}
	if args.ToAddress != nil {
		toAddresses = make(map[common.Address]struct{})
		for _, addr := range *args.ToAddress {
			toAddresses[addr] = struct{}{}
		}
	}
	return fromBlock, toBlock, fromAddresses, toAddresses, nil
}

// resolveBound resolves the bound of a block range: tags to numbers, a hash to
// the number of its block.
func (a *api) resolveBound(head dispatch.Head, ref dispatch.BlockRef) (uint64, error) {
	switch {
	case ref.Number != nil:
		return a.env.Chain.Number(head, *ref.Number)
	case ref.Hash != nil:
		n, ok := a.env.Core.BlockNumberByHash(*ref.Hash)
		if !ok {
			return 0, errors.New("block not found")
		}
		return n, nil
	}
	return 0, errors.New("unknown header selector")
}

// replayBlockWorker replays blocks in parallel and filters the replayed traces.
func (a *api) replayBlockWorker(
	ctx context.Context,
	head dispatch.Head,
	blocks <-chan uint64,
	results chan<- traceWorkerResult,
	fromAddresses map[common.Address]struct{},
	toAddresses map[common.Address]struct{}) {

	for i := range blocks {
		// check the context before block processing; the error is not
		// propagated as the request reports it
		if ctx.Err() != nil {
			return
		}

		traces, err := a.getTracesForBlock(ctx, head, i, fromAddresses, toAddresses)
		if len(traces) == 0 && err == nil {
			continue
		}

		select {
		case results <- traceWorkerResult{trace: traces, err: err}:
		case <-ctx.Done():
			return
		}
	}
}

// getTracesForBlock replays the transactions of a block and filters out the
// usable traces.
func (a *api) getTracesForBlock(
	ctx context.Context,
	head dispatch.Head,
	blockNumber uint64,
	fromAddresses map[common.Address]struct{},
	toAddresses map[common.Address]struct{},
) (
	[]txtrace.ActionTrace,
	error,
) {
	resultTraces := make([]txtrace.ActionTrace, 0)

	block, err := a.env.BlockAt(head, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", blockNumber, err.Error())
	}

	if block == nil {
		return nil, fmt.Errorf("cannot get block from db %v", blockNumber)
	}

	if block.Transactions.Len() == 0 {
		return resultTraces, nil
	}

	// when the block has any transaction, then process it
	traces, err := a.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, err
	}
	if traces == nil {
		return resultTraces, nil
	}

	for _, trace := range *traces {
		if trace.Action != nil {
			if containsAddress(trace.Action.From, trace.Action.To, fromAddresses, toAddresses) {
				resultTraces = append(resultTraces, trace)
			}
		}
	}
	return resultTraces, nil
}

// containsAddress checks if the from or to address is contained in the map.
func containsAddress(addressFrom *common.Address, addressTo *common.Address, fromAddresses map[common.Address]struct{}, toAddresses map[common.Address]struct{}) bool {
	if len(fromAddresses) > 0 {
		if addressFrom == nil {
			return false
		}
		if _, ok := fromAddresses[*addressFrom]; !ok {
			return false
		}
	}

	if len(toAddresses) > 0 {
		if addressTo == nil {
			return false
		} else if _, ok := toAddresses[*addressTo]; !ok {
			return false
		}
	}
	return true
}

// -- the buffer of a response with a size limit --

const bufferStartSize = 10 * 1024

// ErrResponseTooLarge is returned when the result buffer cannot be expanded.
var ErrResponseTooLarge = errors.New("response too large")

// jsonResultBuffer is a bytes buffer for the JSON array of a response.
type jsonResultBuffer struct {
	bytes.Buffer
	maxResultSize int // limits the maximum size of the RPC response in bytes
}

func newJSONResultBuffer(maxResultSize int) (*jsonResultBuffer, error) {
	b := &jsonResultBuffer{maxResultSize: maxResultSize}
	// grow the buffer to the default start size
	b.Grow(bufferStartSize)
	if err := b.writeString("["); err != nil {
		return nil, err
	}
	return b, nil
}

// GetResult returns the finalized byte array.
func (b *jsonResultBuffer) GetResult() ([]byte, error) {
	if err := b.writeString("]"); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// AddObject marshals and adds an object into the buffer. If the buffer size
// is over the limit, an error is returned.
func (b *jsonResultBuffer) AddObject(obj any) error {
	res, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	if b.Len() > 1 {
		if err := b.writeString(","); err != nil {
			return err
		}
	}
	return b.writeBytes(res)
}

func (b *jsonResultBuffer) writeString(s string) error {
	if b.maxResultSize > 0 && b.Len()+len(s) > b.maxResultSize {
		return ErrResponseTooLarge
	}
	_, err := b.WriteString(s)
	return err
}

func (b *jsonResultBuffer) writeBytes(arr []byte) error {
	if b.maxResultSize > 0 && b.Len()+len(arr) > b.maxResultSize {
		return ErrResponseTooLarge
	}
	_, err := b.Write(arr)
	return err
}
