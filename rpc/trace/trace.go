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

// Package trace serves the OpenEthereum trace namespace.
package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/txtrace"
)

// The trace types of trace_call and trace_callMany.
const (
	typeTrace     = "trace"
	typeStateDiff = "stateDiff"
	typeVmTrace   = "vmTrace"
)

// defaultTraceTimeout bounds a traced EVM run when the EVM timeout is zero.
const defaultTraceTimeout = 5 * time.Second

// ErrResponseTooLarge answers a response above MaxResponseSize.
var ErrResponseTooLarge = errors.New("response too large")

type api struct {
	env *exec.Env
}

// Register adds the trace methods.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &api{env: env}
	d.Add("trace_transaction", dispatch.Func(a.transaction))
	d.Add("trace_get", dispatch.Func(a.get))
	d.Add("trace_block", dispatch.Func(a.block))
	d.Add("trace_filter", dispatch.Func(a.filter))
	d.Add("trace_call", dispatch.Func(a.call))
	d.Add("trace_callMany", dispatch.Func(a.callMany))
}

// options are the trace types a call asks for.
type options struct {
	trace, stateDiff, vmTrace bool
}

func parseOptions(traceTypes []string) (options, error) {
	var o options
	for _, t := range traceTypes {
		switch t {
		case typeTrace:
			o.trace = true
		case typeStateDiff:
			o.stateDiff = true
		case typeVmTrace:
			o.vmTrace = true
		default:
			return options{}, fmt.Errorf("unrecognized trace type: %s", t)
		}
	}
	return o, nil
}

// tracedEVM is an EVM with the loggers of the requested trace types.
type tracedEVM struct {
	evm       *vm.EVM
	hooks     *tracing.Hooks
	trace     *txtrace.TraceStructLogger
	stateDiff *txtrace.StateDiffLogger
	vmTrace   *txtrace.VmTraceLogger
	state     state.StateDB // the state, wrapped for the loggers that need it
	stop      func()
}

// newTracedEVM creates the EVM of a block for a traced run; vmBudget bounds
// what a vmTrace keeps. The run is cancelled at the EVM timeout where the
// interpreter polls for it.
func (a *api) newTracedEVM(ctx context.Context, block *evmcore.EvmBlock, st state.StateDB, index uint64, o options, vmBudget uint64) (*tracedEVM, error) {
	header := block.Header()
	number := header.Number.Uint64()
	cfg, err := a.env.VMConfig(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("cannot get vm config for block %d, error: %w", number, err)
	}
	cfg.NoBaseFee = true
	t := &tracedEVM{state: st}
	if o.trace {
		t.trace = txtrace.NewTraceStructLogger(block, uint(index))
		cfg.Tracer = t.trace.Hooks()
	}
	if o.stateDiff {
		t.stateDiff = txtrace.NewStateDiffLogger()
		t.state = exec.WithStateHooks(t.state, t.stateDiff.Hooks())
		onEnter := t.stateDiff.OnEnter
		if cfg.Tracer == nil {
			cfg.Tracer = &tracing.Hooks{OnEnter: onEnter}
		} else {
			existing := cfg.Tracer.OnEnter
			cfg.Tracer.OnEnter = func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
				if existing != nil {
					existing(depth, typ, from, to, input, gas, value)
				}
				onEnter(depth, typ, from, to, input, gas, value)
			}
		}
	}
	if o.vmTrace {
		t.vmTrace = txtrace.NewVmTraceLogger(vmBudget)
		t.state = exec.WithStateHooks(t.state, t.vmTrace.Hooks())
		cfg.Tracer = mergeHooks(cfg.Tracer, t.vmTrace.Hooks())
	}
	t.hooks = cfg.Tracer

	timeout := defaultTraceTimeout
	if evmTimeout := a.env.Config().EVMTimeout; evmTimeout > 0 {
		timeout = evmTimeout
	}
	t.evm = vm.NewEVM(a.env.BlockContext(header), t.state, a.env.ChainConfig(number), cfg)
	deadline, cancel := context.WithTimeout(ctx, timeout)
	stop := context.AfterFunc(deadline, t.evm.Cancel)
	t.stop = func() { stop(); cancel() }
	return t, nil
}

