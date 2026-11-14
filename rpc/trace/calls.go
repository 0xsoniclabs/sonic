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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
)

// errResponseTooLarge refuses traces whose answer exceeds MaxResponseSize.
var errResponseTooLarge = errors.New("response too large")

type callParams struct {
	Arguments exec.CallArguments
	Kinds     []string
	Block     chain.BlockReference
	Config    *exec.TraceCallConfig
}

// call traces a call, with the kinds of traces it asks for, on the state a
// transaction of a served block ran on, the first one's unless named.
func (h *handlers) call(ctx context.Context, params callParams) (dispatch.Encoded, error) {
	kinds, err := parseOptions(params.Kinds)
	if err != nil {
		return nil, err
	}
	head := h.environment.Head()
	block, err := h.tracedBlock(head, params.Block)
	if err != nil {
		return nil, err
	}
	index := 0
	if params.Config != nil && params.Config.TxIndex != nil {
		index = int(*params.Config.TxIndex)
	}
	_, stateDB, err := h.environment.ReplayState(head, block, index)
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	if params.Config != nil {
		if err := params.Config.StateOverrides.Apply(stateDB); err != nil {
			return nil, err
		}
	}
	transaction, message, err := params.Arguments.ToTracedCall(h.environment.Config().RPCGasCap, block.BaseFee)
	if err != nil {
		return nil, err
	}
	answer := newBudget(h.environment.Config().MaxResponseSize)
	result, err := h.traceCall(ctx, block, message, stateDB, transaction, uint64(index), kinds, answer)
	if err != nil {
		return nil, err
	}
	return answer.encode(result)
}

// tracedBlock is the served block a traced call names, worded as the legacy
// tracing words a block it cannot find.
func (h *handlers) tracedBlock(head *evmcore.EvmBlock, reference chain.BlockReference) (*evmcore.EvmBlock, error) {
	block, err := h.environment.FindBlock(head, reference)
	switch {
	case errors.Is(err, chain.ErrMissingSelector):
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	case exec.IsMissing(err):
		return nil, errors.New("header not found")
	}
	return block, err
}

// callRequest is a call of trace_callMany: its arguments and the kinds of
// traces it asks for.
type callRequest struct {
	Arguments exec.CallArguments
	Kinds     []string
}

// UnmarshalJSON decodes a [call, kinds] pair with the legacy wording.
func (r *callRequest) UnmarshalJSON(data []byte) error {
	var pair [2]json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil {
		return fmt.Errorf("each call must be [tx, traceTypes]: %w", err)
	}
	if err := json.Unmarshal(pair[0], &r.Arguments); err != nil {
		return fmt.Errorf("cannot parse transaction args: %w", err)
	}
	if err := json.Unmarshal(pair[1], &r.Kinds); err != nil {
		return fmt.Errorf("cannot parse trace types: %w", err)
	}
	return nil
}

type callManyParams struct {
	Calls  []callRequest
	Block  chain.BlockReference
	Config *exec.TraceCallConfig
}

