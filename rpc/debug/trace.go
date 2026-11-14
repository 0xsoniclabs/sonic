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

package debug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/params"
)

// defaultTraceTimeout bounds a trace that names no timeout.
const defaultTraceTimeout = 5 * time.Second

var errResponseSizeExceeded = errors.New("response size exceeded")

// builtInTracers are the tracers a trace may name unless AllowJSTracers is
// set: go-ethereum's native tracers and its built-in JavaScript ones.
var builtInTracers = map[string]bool{
	"callTracer": true, "flatCallTracer": true, "prestateTracer": true, "4byteTracer": true, "noopTracer": true,
	"keccak256PreimageTracer": true, "erc7562Tracer": true,
	"bigramTracer": true, "trigramTracer": true, "unigramTracer": true, "opcountTracer": true, "evmdisTracer": true,
	"4byteTracerLegacy": true, "callTracerLegacy": true, "noopTracerLegacy": true, "prestateTracerLegacy": true,
}

type traceTransactionParams struct {
	Hash   common.Hash
	Config *tracers.TraceConfig
}

// traceTransaction traces a transaction of a served block, replaying the
// block up to it.
func (h *handlers) traceTransaction(ctx context.Context, params traceTransactionParams) (json.RawMessage, error) {
	head := h.environment.Head()
	transaction, number, index, err := h.environment.TransactionByHash(head, params.Hash)
	if exec.IsMissing(err) {
		return nil, errors.New("transaction not found")
	}
	if err != nil {
		return nil, err
	}
	if number == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	block, err := h.environment.BlockAt(head, number)
	if err != nil {
		return nil, err
	}
	message, stateDB, err := h.environment.ReplayState(head, block, int(index))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	txContext := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: int(index), TxHash: params.Hash}
	return h.trace(ctx, transaction, message, txContext, block.Header(), stateDB, params.Config, nil)
}

type traceBlockByNumberParams struct {
	Number chain.BlockNumber
	Config *tracers.TraceConfig
}

// traceBlockByNumber traces every transaction of a served block.
func (h *handlers) traceBlockByNumber(ctx context.Context, params traceBlockByNumberParams) ([]*transactionTrace, error) {
	head := h.environment.Head()
	block, err := exec.NoneIfMissing(h.environment.FindBlock(head, chain.ReferenceTo(params.Number)))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", params.Number)
	}
	return h.traceBlock(ctx, head, block, params.Config)
}

type traceBlockByHashParams struct {
	Hash   common.Hash
	Config *tracers.TraceConfig
}

// traceBlockByHash traces every transaction of a served block.
func (h *handlers) traceBlockByHash(ctx context.Context, params traceBlockByHashParams) ([]*transactionTrace, error) {
	head := h.environment.Head()
	block, err := exec.NoneIfMissing(h.environment.FindBlock(head, chain.BlockReference{Hash: &params.Hash}))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", params.Hash.Hex())
	}
	return h.traceBlock(ctx, head, block, params.Config)
}

// transactionTrace is the trace of one transaction of a block.
type transactionTrace struct {
	TxHash common.Hash     `json:"txHash"`
	Result json.RawMessage `json:"result,omitempty"`
}

// traceBlock traces the transactions of a block one after the other on one
// state, within the response size as a whole.
func (h *handlers) traceBlock(ctx context.Context, head, block *evmcore.EvmBlock, config *tracers.TraceConfig) ([]*transactionTrace, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	stateDB, _, err := h.environment.StateAt(head, chain.BlockReference{Hash: &block.ParentHash})
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	if _, err := h.environment.ApplyPreBlockSystemCalls(stateDB, block); err != nil {
		return nil, err
	}
	signer := types.LatestSignerForChainID(h.environment.ChainID())
	maxSize := h.environment.Config().MaxResponseSize
	traces := make([]*transactionTrace, len(block.Transactions))
	size := 0
	for i, transaction := range block.Transactions {
		message, err := evmcore.TxAsMessage(transaction, signer, block.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("cannot get message from transaction %s: %w", transaction.Hash(), err)
		}
		txContext := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: i, TxHash: transaction.Hash()}
		result, err := h.trace(ctx, transaction, message, txContext, block.Header(), stateDB, config, nil)
		if err != nil {
			return nil, err
		}
		traces[i] = &transactionTrace{TxHash: transaction.Hash(), Result: result}
		if size += len(result); maxSize > 0 && size > maxSize {
			return nil, errResponseSizeExceeded
		}
	}
	return traces, nil
}

type traceCallParams struct {
	Arguments exec.CallArguments
	Block     chain.BlockReference
	Config    *exec.TraceCallConfig
}

