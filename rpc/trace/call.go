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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/txtrace"
)

// traceTypes are the kinds of trace a traced call asks for.
type traceTypes struct {
	trace, stateDiff, vmTrace bool
}

func parseTraceTypes(names []string) (traceTypes, error) {
	var parsed traceTypes
	for _, name := range names {
		switch name {
		case "trace":
			parsed.trace = true
		case "stateDiff":
			parsed.stateDiff = true
		case "vmTrace":
			parsed.vmTrace = true
		default:
			return traceTypes{}, fmt.Errorf("unrecognized trace type: %s", name)
		}
	}
	return parsed, nil
}

// callRequest is one call of trace_callMany: [transaction, traceTypes].
type callRequest struct {
	arguments  exec.CallArguments
	traceTypes []string
}

func (request *callRequest) UnmarshalJSON(data []byte) error {
	var parts [2]json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return fmt.Errorf("each call must be [tx, traceTypes]: %w", err)
	}
	if err := json.Unmarshal(parts[0], &request.arguments); err != nil {
		return fmt.Errorf("cannot parse transaction args: %w", err)
	}
	if err := json.Unmarshal(parts[1], &request.traceTypes); err != nil {
		return fmt.Errorf("cannot parse trace types: %w", err)
	}
	return nil
}

// tracedCalls is a block traced calls run on, on the state at a position of
// the block.
type tracedCalls struct {
	handler *handlers
	head    exec.Head
	block   *evmcore.EvmBlock
	statedb state.StateDB
	call    *exec.Call
}

// prepareCalls opens the state at the position of the configuration in the
// block the reference names, with the configuration's state overrides; the
// caller releases it. Block overrides are not applied, as in the reference.
func (handler *handlers) prepareCalls(ctx context.Context, reference chain.BlockReference, config *exec.TraceCallConfig) (*tracedCalls, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.environment.CallBlock(head, reference)
	if err != nil {
		return nil, err
	}
	_, statedb, call, err := handler.environment.StateAtTransaction(head, block, config.Position())
	if err != nil {
		return nil, err
	}
	if config != nil && config.StateOverrides != nil {
		if err := config.StateOverrides.Apply(statedb); err != nil {
			statedb.Release()
			return nil, err
		}
	}
	return &tracedCalls{handler: handler, head: head, block: block, statedb: statedb, call: call}, nil
}

// call answers the traces of a call at a position of a block, within
// MaxResponseSize (V12).
func (handler *handlers) call(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	requested, err := parseTraceTypes(dispatch.Argument[[]string](arguments, 1))
	if err != nil {
		return nil, err
	}
	config := dispatch.Argument[*exec.TraceCallConfig](arguments, 3)
	calls, err := handler.prepareCalls(ctx, dispatch.Argument[chain.BlockReference](arguments, 2), config)
	if err != nil {
		return nil, err
	}
	defer calls.statedb.Release()
	limit := handler.environment.Config().MaxResponseSize
	callArguments := dispatch.Argument[exec.CallArguments](arguments, 0)
	result, err := calls.trace(&callArguments, requested, uint64(config.Position()), limit)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(encoded) > limit {
		return nil, errResponseTooLarge
	}
	return dispatch.Encoded(encoded), nil
}

// callMany answers the traces of calls run one after the other on the
// state each left, with the state overrides applied once before the first.
// The calls share one budget of MaxResponseSize: each result is encoded as
// soon as it is there, and a vmTrace stops at what is left (V12).
func (handler *handlers) callMany(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	requests := dispatch.Argument[[]callRequest](arguments, 0)
	config := dispatch.Argument[*exec.TraceCallConfig](arguments, 2)
	calls, err := handler.prepareCalls(ctx, dispatch.Argument[chain.BlockReference](arguments, 1), &exec.TraceCallConfig{StateOverrides: stateOverrides(config)})
	if err != nil {
		return nil, err
	}
	defer calls.statedb.Release()
	buffer, err := newResultBuffer(handler.environment.Config().MaxResponseSize)
	if err != nil {
		return nil, err
	}
	for index := range requests {
		if err := calls.traceInto(buffer, &requests[index], uint64(index)); err != nil {
			return nil, fmt.Errorf("call %d: %w", index, err)
		}
	}
	result, err := buffer.result()
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(result), nil
}

// stateOverrides are the state overrides of a configuration; trace_callMany
// runs every call at the start of the block, whatever position it names.
func stateOverrides(config *exec.TraceCallConfig) *exec.StateOverride {
	if config == nil {
		return nil
	}
	return config.StateOverrides
}

func (calls *tracedCalls) traceInto(buffer *resultBuffer, request *callRequest, index uint64) error {
	requested, err := parseTraceTypes(request.traceTypes)
	if err != nil {
		return err
	}
	result, err := calls.trace(&request.arguments, requested, index, buffer.left())
	if err != nil {
		return err
	}
	return buffer.add(result)
}

// trace runs a call with the tracers of the types, a vmTrace stopping the
// run once it keeps more than the budget.
func (calls *tracedCalls) trace(arguments *exec.CallArguments, requested traceTypes, index uint64, budget int) (*txtrace.TraceCallResult, error) {
	transaction, message, err := arguments.ToTracedCall(calls.handler.environment.Config().RPCGasCap, calls.block.BaseFee)
	if err != nil {
		return nil, err
	}
	run := newTracedRun(calls.block, calls.statedb, index, requested, budget)
	ctx, cancel := context.WithTimeout(calls.head.Context(), calls.handler.timeout())
	defer cancel()
	evm, stop := calls.call.NewEVM(ctx, run.statedb, run.hooks)
	defer stop()
	run.evm = evm
	return run.apply(transaction, message, index)
}

