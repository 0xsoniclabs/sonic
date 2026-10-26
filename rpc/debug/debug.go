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

// Package debug serves the debug namespace: blocks in raw forms and the
// go-ethereum tracers over transactions, blocks and calls.
package debug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native" // the native tracers debug_* names
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// defaultTraceTimeout bounds one traced transaction whose request names no
// timeout.
const defaultTraceTimeout = 5 * time.Second

// ErrMaxResponseSize is the answer of a trace larger than MaxResponseSize.
var ErrMaxResponseSize = errors.New("response size exceeded")

type api struct {
	env *exec.Env
}

// Register adds the debug methods.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &api{env: env}
	d.Add("debug_getBlockRlp", dispatch.Func(a.getBlockRlp))
	d.Add("debug_printBlock", dispatch.Func(a.printBlock))
	d.Add("debug_traceTransaction", dispatch.Func(a.traceTransaction))
	d.Add("debug_traceBlockByNumber", dispatch.Func(a.traceBlockByNumber))
	d.Add("debug_traceBlockByHash", dispatch.Func(a.traceBlockByHash))
	d.Add("debug_traceCall", dispatch.Func(a.traceCall))
	registerProfilingShadows(d)
}

type rawNumberArg struct {
	Number uint64 `rpc:"number"`
}

// rawBlock returns the block of a number the way the reference read it: cast
// to a block number, so that a number above math.MaxInt64 is a tag.
func (a *api) rawBlock(ctx context.Context, number uint64) (*evmcore.EvmBlock, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.FindBlock(head, chain.NumberRef(chain.BlockNumber(number))) // the reference's cast, kept for F1
	if exec.IsMissing(err) || errors.Is(err, chain.ErrInvalidTag) {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, err
}

func (a *api) getBlockRlp(ctx context.Context, p rawNumberArg) (any, error) {
	block, err := a.rawBlock(ctx, p.Number)
	if err != nil {
		// The reference answered any failure of the lookup as a missing block.
		return nil, fmt.Errorf("block #%d not found", p.Number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("%x", encoded), nil
}

func (a *api) printBlock(ctx context.Context, p rawNumberArg) (any, error) {
	block, err := a.rawBlock(ctx, p.Number)
	if err != nil {
		return nil, err
	}
	return spew.Sdump(block), nil
}

type traceTransactionArgs struct {
	Hash   common.Hash          `rpc:"hash"`
	Config *tracers.TraceConfig `rpc:"config"`
}

func (a *api) traceTransaction(ctx context.Context, p traceTransactionArgs) (any, error) {
	head := a.env.Latest(ctx)
	tx, number, index, err := a.env.TxByHash(head, p.Hash)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("transaction %s not found", p.Hash.Hex())
	}
	if err != nil {
		return nil, err
	}
	if number == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	block, err := a.env.BlockAt(head, number)
	if err != nil {
		return nil, err
	}
	msg, st, err := a.env.StateAtTransaction(head, block, int(index))
	if err != nil {
		return nil, err
	}
	defer st.Release()
	txctx := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: int(index), TxHash: p.Hash}
	return a.traceTx(head, tx, msg, txctx, block.Header(), st, p.Config, nil)
}

type traceBlockByNumberArgs struct {
	Number chain.BlockNumber    `rpc:"number"`
	Config *tracers.TraceConfig `rpc:"config"`
}

type traceBlockByHashArgs struct {
	Hash   common.Hash          `rpc:"hash"`
	Config *tracers.TraceConfig `rpc:"config"`
}

func (a *api) traceBlockByNumber(ctx context.Context, p traceBlockByNumberArgs) (any, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.FindBlock(head, chain.NumberRef(p.Number))
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("block #%d not found", p.Number)
	}
	if err != nil {
		return nil, err
	}
	return a.traceBlock(ctx, head, block, p.Config)
}

func (a *api) traceBlockByHash(ctx context.Context, p traceBlockByHashArgs) (any, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.FindBlock(head, chain.HashRef(p.Hash))
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("block %s not found", p.Hash.Hex())
	}
	if err != nil {
		return nil, err
	}
	return a.traceBlock(ctx, head, block, p.Config)
}