// mergeHooks returns hooks that run orig's hook and then added's, for each
// hook added defines.
func mergeHooks(orig, added *tracing.Hooks) *tracing.Hooks {
	if orig == nil {
		return added
	}
	merged := *orig
	if added.OnTxStart != nil {
		first, then := orig.OnTxStart, added.OnTxStart
		merged.OnTxStart = func(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
			if first != nil {
				first(vm, tx, from)
			}
			then(vm, tx, from)
		}
	}
	if added.OnTxEnd != nil {
		first, then := orig.OnTxEnd, added.OnTxEnd
		merged.OnTxEnd = func(receipt *types.Receipt, err error) {
			if first != nil {
				first(receipt, err)
			}
			then(receipt, err)
		}
	}
	if added.OnEnter != nil {
		first, then := orig.OnEnter, added.OnEnter
		merged.OnEnter = func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			if first != nil {
				first(depth, typ, from, to, input, gas, value)
			}
			then(depth, typ, from, to, input, gas, value)
		}
	}
	if added.OnExit != nil {
		first, then := orig.OnExit, added.OnExit
		merged.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if first != nil {
				first(depth, output, gasUsed, err, reverted)
			}
			then(depth, output, gasUsed, err, reverted)
		}
	}
	if added.OnOpcode != nil {
		first, then := orig.OnOpcode, added.OnOpcode
		merged.OnOpcode = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if first != nil {
				first(pc, op, gas, cost, scope, rData, depth, err)
			}
			then(pc, op, gas, cost, scope, rData, depth, err)
		}
	}
	if added.OnFault != nil {
		first, then := orig.OnFault, added.OnFault
		merged.OnFault = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if first != nil {
				first(pc, op, gas, cost, scope, depth, err)
			}
			then(pc, op, gas, cost, scope, depth, err)
		}
	}
	return &merged
}

type hashArg struct {
	Hash common.Hash `rpc:"hash"`
}

func (a *api) transaction(ctx context.Context, p hashArg) (any, error) {
	return a.traceTxHash(ctx, p.Hash, nil)
}

type getArgs struct {
	Hash       common.Hash    `rpc:"hash"`
	TraceIndex []hexutil.Uint `rpc:"traceIndex"`
}

func (a *api) get(ctx context.Context, p getArgs) (any, error) {
	return a.traceTxHash(ctx, p.Hash, &p.TraceIndex)
}

// traceTxHash replays the block of a transaction up to it and traces it.
func (a *api) traceTxHash(ctx context.Context, hash common.Hash, traceIndex *[]hexutil.Uint) (any, error) {
	head := a.env.Latest(ctx)
	_, number, _, err := a.env.TxByHash(head, hash)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("transaction %s not found", hash.String())
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction %s: %v", hash.String(), err)
	}
	block, err := a.env.BlockAt(head, number)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", number, err.Error())
	}
	return nullIfNil(a.replayBlock(ctx, head, block, &hash, traceIndex))
}

// nullIfNil answers null for a nil list of traces.
func nullIfNil(traces *[]txtrace.ActionTrace, err error) (any, error) {
	if err != nil || traces == nil {
		return nil, err
	}
	return traces, nil
}

type blockArg struct {
	Block chain.BlockRef `rpc:"block"`
}

func (a *api) block(ctx context.Context, p blockArg) (any, error) {
	head := a.env.Latest(ctx)
	number, err := a.env.Resolve(head, p.Block)
	if err != nil {
		return nil, err
	}
	block, err := a.env.BlockAt(head, number)
	if err != nil && !exec.IsMissing(err) {
		return nil, fmt.Errorf("cannot get block %v from db got %v", number, err.Error())
	}
	traces, err := a.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot trace block %v got %v", number, err.Error())
	}
	return nullIfNil(traces, nil)
}

