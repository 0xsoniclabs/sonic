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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// defaultTraceTimeout bounds the trace of one transaction without a timeout
// of its own, as in the reference.
const defaultTraceTimeout = 5 * time.Second

// errResponseSize is the reference's answer to a trace above
// MaxResponseSize.
var errResponseSize = errors.New("response size exceeded")

// allowedTracers are the tracers served without AllowJSTracers: the native
// ones and go-ethereum's built-in JavaScript ones (V6).
var allowedTracers = map[string]bool{
	"callTracer": true, "flatCallTracer": true, "prestateTracer": true, "4byteTracer": true,
	"noopTracer": true, "keccak256PreimageTracer": true, "erc7562Tracer": true,
	"bigramTracer": true, "trigramTracer": true, "unigramTracer": true, "opcountTracer": true,
	"evmdisTracer": true, "4byteTracerLegacy": true, "callTracerLegacy": true,
	"noopTracerLegacy": true, "prestateTracerLegacy": true,
}

// traceTransaction replays the block of a transaction up to it and traces
// it (I8).
func (handler *handlers) traceTransaction(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	hash := dispatch.Argument[common.Hash](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	located, err := exec.NullIfAbsent(handler.environment.TransactionByHash(head, hash))
	if err != nil {
		return nil, err
	}
	if located == nil {
		return nil, errors.New("transaction not found")
	}
	if located.Block == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	block, err := handler.environment.BlockAt(head, located.Block)
	if err != nil {
		return nil, err
	}
	message, statedb, call, err := handler.environment.StateAtTransaction(head, block, int(located.Index))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	subject := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: int(located.Index), TxHash: hash}
	return handler.traceMessage(ctx, located.Transaction, message, subject, block.Header(), statedb, dispatch.Argument[*tracers.TraceConfig](arguments, 1), call)
}

// traceResult is the trace of one transaction of a block.
type traceResult struct {
	TransactionHash common.Hash `json:"txHash"`
	Result          any         `json:"result,omitempty"`
	Error           string      `json:"error,omitempty"`
}

func (handler *handlers) traceBlockByNumber(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	number := dispatch.Argument[chain.BlockNumber](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := exec.NullIfAbsent(handler.environment.FindBlock(head, chain.ReferenceToNumber(number)))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", int64(number))
	}
	return handler.traceBlock(head, block, dispatch.Argument[*tracers.TraceConfig](arguments, 1))
}

func (handler *handlers) traceBlockByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	hash := dispatch.Argument[common.Hash](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := exec.NullIfAbsent(handler.environment.FindBlock(head, chain.ReferenceToHash(hash)))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return handler.traceBlock(head, block, dispatch.Argument[*tracers.TraceConfig](arguments, 1))
}

// traceBlock traces every transaction of a block on the state its
// predecessors left, within MaxResponseSize for the whole block.
func (handler *handlers) traceBlock(head exec.Head, block *evmcore.EvmBlock, config *tracers.TraceConfig) ([]*traceResult, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	statedb, call, err := handler.environment.ReplayState(head, block)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	header := block.Header()
	results := make([]*traceResult, len(block.Transactions))
	size := 0
	limit := handler.environment.Config().MaxResponseSize
	for index, transaction := range block.Transactions {
		message, err := handler.environment.TransactionMessage(block, transaction)
		if err != nil {
			return nil, fmt.Errorf("cannot get message from transaction %s: %w", transaction.Hash(), err)
		}
		subject := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: index, TxHash: transaction.Hash()}
		result, err := handler.traceMessage(head.Context(), transaction, message, subject, header, statedb, config, call)
		if err != nil {
			return nil, err
		}
		results[index] = &traceResult{TransactionHash: transaction.Hash(), Result: result}
		size += len(result)
		if limit > 0 && size > limit {
			return nil, errResponseSize
		}
	}
	return results, nil
}

// traceCall traces a call on top of a block, at a position of it, with the
// overrides of the configuration. The block context takes the block
// overrides and keeps the chain configuration of the block (section 6).
func (handler *handlers) traceCall(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	reference := dispatch.Argument[chain.BlockReference](arguments, 1)
	config := dispatch.Argument[*exec.TraceCallConfig](arguments, 2)
	if reference.Number != nil && *reference.Number == chain.PendingBlockNumber {
		return nil, errors.New("tracing on top of pending is not supported")
	}
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.callBlock(head, reference)
	if err != nil {
		return nil, err
	}
	_, statedb, call, err := handler.environment.StateAtTransaction(head, block, config.Position())
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	var traceConfig *tracers.TraceConfig
	if config != nil {
		call.OverrideBlock(config.BlockOverrides)
		if config.StateOverrides != nil {
			if err := config.StateOverrides.Apply(statedb); err != nil {
				return nil, err
			}
		}
		traceConfig = &config.TraceConfig
	}
	callArguments := dispatch.Argument[exec.CallArguments](arguments, 0)
	transaction, message, err := callTransaction(&callArguments, block, handler.environment.Config().RPCGasCap)
	if err != nil {
		return nil, err
	}
	return handler.traceMessage(ctx, transaction, message, new(tracers.Context), block.Header(), statedb, traceConfig, call)
}