// txTraceResult is the trace of one transaction of a block.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`
	Result any         `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// traceBlock traces every transaction of a block in order; the traces
// together are bounded by MaxResponseSize. The result is encoded once.
func (a *api) traceBlock(ctx context.Context, head *exec.Head, block *evmcore.EvmBlock, config *tracers.TraceConfig) (any, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	st, err := a.env.ParentState(head, block)
	if err != nil {
		return nil, err
	}
	defer st.Release()
	if _, err := a.env.PreBlock(ctx, block, st); err != nil {
		return nil, err
	}

	maxSize := a.env.Config().MaxResponseSize
	signer := types.LatestSignerForChainID(a.env.ChainConfig(block.NumberU64()).ChainID)
	results := make([]*txTraceResult, len(block.Transactions))
	size := 0
	for i, tx := range block.Transactions {
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("cannot get message from transaction %s: %w", tx.Hash(), err)
		}
		txctx := &tracers.Context{BlockHash: block.Hash, BlockNumber: block.Number, TxIndex: i, TxHash: tx.Hash()}
		res, err := a.traceTx(head, tx, msg, txctx, block.Header(), st, config, nil)
		if err != nil {
			return nil, err
		}
		results[i] = &txTraceResult{TxHash: tx.Hash(), Result: res}
		size += len(res)
		if maxSize > 0 && size > maxSize {
			return nil, ErrMaxResponseSize
		}
	}
	enc, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(enc), nil
}

type traceCallArgs struct {
	Args   exec.CallArgs         `rpc:"args"`
	Block  chain.BlockRef        `rpc:"block"`
	Config *exec.TraceCallConfig `rpc:"config"`
}

// traceCall traces a call on the state of a block at a position in it: the
// block first, then the state overrides, then the call object. The EVM uses
// the chain configuration of the queried block.
func (a *api) traceCall(ctx context.Context, p traceCallArgs) (any, error) {
	if p.Block.IsPending() {
		return nil, errors.New("tracing on top of pending is not supported")
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

	blockCtx := a.env.BlockContext(&block.EvmHeader)
	if p.Config != nil && p.Config.BlockOverrides != nil {
		p.Config.BlockOverrides.Apply(&blockCtx)
	}
	if p.Config != nil && p.Config.StateOverrides != nil {
		if err := p.Config.StateOverrides.Apply(st); err != nil {
			return nil, err
		}
	}
	tx, msg, err := a.env.TxMessage(&p.Args, block)
	if err != nil {
		return nil, err
	}
	var traceConfig *tracers.TraceConfig
	if p.Config != nil {
		traceConfig = &p.Config.TraceConfig
	}
	return a.traceTx(head, tx, msg, new(tracers.Context), &block.EvmHeader, st, traceConfig, &blockCtx)
}

// isTracerWhitelisted reports whether a tracer is one of the built-in ones,
// the only ones debug_trace* runs unless JavaScript tracers are allowed.
func isTracerWhitelisted(name string) bool {
	switch name {
	case
		// native tracers
		"callTracer",
		"flatCallTracer",
		"prestateTracer",
		"4byteTracer",
		"noopTracer",
		"keccak256PreimageTracer",
		"erc7562Tracer",
		// built-in JS tracers
		"bigramTracer",
		"trigramTracer",
		"unigramTracer",
		"opcountTracer",
		"evmdisTracer",
		"4byteTracerLegacy",
		"callTracerLegacy",
		"noopTracerLegacy",
		"prestateTracerLegacy":
		return true
	}
	return false
}

// traceTx runs one transaction under the tracer the configuration names, the
// struct logger by default. The tracer's name, configuration and timeout are
// checked here, after the subject of the trace was found; a timeout that does
// not parse, or is above the EVM timeout, is refused with -32602.
func (a *api) traceTx(head *exec.Head, tx *types.Transaction, msg *core.Message, txctx *tracers.Context, header *evmcore.EvmHeader, st state.StateDB, config *tracers.TraceConfig, blockCtx *vm.BlockContext) (json.RawMessage, error) {
	ctx := head.Context()
	cfg := a.env.Config()
	if config == nil {
		config = &tracers.TraceConfig{}
	}
	number := header.Number.Uint64()
	chainConfig := a.env.ChainConfig(number)

	var tracer *tracers.Tracer
	if config.Tracer == nil {
		if config.Config == nil {
			config.Config = &logger.Config{Limit: cfg.StructLogLimit}
		} else if cfg.StructLogLimit > 0 && (config.Limit == 0 || config.Limit > cfg.StructLogLimit) {
			config.Limit = cfg.StructLogLimit
		}
		structLogger := logger.NewStructLogger(config.Config)
		tracer = &tracers.Tracer{Hooks: structLogger.Hooks(), GetResult: structLogger.GetResult, Stop: structLogger.Stop}
	} else {
		if !cfg.AllowJSTracers && !isTracerWhitelisted(*config.Tracer) {
			return nil, fmt.Errorf("custom tracer is not permitted")
		}
		var err error
		if tracer, err = tracers.DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig, chainConfig); err != nil {
			return nil, err
		}
	}
	timeout := defaultTraceTimeout
	if config.Timeout != nil {
		var err error
		if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
			return nil, dispatch.InvalidParams("invalid tracer timeout %q: %v", *config.Timeout, err)
		}
		if cfg.EVMTimeout > 0 && timeout > cfg.EVMTimeout {
			return nil, dispatch.InvalidParams("tracer timeout %v is above the limit %v", timeout, cfg.EVMTimeout)
		}
	}

	vmConfig, err := a.env.VMConfig(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	vmConfig.Tracer = tracer.Hooks
	vmConfig.NoBaseFee = true
	traced := exec.WithStateHooks(st, tracer.Hooks)
	if blockCtx == nil {
		bc := a.env.BlockContext(header)
		blockCtx = &bc
	}
	evm := vm.NewEVM(*blockCtx, traced, chainConfig, vmConfig)

	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stop := context.AfterFunc(deadline, func() {
		if errors.Is(deadline.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			evm.Cancel()
		}
	})
	defer stop()

	traced.SetTxContext(txctx.TxHash, txctx.TxIndex)
	var usedGas uint64
	if _, err := evmcore.ApplyTransactionWithEVM(msg, chainConfig, core.NewGasPool(msg.GasLimit), traced, header.Number, txctx.BlockHash, tx, &usedGas, evm); err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	result, err := tracer.GetResult()
	if err != nil {
		return nil, err
	}
	if cfg.MaxResponseSize > 0 && len(result) > cfg.MaxResponseSize {
		return nil, ErrMaxResponseSize
	}
	return result, nil
}
