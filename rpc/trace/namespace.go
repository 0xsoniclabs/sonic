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

package trace

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// defaultTraceTimeout bounds a replay unless RPCEVMTimeout is set.
const defaultTraceTimeout = 5 * time.Second

type handlers struct {
	environment *exec.Environment
}

// Register adds the methods of the trace namespace, the OpenEthereum
// traces, to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
	h := &handlers{environment: environment}
	dispatch.Add(dispatcher, "trace_transaction", h.transaction)
	dispatch.Add(dispatcher, "trace_get", h.get)
	dispatch.Add(dispatcher, "trace_block", h.block)
	dispatch.Add(dispatcher, "trace_filter", h.filter)
	dispatch.Add(dispatcher, "trace_call", h.call)
	dispatch.Add(dispatcher, "trace_callMany", h.callMany)
}

type hashParams struct {
	Hash common.Hash
}

// transaction answers the traces of a transaction of a served block.
func (h *handlers) transaction(ctx context.Context, params hashParams) (*[]ActionTrace, error) {
	return h.traceTransaction(ctx, params.Hash, nil)
}

type getParams struct {
	Hash  common.Hash
	Index []hexutil.Uint
}

// get answers the trace of a transaction at a position of its traces.
func (h *handlers) get(ctx context.Context, params getParams) (*[]ActionTrace, error) {
	return h.traceTransaction(ctx, params.Hash, &params.Index)
}

func (h *handlers) traceTransaction(ctx context.Context, hash common.Hash, traceIndex *[]hexutil.Uint) (*[]ActionTrace, error) {
	head := h.environment.Head()
	_, number, _, err := h.environment.TransactionByHash(head, hash)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("transaction %s not found", hash.String())
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction %s: %v", hash.String(), err)
	}
	block, err := h.environment.BlockAt(head, number)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", number, err.Error())
	}
	return h.replayBlock(ctx, head, block, &hash, traceIndex)
}

type blockParams struct {
	Block chain.BlockReference
}

// block answers the traces of every transaction of a served block.
func (h *handlers) block(ctx context.Context, params blockParams) (*[]ActionTrace, error) {
	head := h.environment.Head()
	number, err := h.environment.Resolve(head, params.Block)
	if err != nil {
		return nil, err
	}
	block, err := exec.NoneIfMissing(h.environment.BlockAt(head, number))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %v from db got %v", number, err.Error())
	}
	traces, err := h.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot trace block %v got %v", number, err.Error())
	}
	return traces, nil
}

// replayBlock replays a block on one state and traces all its transactions,
// or the one of a hash, and of that one the trace at a position if given.
// The replay of every transaction must end as its receipt tells.
func (h *handlers) replayBlock(ctx context.Context, head, block *evmcore.EvmBlock, txHash *common.Hash, traceIndex *[]hexutil.Uint) (*[]ActionTrace, error) {
	if block == nil {
		return nil, fmt.Errorf("invalid block for tracing")
	}
	if block.NumberU64() == 0 {
		return nil, fmt.Errorf("genesis block is not traceable")
	}
	number := block.NumberU64()
	stateDB, _, err := h.environment.StateAt(head, chain.ReferenceTo(chain.BlockNumber(number-1)))
	if err != nil {
		return nil, fmt.Errorf("cannot get state for block %v, error: %v", number, err.Error())
	}
	defer stateDB.Release()
	if _, err := h.environment.ApplyPreBlockSystemCalls(stateDB, block); err != nil {
		return nil, fmt.Errorf("cannot apply pre-block system calls for block %v, error: %v", number, err.Error())
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, number))
	if err != nil {
		return nil, fmt.Errorf("cannot get receipts for block %v, error: %v", number, err.Error())
	}
	signer := types.LatestSignerForChainID(h.environment.ChainID())
	callTrace := CallTrace{Actions: make([]ActionTrace, 0)}
	for i, transaction := range block.Transactions {
		message, err := evmcore.TxAsMessage(transaction, signer, block.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("cannot get message from transaction %s, error %s", transaction.Hash().String(), err)
		}
		if len(receipts) <= i || receipts[i] == nil {
			return nil, fmt.Errorf("no receipt found for transaction %s", transaction.Hash().String())
		}
		if txHash != nil && *txHash != transaction.Hash() {
			if err := h.replayUntraced(stateDB, block, transaction, i, message, receipts[i].Status); err != nil {
				return nil, err
			}
			continue
		}
		traces, err := h.traceReplayed(ctx, message, stateDB, block, transaction, uint64(receipts[i].TransactionIndex), receipts[i].Status)
		if err != nil {
			return nil, fmt.Errorf("cannot get transaction trace for transaction %s, error %s", transaction.Hash().String(), err)
		}
		callTrace.AddTraces(traces, traceIndex)
		if txHash != nil {
			break
		}
	}
	if len(callTrace.Actions) == 0 {
		if traceIndex != nil || txHash != nil {
			return nil, nil
		}
		return emptyBlockTrace(block.Hash, *block.Number), nil
	}
	return &callTrace.Actions, nil
}