// replayBlock replays a block and returns the traces of its transactions, or
// of the one of txHash; traceIndex selects one trace of them. A transaction
// before the traced one is replayed untraced; its status must match its
// receipt's.
func (a *api) replayBlock(ctx context.Context, head *exec.Head, block *evmcore.EvmBlock, txHash *common.Hash, traceIndex *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	if block == nil {
		return nil, fmt.Errorf("invalid block for tracing")
	}
	if block.NumberU64() == 0 {
		return nil, fmt.Errorf("genesis block is not traceable")
	}
	number := block.NumberU64()
	callTrace := txtrace.CallTrace{Actions: make([]txtrace.ActionTrace, 0)}
	signer := types.LatestSignerForChainID(a.env.ChainConfig(number).ChainID)

	st, err := a.env.StateAtNumber(head, number-1)
	if err != nil {
		return nil, fmt.Errorf("cannot get state for block %v, error: %v", number, exec.HeaderNotFound(err).Error())
	}
	defer st.Release()
	if _, err := a.env.PreBlock(ctx, block, st); err != nil {
		return nil, fmt.Errorf("cannot apply pre-block system calls for block %v, error: %v", number, err.Error())
	}
	receipts, err := a.env.ReceiptsAt(head, number)
	if err != nil && !exec.IsMissing(err) {
		return nil, fmt.Errorf("cannot get receipts for block %v, error: %v", number, err.Error())
	}

	for i, tx := range block.Transactions {
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("cannot get message from transaction %s, error %s", tx.Hash().String(), err)
		}
		if len(receipts) <= i || receipts[i] == nil {
			return nil, fmt.Errorf("no receipt found for transaction %s", tx.Hash().String())
		}
		if txHash == nil || *txHash == tx.Hash() {
			traces, err := a.traceHistoricTx(ctx, msg, st, block, tx, uint64(receipts[i].TransactionIndex), receipts[i].Status)
			if err != nil {
				return nil, fmt.Errorf("cannot get transaction trace for transaction %s, error %s", tx.Hash().String(), err)
			}
			callTrace.AddTraces(traces, traceIndex)
			if txHash != nil {
				break
			}
			continue
		}
		if err := a.replayUntraced(ctx, block, st, tx, i, msg, receipts[i].Status); err != nil {
			return nil, err
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

// replayUntraced runs a transaction to prepare the state for the next one.
func (a *api) replayUntraced(ctx context.Context, block *evmcore.EvmBlock, st state.StateDB, tx *types.Transaction, i int, msg *core.Message, status uint64) error {
	log.Debug("Replaying transaction without trace", "txHash", tx.Hash().String())
	st.SetTxContext(tx.Hash(), i)
	number := block.NumberU64()
	cfg, err := a.env.VMConfig(ctx, number)
	if err != nil {
		return fmt.Errorf("cannot get vm config for block %d, error: %w", number, err)
	}
	cfg.NoBaseFee = true
	evm := vm.NewEVM(a.env.BlockContext(block.Header()), st, a.env.ChainConfig(number), cfg)
	res, err := core.ApplyMessage(evm, msg, core.NewGasPool(msg.GasLimit))
	failed := false
	if err != nil {
		failed = true
		log.Error("Cannot replay transaction", "txHash", tx.Hash().String(), "err", err.Error())
	}
	if res != nil && res.Err != nil {
		failed = true
		log.Debug("Error replaying transaction", "txHash", tx.Hash().String(), "err", res.Err.Error())
	}
	st.EndTransaction()
	if (failed && status == 1) || (!failed && status == 0) {
		return fmt.Errorf("invalid transaction replay state at %s", tx.Hash().String())
	}
	return nil
}

// traceHistoricTx replays a transaction of the chain with tracing; its status
// must match its receipt's.
func (a *api) traceHistoricTx(ctx context.Context, msg *core.Message, st state.StateDB, block *evmcore.EvmBlock, tx *types.Transaction, index, status uint64) (*[]txtrace.ActionTrace, error) {
	t, err := a.newTracedEVM(ctx, block, st, index, options{trace: true}, 0)
	if err != nil {
		return nil, err
	}
	defer t.stop()
	st.SetTxContext(tx.Hash(), int(index))
	// index doubles as the counter of used gas, as in the reference.
	receipt, err := evmcore.ApplyTransactionWithEVM(msg, a.env.ChainConfig(block.NumberU64()), core.NewGasPool(msg.GasLimit), st, block.Number, block.Hash, tx, &index, t.evm)
	actions := t.trace.GetResult()
	if err != nil {
		errTrace := txtrace.GetErrorTraceFromMsg(msg, block.Hash, *block.Number, tx.Hash(), index, err)
		if status == 1 {
			return nil, fmt.Errorf("invalid transaction replay state at %s, error: %s", tx.Hash().String(), err.Error())
		}
		return &[]txtrace.ActionTrace{*errTrace}, nil
	}
	if t.evm.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when replaying tx")
	}
	if status != receipt.Status {
		return nil, fmt.Errorf("invalid transaction replay state at %s, want %v but got %v", tx.Hash().String(), status, receipt.Status)
	}
	return actions, nil
}

func emptyBlockTrace(blockHash common.Hash, blockNumber big.Int) *[]txtrace.ActionTrace {
	trace := txtrace.CallTrace{Actions: make([]txtrace.ActionTrace, 0)}
	blockTrace := txtrace.CreateActionTrace(blockHash, blockNumber, common.Hash{}, 0, "empty")
	blockTrace.Action = txtrace.NewAddressAction(common.Address{}, 0, []byte{}, nil, hexutil.Big{}, nil)
	blockTrace.Error = "Empty block"
	trace.AddTrace(blockTrace)
	return &trace.Actions
}

type callArgs struct {
	Args       exec.CallArgs         `rpc:"args"`
	TraceTypes []string              `rpc:"traceTypes"`
	Block      chain.BlockRef        `rpc:"block"`
	Config     *exec.TraceCallConfig `rpc:"config"`
}

// budget is what is left of MaxResponseSize for the results of one request.
type budget struct {
	left, max int
}

func (a *api) newBudget() *budget {
	max := a.env.Config().MaxResponseSize
	return &budget{left: max, max: max}
}

// vmLimit is what a vmTrace may keep: what is left of the budget.
func (b *budget) vmLimit() uint64 {
	if b.max <= 0 {
		return ^uint64(0)
	}
	return uint64(b.left)
}

// take encodes a result and charges it to the budget.
func (b *budget) take(v any) (json.RawMessage, error) {
	enc, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if b.max > 0 {
		if len(enc) > b.left {
			return nil, ErrResponseTooLarge
		}
		b.left -= len(enc)
	}
	return enc, nil
}

// call traces a call on the state of a block at a position in it. The whole
// response is bounded by MaxResponseSize.
func (a *api) call(ctx context.Context, p callArgs) (any, error) {
	o, err := parseOptions(p.TraceTypes)
	if err != nil {
		return nil, err
	}
	head := a.env.Latest(ctx)
	block, err := a.env.TraceBlock(head, p.Block)
	if err != nil {
		return nil, err
	}
	var txIndex uint
	if p.Config != nil && p.Config.TxIndex != nil {
		txIndex = uint(*p.Config.TxIndex)
	}
	_, st, err := a.env.StateAtTransaction(head, block, int(txIndex))
	if err != nil {
		return nil, err
	}
	defer st.Release()
	if p.Config != nil {
		if err := p.Config.StateOverrides.Apply(st); err != nil {
			return nil, err
		}
	}
	tx, msg, err := a.env.TxMessage(&p.Args, block)
	if err != nil {
		return nil, err
	}
	b := a.newBudget()
	result, err := a.traceCall(ctx, block, msg, st, tx, uint64(txIndex), o, b)
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(result), nil
}

// traceCall runs one traced call and encodes its result against the budget.
func (a *api) traceCall(ctx context.Context, block *evmcore.EvmBlock, msg *core.Message, st state.StateDB, tx *types.Transaction, index uint64, o options, b *budget) (json.RawMessage, error) {
	t, err := a.newTracedEVM(ctx, block, st, index, o, b.vmLimit())
	if err != nil {
		return nil, err
	}
	defer t.stop()
	t.state.SetTxContext(tx.Hash(), int(index))
	if t.hooks != nil && t.hooks.OnTxStart != nil {
		t.hooks.OnTxStart(t.evm.GetVMContext(), tx, msg.From)
	}
	vmContext, err := evmcore.NewEVMTxContext(msg)
	if err != nil {
		return nil, err
	}
	t.evm.SetTxContext(vmContext)
	result, err := core.ApplyMessage(t.evm, msg, core.NewGasPool(msg.GasLimit))
	if err != nil {
		return nil, err
	}
	t.state.EndTransaction()
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: result.UsedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	}
	if t.hooks != nil && t.hooks.OnTxEnd != nil {
		t.hooks.OnTxEnd(receipt, result.Err)
	}
	if t.evm.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when tracing call")
	}

	res := &txtrace.TraceCallResult{Output: hexutil.Bytes(result.ReturnData)}
	if o.trace {
		res.Trace = *t.trace.GetResult()
	}
	if o.stateDiff {
		res.StateDiff = t.stateDiff.GetResult()
	}
	if o.vmTrace {
		if t.vmTrace.Err() != nil {
			return nil, ErrResponseTooLarge
		}
		res.VmTrace = t.vmTrace.GetResult()
	}
	return b.take(res)
}