// callMany traces calls one after the other on the state of a served block,
// each on the state the ones before it left, within one budget for the whole
// answer.
func (h *handlers) callMany(ctx context.Context, params callManyParams) (dispatch.Encoded, error) {
	head := h.environment.Head()
	block, err := h.tracedBlock(head, params.Block)
	if err != nil {
		return nil, err
	}
	_, stateDB, err := h.environment.ReplayState(head, block, 0)
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	if params.Config != nil {
		if err := params.Config.StateOverrides.Apply(stateDB); err != nil {
			return nil, err
		}
	}
	answer := newBudget(h.environment.Config().MaxResponseSize)
	results := make([]json.RawMessage, 0, len(params.Calls))
	for i, call := range params.Calls {
		kinds, err := parseOptions(call.Kinds)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		transaction, message, err := call.Arguments.ToTracedCall(h.environment.Config().RPCGasCap, block.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		result, err := h.traceCall(ctx, block, message, stateDB, transaction, uint64(i), kinds, answer)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		encoded, err := answer.encode(result)
		if err != nil {
			return nil, err
		}
		results = append(results, json.RawMessage(encoded))
	}
	return json.Marshal(results)
}

// traceCall runs a message with the loggers of the kinds asked for, on a
// state it ends a transaction of, and answers their results. A vmTrace stops
// at what is left of the answer's budget.
func (h *handlers) traceCall(ctx context.Context, block *evmcore.EvmBlock, message *core.Message, stateDB state.StateDB, transaction *types.Transaction, index uint64, kinds options, answer *budget) (*TraceCallResult, error) {
	traced, cancel, err := h.newTracedEVM(ctx, block, stateDB, index, kinds, answer.left())
	if err != nil {
		return nil, err
	}
	defer cancel()
	traced.state.SetTxContext(transaction.Hash(), int(index))
	hooks := traced.evm.Config.Tracer
	if hooks != nil && hooks.OnTxStart != nil {
		hooks.OnTxStart(traced.evm.GetVMContext(), transaction, message.From)
	}
	txContext, err := evmcore.NewEVMTxContext(message)
	if err != nil {
		return nil, err
	}
	traced.evm.SetTxContext(txContext)
	result, err := core.ApplyMessage(traced.evm, message, core.NewGasPool(message.GasLimit))
	if err != nil {
		return nil, err
	}
	traced.state.EndTransaction()
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: result.UsedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	}
	if hooks != nil && hooks.OnTxEnd != nil {
		hooks.OnTxEnd(receipt, result.Err)
	}
	// A vmTrace over its bound stops the run, which is not a timeout.
	if kinds.vmTrace {
		if err := traced.vmTrace.Err(); err != nil {
			return nil, err
		}
	}
	if traced.evm.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when tracing call")
	}
	traces := &TraceCallResult{Output: hexutil.Bytes(result.ReturnData)}
	if kinds.trace {
		traces.Trace = *traced.trace.GetResult()
	}
	if kinds.stateDiff {
		traces.StateDiff = traced.stateDiff.GetResult()
	}
	if kinds.vmTrace {
		traces.VmTrace = traced.vmTrace.GetResult()
	}
	return traces, nil
}

// budget is what is left of MaxResponseSize to an answer, zero for no
// bound.
type budget struct {
	limit, used int
}

func newBudget(limit int) *budget {
	return &budget{limit: limit}
}

// left is how many bytes the answer may still take, the most a vmTrace may
// keep.
func (b *budget) left() uint64 {
	if b.limit == 0 {
		return vmTraceSizeLimit
	}
	return uint64(max(b.limit-b.used, 0))
}

// encode encodes a part of the answer and counts it against the budget.
func (b *budget) encode(part any) (dispatch.Encoded, error) {
	encoded, err := json.Marshal(part)
	if err != nil {
		return nil, err
	}
	if b.used += len(encoded); b.limit > 0 && b.used > b.limit {
		return nil, errResponseTooLarge
	}
	return encoded, nil
}

// mergeHooks answers hooks that call the first hooks, then the second ones,
// for each hook the second ones set; either may be nil.
func mergeHooks(first, second *tracing.Hooks) *tracing.Hooks {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	merged := *first
	if second.OnTxStart != nil {
		a, b := first.OnTxStart, second.OnTxStart
		merged.OnTxStart = func(vm *tracing.VMContext, transaction *types.Transaction, from common.Address) {
			if a != nil {
				a(vm, transaction, from)
			}
			b(vm, transaction, from)
		}
	}
	if second.OnTxEnd != nil {
		a, b := first.OnTxEnd, second.OnTxEnd
		merged.OnTxEnd = func(receipt *types.Receipt, err error) {
			if a != nil {
				a(receipt, err)
			}
			b(receipt, err)
		}
	}
	if second.OnEnter != nil {
		a, b := first.OnEnter, second.OnEnter
		merged.OnEnter = func(depth int, kind byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			if a != nil {
				a(depth, kind, from, to, input, gas, value)
			}
			b(depth, kind, from, to, input, gas, value)
		}
	}
	if second.OnExit != nil {
		a, b := first.OnExit, second.OnExit
		merged.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if a != nil {
				a(depth, output, gasUsed, err, reverted)
			}
			b(depth, output, gasUsed, err, reverted)
		}
	}
	if second.OnOpcode != nil {
		a, b := first.OnOpcode, second.OnOpcode
		merged.OnOpcode = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, returnData []byte, depth int, err error) {
			if a != nil {
				a(pc, op, gas, cost, scope, returnData, depth, err)
			}
			b(pc, op, gas, cost, scope, returnData, depth, err)
		}
	}
	if second.OnFault != nil {
		a, b := first.OnFault, second.OnFault
		merged.OnFault = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if a != nil {
				a(pc, op, gas, cost, scope, depth, err)
			}
			b(pc, op, gas, cost, scope, depth, err)
		}
	}
	return &merged
}
