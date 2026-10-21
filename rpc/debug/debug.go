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

// Package debug implements the debug_ namespace: tracing of transactions,
// blocks and calls, and the raw encodings of blocks.
package debug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	runtimedebug "runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js" // registers the built-in JavaScript tracers
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native" // registers the native tracers
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second
)

// ErrMaxResponseSize is returned if the size of a response is over the limit.
var ErrMaxResponseSize = errors.New("response size exceeded")

var errMethodNotEnabled = errors.New("this method is not enabled")

type api struct {
	env *exec.Env
}

// Register registers the debug_ methods. The profiling methods of
// go-ethereum's debug handler are answered with an error: they are not enabled
// on the public interface.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &api{env: env}
	dispatch.Add1(d, "debug_getBlockRlp", a.getBlockRlp, "number")
	dispatch.Add1(d, "debug_printBlock", a.printBlock, "number")
	dispatch.Add2(d, "debug_traceTransaction", a.traceTransaction, "hash", "config?")
	dispatch.Add2(d, "debug_traceBlockByNumber", a.traceBlockByNumber, "number", "config?")
	dispatch.Add2(d, "debug_traceBlockByHash", a.traceBlockByHash, "hash", "config?")
	dispatch.Add3(d, "debug_traceCall", a.traceCall, "args", "block", "config?")

	dispatch.Add1(d, "debug_getRawBlock", a.getRawBlock, "blockNrOrHash")
	dispatch.Add1(d, "debug_getRawHeader", a.getRawHeader, "blockNrOrHash")
	dispatch.Add1(d, "debug_getRawReceipts", a.getRawReceipts, "blockNrOrHash")
	dispatch.Add1(d, "debug_getRawTransaction", a.getRawTransaction, "hash")

	registerDisabled(d)
}

func registerDisabled(d *dispatch.Dispatcher) {
	disabled := func(ctx context.Context) (any, error) { return nil, errMethodNotEnabled }
	dispatch.Add1(d, "debug_verbosity", func(ctx context.Context, _ int) (any, error) { return disabled(ctx) }, "level")
	dispatch.Add1(d, "debug_vmodule", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "pattern")
	dispatch.Add0(d, "debug_memStats", func(ctx context.Context) (*runtime.MemStats, error) { return nil, errMethodNotEnabled })
	dispatch.Add0(d, "debug_gcStats", func(ctx context.Context) (*runtimedebug.GCStats, error) { return nil, errMethodNotEnabled })
	dispatch.Add2(d, "debug_cpuProfile", func(ctx context.Context, _ string, _ uint) (any, error) { return disabled(ctx) }, "file", "nsec")
	dispatch.Add1(d, "debug_startCPUProfile", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "file")
	dispatch.Add0(d, "debug_stopCPUProfile", disabled)
	dispatch.Add2(d, "debug_goTrace", func(ctx context.Context, _ string, _ uint) (any, error) { return disabled(ctx) }, "file", "nsec")
	dispatch.Add1(d, "debug_startGoTrace", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "file")
	dispatch.Add0(d, "debug_stopGoTrace", disabled)
	dispatch.Add2(d, "debug_blockProfile", func(ctx context.Context, _ string, _ uint) (any, error) { return disabled(ctx) }, "file", "nsec")
	dispatch.Add1(d, "debug_setBlockProfileRate", func(ctx context.Context, _ int) (any, error) { return disabled(ctx) }, "rate")
	dispatch.Add1(d, "debug_writeBlockProfile", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "file")
	dispatch.Add2(d, "debug_mutexProfile", func(ctx context.Context, _ string, _ uint) (any, error) { return disabled(ctx) }, "file", "nsec")
	dispatch.Add1(d, "debug_setMutexProfileFraction", func(ctx context.Context, _ int) (any, error) { return disabled(ctx) }, "rate")
	dispatch.Add1(d, "debug_writeMutexProfile", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "file")
	dispatch.Add1(d, "debug_writeMemProfile", func(ctx context.Context, _ string) (any, error) { return disabled(ctx) }, "file")
	dispatch.Add1(d, "debug_stacks", func(ctx context.Context, _ *string) (string, error) { return "", errMethodNotEnabled }, "filter?")
	dispatch.Add0(d, "debug_freeOSMemory", disabled)
	dispatch.Add1(d, "debug_setGCPercent", func(ctx context.Context, _ int) (int, error) { return 0, errMethodNotEnabled }, "v")
	dispatch.Add1(d, "debug_setMemoryLimit", func(ctx context.Context, _ int64) (int64, error) { return 0, errMethodNotEnabled }, "limit")
}