// tracedRun is one traced call: its tracers, and the state the ones that
// follow the state's changes wrap.
type tracedRun struct {
	requested  traceTypes
	statedb    state.StateDB
	hooks      *tracing.Hooks
	evm        *vm.EVM
	actions    *txtrace.TraceStructLogger
	stateDiff  *txtrace.StateDiffLogger
	vmTrace    *txtrace.VmTraceLogger
	overBudget bool
}

func newTracedRun(block *evmcore.EvmBlock, statedb state.StateDB, index uint64, requested traceTypes, budget int) *tracedRun {
	run := &tracedRun{requested: requested, statedb: statedb}
	if requested.trace {
		run.actions = txtrace.NewTraceStructLogger(block, uint(index))
		run.hooks = run.actions.Hooks()
	}
	if requested.stateDiff {
		run.stateDiff = txtrace.NewStateDiffLogger()
		run.statedb = state.WrapWithLogger(run.statedb, run.stateDiff.Hooks())
		run.hooks = mergeHooks(run.hooks, &tracing.Hooks{OnEnter: run.stateDiff.OnEnter})
	}
	if requested.vmTrace {
		run.vmTrace = txtrace.NewVmTraceLoggerWithLimit(uint64(max(budget, 0)), run.stop)
		run.statedb = state.WrapWithLogger(run.statedb, run.vmTrace.Hooks())
		run.hooks = mergeHooks(run.hooks, run.vmTrace.Hooks())
	}
	return run
}

// stop ends the run once the vmTrace dropped its trace.
func (run *tracedRun) stop() {
	run.overBudget = true
	run.evm.Cancel()
}

// apply runs the message as the reference's traceCallExec did, telling the
// tracers of the start and the end of the transaction itself.
func (run *tracedRun) apply(transaction *types.Transaction, message *core.Message, index uint64) (*txtrace.TraceCallResult, error) {
	run.statedb.SetTxContext(transaction.Hash(), int(index))
	if run.hooks != nil && run.hooks.OnTxStart != nil {
		run.hooks.OnTxStart(run.evm.GetVMContext(), transaction, message.From)
	}
	transactionContext, err := evmcore.NewEVMTxContext(message)
	if err != nil {
		return nil, err
	}
	run.evm.SetTxContext(transactionContext)
	result, err := core.ApplyMessage(run.evm, message, core.NewGasPool(message.GasLimit))
	if err != nil {
		return nil, err
	}
	run.statedb.EndTransaction()
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: result.UsedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	}
	if run.hooks != nil && run.hooks.OnTxEnd != nil {
		run.hooks.OnTxEnd(receipt, result.Err)
	}
	if run.overBudget {
		return nil, errResponseTooLarge
	}
	if run.evm.Cancelled() {
		return nil, errors.New("EVM was cancelled when tracing call")
	}
	return run.result(result.ReturnData), nil
}

func (run *tracedRun) result(output []byte) *txtrace.TraceCallResult {
	result := &txtrace.TraceCallResult{Output: hexutil.Bytes(output)}
	if run.requested.trace {
		result.Trace = *run.actions.GetResult()
	}
	if run.requested.stateDiff {
		result.StateDiff = run.stateDiff.GetResult()
	}
	if run.requested.vmTrace {
		result.VmTrace = run.vmTrace.GetResult()
	}
	return result
}

// mergeHooks returns hooks that run the first's, then the second's, for each
// hook the second defines; the first's other hooks are kept. Either may be
// nil.
func mergeHooks(first, second *tracing.Hooks) *tracing.Hooks {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	merged := *first
	merged.OnTxStart = chainTransactionStart(first.OnTxStart, second.OnTxStart)
	merged.OnTxEnd = chainTransactionEnd(first.OnTxEnd, second.OnTxEnd)
	merged.OnEnter = chainEnter(first.OnEnter, second.OnEnter)
	merged.OnExit = chainExit(first.OnExit, second.OnExit)
	merged.OnOpcode = chainOpcode(first.OnOpcode, second.OnOpcode)
	merged.OnFault = chainFault(first.OnFault, second.OnFault)
	return &merged
}

func chainTransactionStart(first, second tracing.TxStartHook) tracing.TxStartHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(vmContext *tracing.VMContext, transaction *types.Transaction, sender common.Address) {
		first(vmContext, transaction, sender)
		second(vmContext, transaction, sender)
	}
}

func chainTransactionEnd(first, second tracing.TxEndHook) tracing.TxEndHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(receipt *types.Receipt, err error) {
		first(receipt, err)
		second(receipt, err)
	}
}

func chainEnter(first, second tracing.EnterHook) tracing.EnterHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(depth int, kind byte, sender, recipient common.Address, input []byte, gas uint64, value *big.Int) {
		first(depth, kind, sender, recipient, input, gas, value)
		second(depth, kind, sender, recipient, input, gas, value)
	}
}

func chainExit(first, second tracing.ExitHook) tracing.ExitHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
		first(depth, output, gasUsed, err, reverted)
		second(depth, output, gasUsed, err, reverted)
	}
}

func chainOpcode(first, second tracing.OpcodeHook) tracing.OpcodeHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(counter uint64, operation byte, gas, cost uint64, scope tracing.OpContext, returned []byte, depth int, err error) {
		first(counter, operation, gas, cost, scope, returned, depth, err)
		second(counter, operation, gas, cost, scope, returned, depth, err)
	}
}

func chainFault(first, second tracing.FaultHook) tracing.FaultHook {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(counter uint64, operation byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
		first(counter, operation, gas, cost, scope, depth, err)
		second(counter, operation, gas, cost, scope, depth, err)
	}
}