// CallRequest is one [transaction, traceTypes] pair of trace_callMany.
type CallRequest struct {
	Args       exec.CallArgs
	TraceTypes []string
}

// UnmarshalJSON reads the [transaction, traceTypes] pair.
func (r *CallRequest) UnmarshalJSON(data []byte) error {
	var pair [2]json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil {
		return fmt.Errorf("each call must be [tx, traceTypes]: %w", err)
	}
	if err := json.Unmarshal(pair[0], &r.Args); err != nil {
		return fmt.Errorf("cannot parse transaction args: %w", err)
	}
	if err := json.Unmarshal(pair[1], &r.TraceTypes); err != nil {
		return fmt.Errorf("cannot parse trace types: %w", err)
	}
	return nil
}

type callManyArgs struct {
	Calls  []CallRequest         `rpc:"calls"`
	Block  chain.BlockRef        `rpc:"block"`
	Config *exec.TraceCallConfig `rpc:"config"`
}

// callMany traces calls in sequence, each on the state the ones before left.
// The calls share one budget of MaxResponseSize; each result is encoded as
// soon as it is there.
func (a *api) callMany(ctx context.Context, p callManyArgs) (any, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.TraceBlock(head, p.Block)
	if err != nil {
		return nil, err
	}
	_, st, err := a.env.StateAtTransaction(head, block, 0)
	if err != nil {
		return nil, err
	}
	defer st.Release()
	if p.Config != nil {
		if err := p.Config.StateOverrides.Apply(st); err != nil {
			return nil, err
		}
	}
	b := a.newBudget()
	b.left-- // the opening bracket
	results := make([]json.RawMessage, 0, len(p.Calls))
	for i, call := range p.Calls {
		o, err := parseOptions(call.TraceTypes)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		tx, msg, err := a.env.TxMessage(&call.Args, block)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		if i > 0 {
			b.left-- // the separating comma
		}
		result, err := a.traceCall(ctx, block, msg, st, tx, uint64(i), o, b)
		if err != nil {
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		results = append(results, result)
	}
	enc, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(enc), nil
}