// getBlockRlp retrieves the RLP encoding of a single block.
func (a *api) getBlockRlp(_ context.Context, number uint64) (string, error) {
	block, _ := a.env.BlockByNumber(a.env.Chain.Head(), dispatch.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// printBlock retrieves a block and returns its pretty printed form.
func (a *api) printBlock(_ context.Context, number uint64) (string, error) {
	block, err := a.env.BlockByNumber(a.env.Chain.Head(), dispatch.BlockNumber(number))
	if err != nil {
		return "", err
	}
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// -- raw encodings --

// rawBlock finds the block a reference names, for the raw encodings.
func (a *api) rawBlock(ref dispatch.BlockRef) (*evmcore.EvmBlock, error) {
	block, err := a.env.FindBlock(a.env.Chain.Head(), ref)
	if err != nil {
		return nil, err
	}
	if block == nil {
		if ref.Hash != nil {
			return nil, fmt.Errorf("block %s not found", ref.Hash.Hex())
		}
		return nil, fmt.Errorf("block #%d not found", ref.Number.Int64())
	}
	return block, nil
}

// getRawBlock returns the RLP encoding of a block.
func (a *api) getRawBlock(_ context.Context, ref dispatch.BlockRef) (hexutil.Bytes, error) {
	block, err := a.rawBlock(ref)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(block.EthBlock())
}

// getRawHeader returns the RLP encoding of a block header.
func (a *api) getRawHeader(_ context.Context, ref dispatch.BlockRef) (hexutil.Bytes, error) {
	block, err := a.rawBlock(ref)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(block.EthHeader())
}

// getRawReceipts returns the consensus encoding of the receipts of a block.
func (a *api) getRawReceipts(_ context.Context, ref dispatch.BlockRef) ([]hexutil.Bytes, error) {
	block, err := a.rawBlock(ref)
	if err != nil {
		return nil, err
	}
	receipts, err := a.env.Core.Receipts(block.NumberU64())
	if err != nil {
		return nil, err
	}
	result := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		encoded, err := receipt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result[i] = encoded
	}
	return result, nil
}

// getRawTransaction returns the binary encoding of a transaction.
func (a *api) getRawTransaction(_ context.Context, hash common.Hash) (hexutil.Bytes, error) {
	head := a.env.Chain.Head()
	tx, _, _, err := a.env.TxByHash(head, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		if tx = a.env.Core.PoolTransaction(hash); tx == nil {
			return nil, nil
		}
	}
	return tx.MarshalBinary()
}

// -- tracing --

// traceTransaction returns the structured logs created during the execution of
// the transaction.
func (a *api) traceTransaction(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) (any, error) {
	head := a.env.Chain.Head()
	tx, blockNumber, index, err := a.env.TxByHash(head, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction %s not found", hash.Hex())
	}
	// It shouldn't happen in practice.
	if blockNumber == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	block, err := a.env.BlockAt(head, blockNumber)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", blockNumber)
	}
	msg, statedb, err := exec.StateAtTransaction(ctx, a.env, head, block, int(index))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	txctx := &tracers.Context{
		BlockHash:   block.Hash,
		BlockNumber: block.Number,
		TxIndex:     int(index),
		TxHash:      hash,
	}
	return a.traceTx(ctx, head, tx, msg, txctx, block.Header(), statedb, config, nil)
}

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

// traceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (a *api) traceTx(
	ctx context.Context,
	head dispatch.Head,
	tx *types.Transaction,
	message *gethcore.Message,
	txctx *tracers.Context,
	blockHeader *evmcore.EvmHeader,
	statedb state.StateDB,
	config *tracers.TraceConfig,
	blockCtx *vm.BlockContext,
) (json.RawMessage, error) {
	var (
		tracer  *tracers.Tracer
		err     error
		timeout = defaultTraceTimeout
		usedGas uint64
		cfg     = a.env.Cfg
	)
	if config == nil {
		config = &tracers.TraceConfig{}
	}

	chainConfig := a.env.ChainConfig(blockHeader.Number.Uint64())

	// Default tracer is the struct logger
	if config.Tracer == nil {
		if config.Config == nil {
			config.Config = &logger.Config{Limit: cfg.StructLogLimit}
		} else {
			if cfg.StructLogLimit > 0 &&
				(config.Limit == 0 || config.Limit > cfg.StructLogLimit) {

				config.Limit = cfg.StructLogLimit
			}
		}
		logger := logger.NewStructLogger(config.Config)
		tracer = &tracers.Tracer{
			Hooks:     logger.Hooks(),
			GetResult: logger.GetResult,
			Stop:      logger.Stop,
		}
	} else {
		if !cfg.AllowJSTracers && !isTracerWhitelisted(*config.Tracer) {
			return nil, fmt.Errorf("custom tracer is not permitted")
		}
		tracer, err = tracers.DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig, chainConfig)
		if err != nil {
			return nil, err
		}
	}

	// A malformed or excessive timeout is refused where the tracer is built, once
	// its subject is found.
	if config.Timeout != nil {
		timeout, err = time.ParseDuration(*config.Timeout)
		if err != nil {
			return nil, dispatch.InvalidParams("invalid timeout: %v", err)
		}
		if cfg.EVMTimeout > 0 && timeout > cfg.EVMTimeout {
			return nil, dispatch.InvalidParams("timeout %v exceeds the limit of %v", timeout, cfg.EVMTimeout)
		}
	}

	evmconfig, err := a.env.VMConfig(blockHeader.Number.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	evmconfig.Tracer = tracer.Hooks
	evmconfig.NoBaseFee = true

	loggingStateDB := exec.WrapStateDbWithLogger(statedb, tracer.Hooks)

	var vmBlockCtx vm.BlockContext
	if blockCtx != nil {
		vmBlockCtx = *blockCtx
	} else {
		vmBlockCtx = a.env.BlockContext(head, blockHeader)
	}
	// The chain configuration is that of the block, whatever the block context
	// was overridden to.
	vmenv := a.env.NewEVMWithConfig(loggingStateDB, evmconfig, vmBlockCtx, chainConfig)

	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stopWatch := context.AfterFunc(deadlineCtx, func() {
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
		}
		// Stop evm execution. Note cancellation is not necessarily immediate.
		vmenv.Cancel()
	})
	defer stopWatch()

	// Call SetTxContext to clear out the statedb access list
	loggingStateDB.SetTxContext(txctx.TxHash, txctx.TxIndex)

	// Run the transaction with tracing enabled.
	_, err = evmcore.ApplyTransactionWithEVM(
		message,
		chainConfig,
		gethcore.NewGasPool(message.GasLimit),
		loggingStateDB,
		blockHeader.Number,
		txctx.BlockHash,
		tx,
		&usedGas,
		vmenv,
	)
	if err != nil {
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

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result any         `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// traceBlockByNumber returns the structured logs created during the execution
// of the transactions of the block.
func (a *api) traceBlockByNumber(ctx context.Context, number dispatch.BlockNumber, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	head := a.env.Chain.Head()
	block, err := a.env.BlockByNumber(head, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return a.traceBlock(ctx, head, block, config)
}

// traceBlockByHash returns the structured logs created during the execution of
// the transactions of the block.
func (a *api) traceBlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	head := a.env.Chain.Head()
	block, err := a.env.BlockByHash(head, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return a.traceBlock(ctx, head, block, config)
}

// traceBlock configures a new tracer according to the provided configuration,
// and executes all the transactions contained within. The return value will be
// one item per transaction, dependent on the requested tracer. The history
// storage contract is not written.
func (a *api) traceBlock(ctx context.Context, head dispatch.Head, block *evmcore.EvmBlock, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	statedb, _, err := a.env.StateAt(head, dispatch.RefHash(block.ParentHash))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	var (
		chainConfig   = a.env.ChainConfig(block.NumberU64())
		txs           = block.Transactions
		signer        = types.LatestSignerForChainID(chainConfig.ChainID)
		results       = make([]*txTraceResult, len(txs))
		resultsLength int
	)
	for i, tx := range txs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			return nil, err
		}
		txctx := &tracers.Context{
			BlockHash:   block.Hash,
			BlockNumber: block.Number,
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}
		res, err := a.traceTx(ctx, head, tx, msg, txctx, block.Header(), statedb, config, nil)
		if err != nil {
			return nil, err
		}
		results[i] = &txTraceResult{TxHash: tx.Hash(), Result: res}
		resultsLength += len(res)

		statedb.EndTransaction()

		// limit the response size.
		if a.env.Cfg.MaxResponseSize > 0 && resultsLength > a.env.Cfg.MaxResponseSize {
			return nil, ErrMaxResponseSize
		}
	}
	return results, nil
}

// traceCallConfig is the config for traceCall API. It holds more fields to
// override the state for tracing.
type traceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *exec.StateOverride
	BlockOverrides *exec.BlockOverrides
	TxIndex        *hexutil.Uint
}

// traceCall generates traces for non historical transactions. It is similar to
// eth_call but with debug capabilities.
func (a *api) traceCall(ctx context.Context, args exec.CallArgs, ref dispatch.BlockRef, config *traceCallConfig) (any, error) {
	// If pending block, return error
	if ref.Number != nil && *ref.Number == dispatch.PendingBlockNumber {
		return nil, errors.New("tracing on top of pending is not supported")
	}

	head := a.env.Chain.Head()
	// Get block
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

	blockCtx := a.env.BlockContext(head, &block.EvmHeader)
	if config != nil {
		config.BlockOverrides.Apply(&blockCtx)
	}

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

	var traceConfig *tracers.TraceConfig
	if config != nil {
		traceConfig = &config.TraceConfig
	}
	return a.traceTx(ctx, head, tx, msg, new(tracers.Context), &block.EvmHeader, statedb, traceConfig, &blockCtx)
}