// replayUntraced runs a transaction to prepare the state for the ones after
// it, and checks that it ends as its receipt tells.
func (h *handlers) replayUntraced(stateDB state.StateDB, block *evmcore.EvmBlock, transaction *types.Transaction, position int, message *core.Message, status uint64) error {
	log.Debug("Replaying transaction without trace", "txHash", transaction.Hash().String())
	stateDB.SetTxContext(transaction.Hash(), position)
	vmConfig, err := h.environment.VMConfig(block.NumberU64())
	if err != nil {
		return fmt.Errorf("cannot get vm config for block %d, error: %w", block.NumberU64(), err)
	}
	vmConfig.NoBaseFee = true
	evm := h.environment.NewEVM(stateDB, block.Header(), nil, vmConfig)
	result, err := core.ApplyMessage(evm, message, core.NewGasPool(message.GasLimit))
	failed := false
	if err != nil {
		failed = true
		log.Error("Cannot replay transaction", "txHash", transaction.Hash().String(), "err", err.Error())
	}
	if result != nil && result.Err != nil {
		failed = true
		log.Debug("Error replaying transaction", "txHash", transaction.Hash().String(), "err", result.Err.Error())
	}
	stateDB.EndTransaction()
	if failed != (status == types.ReceiptStatusFailed) {
		return fmt.Errorf("invalid transaction replay state at %s", transaction.Hash().String())
	}
	return nil
}

// traceReplayed replays a transaction with its traces, and checks that it
// ends as its receipt tells.
func (h *handlers) traceReplayed(ctx context.Context, message *core.Message, stateDB state.StateDB, block *evmcore.EvmBlock, transaction *types.Transaction, index uint64, status uint64) (*[]ActionTrace, error) {
	traced, cancel, err := h.newTracedEVM(ctx, block, stateDB, index, options{trace: true}, 0)
	if err != nil {
		return nil, err
	}
	defer cancel()
	stateDB.SetTxContext(transaction.Hash(), int(index))
	chainConfig := h.environment.ChainConfig(block.NumberU64())
	receipt, err := evmcore.ApplyTransactionWithEVM(message, chainConfig, core.NewGasPool(message.GasLimit), stateDB, block.Number, block.Hash, transaction, &index, traced.evm)
	actions := traced.trace.GetResult()
	if err != nil {
		if status == types.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("invalid transaction replay state at %s, error: %s", transaction.Hash().String(), err.Error())
		}
		return &[]ActionTrace{*GetErrorTraceFromMsg(message, block.Hash, *block.Number, transaction.Hash(), index, err)}, nil
	}
	if traced.evm.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when replaying tx")
	}
	if status != receipt.Status {
		return nil, fmt.Errorf("invalid transaction replay state at %s, want %v but got %v", transaction.Hash().String(), status, receipt.Status)
	}
	return actions, nil
}

// emptyBlockTrace is the trace of a block without transactions.
func emptyBlockTrace(blockHash common.Hash, blockNumber big.Int) *[]ActionTrace {
	empty := CallTrace{Actions: make([]ActionTrace, 0)}
	blockTrace := CreateActionTrace(blockHash, blockNumber, common.Hash{}, 0, "empty")
	blockTrace.Action = NewAddressAction(common.Address{}, 0, []byte{}, nil, hexutil.Big{}, nil)
	blockTrace.Error = "Empty block"
	empty.AddTrace(blockTrace)
	return &empty.Actions
}

// options are the kinds of traces of a traced call.
type options struct {
	trace, stateDiff, vmTrace bool
}

// parseOptions reads the kinds of traces a call asks for.
func parseOptions(kinds []string) (options, error) {
	var parsed options
	for _, kind := range kinds {
		switch kind {
		case "trace":
			parsed.trace = true
		case "stateDiff":
			parsed.stateDiff = true
		case "vmTrace":
			parsed.vmTrace = true
		default:
			return options{}, fmt.Errorf("unrecognized trace type: %s", kind)
		}
	}
	return parsed, nil
}

// tracedEVM is an EVM with the loggers of the kinds of traces asked for, and
// the state the EVM runs on, wrapped as they need.
type tracedEVM struct {
	evm       *vm.EVM
	trace     *TraceStructLogger
	stateDiff *StateDiffLogger
	vmTrace   *VmTraceLogger
	state     state.StateDB
}

// newTracedEVM builds the EVM of a block with the loggers of the options,
// a vmTrace keeping at most vmTraceLimit bytes, without a base fee, stopped
// at RPCEVMTimeout or the default timeout.
func (h *handlers) newTracedEVM(ctx context.Context, block *evmcore.EvmBlock, stateDB state.StateDB, index uint64, kinds options, vmTraceLimit uint64) (*tracedEVM, context.CancelFunc, error) {
	vmConfig, err := h.environment.VMConfig(block.NumberU64())
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get vm config for block %d, error: %w", block.NumberU64(), err)
	}
	vmConfig.NoBaseFee = true
	traced := &tracedEVM{state: stateDB}
	if kinds.trace {
		traced.trace = NewTraceStructLogger(block, uint(index))
		vmConfig.Tracer = traced.trace.Hooks()
	}
	if kinds.stateDiff {
		traced.stateDiff = NewStateDiffLogger(traced.state)
		traced.state = traced.stateDiff
		// Tosca's interpreter does not stop when the EVM is cancelled.
		vmConfig.Interpreter = vmConfig.InterpreterForTracing
	}
	if kinds.vmTrace {
		traced.vmTrace = NewVmTraceLogger(vmTraceLimit)
		traced.state = exec.WithHooks(traced.state, traced.vmTrace.Hooks())
		vmConfig.Tracer = mergeHooks(vmConfig.Tracer, traced.vmTrace.Hooks())
	}
	timeout := defaultTraceTimeout
	if configured := h.environment.Config().RPCEVMTimeout; configured > 0 {
		timeout = configured
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	traced.evm = h.environment.NewEVM(traced.state, block.Header(), nil, vmConfig)
	if traced.vmTrace != nil {
		traced.vmTrace.OnDrop(traced.evm.Cancel)
	}
	stop := context.AfterFunc(ctx, traced.evm.Cancel)
	return traced, func() {
		stop()
		cancel()
	}, nil
}