// callBlock finds the block of a call to trace, worded as the reference's
// lookup worded a block it did not find.
func (handler *handlers) callBlock(head exec.Head, reference chain.BlockReference) (*evmcore.EvmBlock, error) {
	if reference.Number == nil && reference.Hash == nil {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	block, err := exec.NullIfAbsent(handler.environment.FindBlock(head, reference))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.New("header not found")
	}
	return block, nil
}

// callTransaction is the message of a call and the legacy transaction that
// stands for it, as the reference built them.
func callTransaction(arguments *exec.CallArguments, block *evmcore.EvmBlock, gasCap uint64) (*types.Transaction, *core.Message, error) {
	message, err := arguments.ToMessage(gasCap, block.BaseFee)
	if err != nil {
		return nil, nil, err
	}
	transaction := types.NewTx(&types.LegacyTx{
		To:       message.To,
		Nonce:    message.Nonce,
		Gas:      message.GasLimit,
		GasPrice: message.GasPrice,
		Value:    message.Value,
		Data:     message.Data,
	})
	return transaction, message, nil
}

// traceMessage runs a message with the tracer the configuration asks for, the
// struct logger by default, within the timeout of the configuration, and
// answers the tracer's result within MaxResponseSize.
func (handler *handlers) traceMessage(
	ctx context.Context,
	transaction *types.Transaction,
	message *core.Message,
	subject *tracers.Context,
	header *evmcore.EvmHeader,
	statedb state.StateDB,
	config *tracers.TraceConfig,
	call *exec.Call,
) (json.RawMessage, error) {
	if config == nil {
		config = &tracers.TraceConfig{}
	}
	chainConfig := handler.environment.ChainConfig(header.Number.Uint64())
	tracer, err := handler.newTracer(config, subject, chainConfig)
	if err != nil {
		return nil, err
	}
	timeout, err := handler.traceTimeout(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	logging := state.WrapWithLogger(statedb, tracer.Hooks)
	evm, stopCancel := call.NewEVM(ctx, logging, tracer.Hooks)
	defer stopCancel()
	stopTracer := afterDeadline(ctx, func() { tracer.Stop(errors.New("execution timeout")) })
	defer stopTracer()

	logging.SetTxContext(subject.TxHash, subject.TxIndex)
	var usedGas uint64
	if _, err := evmcore.ApplyTransactionWithEVM(message, chainConfig, core.NewGasPool(message.GasLimit), logging, header.Number, subject.BlockHash, transaction, &usedGas, evm); err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	result, err := tracer.GetResult()
	if err != nil {
		return nil, err
	}
	if limit := handler.environment.Config().MaxResponseSize; limit > 0 && len(result) > limit {
		return nil, errResponseSize
	}
	return result, nil
}

// newTracer builds the tracer of a configuration: a named one from the
// allow-list unless AllowJSTracers, else the struct logger with its entries
// bounded by StructLogLimit (V6).
func (handler *handlers) newTracer(config *tracers.TraceConfig, subject *tracers.Context, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	limits := handler.environment.Config()
	if config.Tracer != nil {
		if !limits.AllowJSTracers && !allowedTracers[*config.Tracer] {
			return nil, errors.New("custom tracer is not permitted")
		}
		return tracers.DefaultDirectory.New(*config.Tracer, subject, config.TracerConfig, chainConfig)
	}
	loggerConfig := config.Config
	if loggerConfig == nil {
		loggerConfig = &logger.Config{}
	}
	structLogger := logger.NewStructLogger(loggerConfig)
	hooks := structLogger.Hooks()
	hooks.OnOpcode = limitStructLogs(hooks.OnOpcode, limits.StructLogLimit)
	return &tracers.Tracer{Hooks: hooks, GetResult: structLogger.GetResult, Stop: structLogger.Stop}, nil
}

// traceTimeout is the timeout a configuration asks for, the default without
// one; a timeout that does not parse, or above RPCEVMTimeout, is refused
// (V6, section 6).
func (handler *handlers) traceTimeout(config *tracers.TraceConfig) (time.Duration, error) {
	if config.Timeout == nil {
		return defaultTraceTimeout, nil
	}
	timeout, err := time.ParseDuration(*config.Timeout)
	if err != nil {
		return 0, dispatch.InvalidParameters("invalid tracer timeout %q: %v", *config.Timeout, err)
	}
	if limit := handler.environment.Config().RPCEVMTimeout; limit > 0 && timeout > limit {
		return 0, dispatch.InvalidParameters("tracer timeout %s above the limit of %s", timeout, limit)
	}
	return timeout, nil
}

// afterDeadline runs the function when ctx ends at its deadline, not when it
// is cancelled.
func afterDeadline(ctx context.Context, function func()) (stop func() bool) {
	return context.AfterFunc(ctx, func() {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			function()
		}
	})
}

// limitStructLogs lets the opcode hook run at most limit times: zero is no
// limit, a negative limit suppresses the hook, as the flag documents.
func limitStructLogs(onOpcode tracing.OpcodeHook, limit int) tracing.OpcodeHook {
	if onOpcode == nil || limit == 0 {
		return onOpcode
	}
	if limit < 0 {
		return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {}
	}
	reported := 0
	return func(counter uint64, operation byte, gas, cost uint64, scope tracing.OpContext, returned []byte, depth int, err error) {
		if reported >= limit {
			return
		}
		reported++
		onOpcode(counter, operation, gas, cost, scope, returned, depth, err)
	}
}