// traceCall traces a call on the state a transaction of a served block ran
// on, the first one's unless named, with overrides.
func (h *handlers) traceCall(ctx context.Context, params traceCallParams) (json.RawMessage, error) {
	if params.Block.Number != nil && *params.Block.Number == chain.PendingBlockNumber {
		return nil, errors.New("tracing on top of pending is not supported")
	}
	head := h.environment.Head()
	block, err := h.environment.FindBlock(head, params.Block)
	switch {
	case errors.Is(err, chain.ErrMissingSelector):
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	case exec.IsMissing(err):
		return nil, errors.New("header not found")
	case err != nil:
		return nil, err
	}
	config := params.Config
	if config == nil {
		config = &exec.TraceCallConfig{}
	}
	index := 0
	if config.TxIndex != nil {
		index = int(*config.TxIndex)
	}
	_, stateDB, err := h.environment.ReplayState(head, block, index)
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	if err := config.StateOverrides.Apply(stateDB); err != nil {
		return nil, err
	}
	transaction, message, err := params.Arguments.ToTracedCall(h.environment.Config().RPCGasCap, block.BaseFee)
	if err != nil {
		return nil, err
	}
	return h.trace(ctx, transaction, message, new(tracers.Context), block.Header(), stateDB, &config.TraceConfig, config.BlockOverrides)
}

// trace runs a message with the tracer a configuration names, the struct
// logger if none, within its timeout, and answers the tracer's result. The
// tracer is built after its subject is found, and the timeout is checked
// after the tracer, as the legacy tracing does; unlike the legacy tracing,
// a timeout above RPCEVMTimeout is refused.
func (h *handlers) trace(
	ctx context.Context,
	transaction *types.Transaction,
	message *core.Message,
	txContext *tracers.Context,
	header *evmcore.EvmHeader,
	stateDB state.StateDB,
	config *tracers.TraceConfig,
	overrides *exec.BlockOverrides,
) (json.RawMessage, error) {
	if config == nil {
		config = &tracers.TraceConfig{}
	}
	number := header.Number.Uint64()
	chainConfig := h.environment.ChainConfig(number)
	tracer, err := h.newTracer(config, txContext, chainConfig)
	if err != nil {
		return nil, err
	}
	vmConfig, err := h.environment.VMConfig(number)
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	vmConfig.Tracer = tracer.Hooks
	vmConfig.NoBaseFee = true
	hooked := exec.WithHooks(stateDB, tracer.Hooks)
	evm := h.environment.NewEVM(hooked, header, overrides, vmConfig)
	timeout, err := h.traceTimeout(config.Timeout)
	if err != nil {
		return nil, err
	}
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stop := context.AfterFunc(deadline, func() {
		if errors.Is(deadline.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			evm.Cancel()
		}
	})
	defer stop()
	hooked.SetTxContext(txContext.TxHash, txContext.TxIndex)
	var usedGas uint64
	if _, err := evmcore.ApplyTransactionWithEVM(message, chainConfig, core.NewGasPool(message.GasLimit), hooked,
		header.Number, txContext.BlockHash, transaction, &usedGas, evm); err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	result, err := tracer.GetResult()
	if err != nil {
		return nil, err
	}
	if maxSize := h.environment.Config().MaxResponseSize; maxSize > 0 && len(result) > maxSize {
		return nil, errResponseSizeExceeded
	}
	return result, nil
}

// newTracer builds the tracer a configuration names: go-ethereum's struct
// logger, its steps bounded by StructLogLimit, unless it names another.
func (h *handlers) newTracer(config *tracers.TraceConfig, txContext *tracers.Context, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	if config.Tracer == nil {
		if config.Config == nil {
			config.Config = &logger.Config{}
		}
		structLogger := logger.NewStructLogger(config.Config)
		hooks := structLogger.Hooks()
		hooks.OnOpcode = limitStructLogs(hooks.OnOpcode, h.environment.Config().StructLogLimit)
		return &tracers.Tracer{Hooks: hooks, GetResult: structLogger.GetResult, Stop: structLogger.Stop}, nil
	}
	if !h.environment.Config().AllowJSTracers && !builtInTracers[*config.Tracer] {
		return nil, errors.New("custom tracer is not permitted")
	}
	return tracers.DefaultDirectory.New(*config.Tracer, txContext, config.TracerConfig, chainConfig)
}

// traceTimeout is the timeout a trace names, at most RPCEVMTimeout unless
// that is zero, or the default.
func (h *handlers) traceTimeout(named *string) (time.Duration, error) {
	if named == nil {
		return defaultTraceTimeout, nil
	}
	timeout, err := time.ParseDuration(*named)
	if err != nil {
		return 0, dispatch.InvalidParams(err.Error())
	}
	if limit := h.environment.Config().RPCEVMTimeout; limit > 0 && timeout > limit {
		return 0, dispatch.InvalidParams(fmt.Sprintf("tracer timeout %v is above the limit of %v", timeout, limit))
	}
	return timeout, nil
}

// limitStructLogs calls an opcode hook at most limit times: every time for
// a zero limit, never for a negative one, as --rpc.structloglimit says.
func limitStructLogs(onOpcode tracing.OpcodeHook, limit int) tracing.OpcodeHook {
	if onOpcode == nil || limit == 0 {
		return onOpcode
	}
	if limit < 0 {
		return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {}
	}
	reported := 0
	return func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, returnData []byte, depth int, err error) {
		if reported >= limit {
			return
		}
		reported++
		onOpcode(pc, op, gas, cost, scope, returnData, depth, err)
	}
}
