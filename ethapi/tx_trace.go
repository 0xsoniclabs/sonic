package ethapi

import (
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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/txtrace"
	"github.com/0xsoniclabs/sonic/utils/signers/gsignercache"
)

const (
	TraceTypeTrace     = "trace"
	TraceTypeStateDiff = "stateDiff"
	TraceTypeVmTrace   = "vmTrace"
)

// PublicTxTraceAPI provides an API to access transaction tracing
// It offers only methods that operate on public data that is freely available to anyone
type PublicTxTraceAPI struct {
	b               Backend
	maxResponseSize int // in bytes
}

// NewPublicTxTraceAPI creates a new transaction trace API
func NewPublicTxTraceAPI(b Backend, maxResponseSize int) *PublicTxTraceAPI {
	return &PublicTxTraceAPI{
		b:               b,
		maxResponseSize: maxResponseSize,
	}
}

// Transaction - trace_transaction function returns transaction inner traces
func (s *PublicTxTraceAPI) Transaction(ctx context.Context, hash common.Hash) (*[]txtrace.ActionTrace, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_transaction call finished", "txHash", hash.String(), "runtime", time.Since(start))
	}(time.Now())
	return s.traceTxHash(ctx, hash, nil)
}

// Call - trace_call function returns transaction inner traces for non historical transactions
func (s *PublicTxTraceAPI) Call(ctx context.Context, args TransactionArgs, traceTypes []string, blockNrOrHash rpc.BlockNumberOrHash, config *TraceCallConfig) (*[]txtrace.ActionTrace, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_Call call finished", "txArgs", args, "runtime", time.Since(start))
	}(time.Now())

	for _, traceType := range traceTypes {
		switch traceType {
		case TraceTypeTrace:
			continue
		case TraceTypeStateDiff:
			return nil, fmt.Errorf("stateDiff trace type is not supported")
		case TraceTypeVmTrace:
			return nil, fmt.Errorf("vmTrace trace type is not supported")
		default:
			return nil, fmt.Errorf("unrecognized trace type: %s", traceType)
		}
	}

	block, err := getEvmBlockFromNumberOrHash(ctx, blockNrOrHash, s.b)
	if err != nil {
		return nil, err
	}
	var txIndex uint
	if config != nil && config.TxIndex != nil {
		txIndex = uint(*config.TxIndex)
	}

	// Get state
	_, statedb, err := stateAtTransaction(ctx, block, int(txIndex), s.b)
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

	tx, msg, err := getTxAndMessage(&args, block, s.b)
	if err != nil {
		return nil, err
	}

	return s.traceTx(ctx, s.b, block.Header(), msg, statedb, block, tx, uint64(txIndex), 1)
}

// Block - trace_block function returns transaction traces in given block
func (s *PublicTxTraceAPI) Block(ctx context.Context, numberOrHash rpc.BlockNumberOrHash) (*[]txtrace.ActionTrace, error) {

	blockNumber, _ := numberOrHash.Number()

	if blockNumber == rpc.PendingBlockNumber {
		return nil, fmt.Errorf("cannot trace pending block")
	}

	currentBlockNumber := s.b.CurrentBlock().NumberU64()
	if blockNumber == rpc.LatestBlockNumber {
		blockNumber = rpc.BlockNumber(currentBlockNumber)
	}

	if uint64(blockNumber.Int64()) > currentBlockNumber {
		return nil, fmt.Errorf("requested block number %v is greater than current head block number %v", blockNumber.Int64(), currentBlockNumber)
	}

	defer func(start time.Time) {
		log.Debug("Executing trace_block call finished", "block", blockNumber.Int64(), "runtime", time.Since(start))
	}(time.Now())

	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get block %v from db got %v", blockNumber.Int64(), err.Error())
	}

	traces, err := s.replayBlock(ctx, block, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot trace block %v got %v", blockNumber.Int64(), err.Error())
	}

	return traces, nil
}

// Get - trace_get function returns transaction traces on specified index position of the traces
// If index is nil, then just root trace is returned
func (s *PublicTxTraceAPI) Get(ctx context.Context, hash common.Hash, traceIndex []hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	defer func(start time.Time) {
		log.Debug("Executing trace_get call finished", "txHash", hash.String(), "index", traceIndex, "runtime", time.Since(start))
	}(time.Now())
	return s.traceTxHash(ctx, hash, &traceIndex)
}

// traceTxHash looks for a block of this transaction hash and trace it
func (s *PublicTxTraceAPI) traceTxHash(ctx context.Context, hash common.Hash, traceIndex *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	tx, blockNumber, _, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction %s: %v", hash.String(), err)
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction %s not found", hash.String())
	}
	blkNr := rpc.BlockNumber(blockNumber)
	block, err := s.b.BlockByNumber(ctx, blkNr)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", blkNr, err.Error())
	}

	return s.replayBlock(ctx, block, &hash, traceIndex)
}

// Replays block and returns traces acording to parameters
//
// txHash
//   - if is nil, all transaction traces in the block are collected
//   - is value, then only trace for that transaction is returned
//
// traceIndex - when specified, then only trace on that index is returned
func (s *PublicTxTraceAPI) replayBlock(ctx context.Context, block *evmcore.EvmBlock, txHash *common.Hash, traceIndex *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {

	if block == nil {
		return nil, fmt.Errorf("invalid block for tracing")
	}

	if block.NumberU64() == 0 {
		return nil, fmt.Errorf("genesis block is not traceable")
	}

	blockNumber := block.Number.Int64()
	parentBlockNr := rpc.BlockNumber(blockNumber - 1)
	callTrace := txtrace.CallTrace{
		Actions: make([]txtrace.ActionTrace, 0),
	}

	signer := gsignercache.Wrap(types.MakeSigner(s.b.ChainConfig(), block.Number, uint64(block.Time.Unix())))

	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHash{BlockNumber: &parentBlockNr})
	if err != nil {
		return nil, fmt.Errorf("cannot get state for block %v, error: %v", block.NumberU64(), err.Error())
	}
	defer state.Release()

	receipts, err := s.b.GetReceiptsByNumber(ctx, rpc.BlockNumber(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("cannot get receipts for block %v, error: %v", block.NumberU64(), err.Error())
	}

	// loop thru all transactions in the block and process them
	for i, tx := range block.Transactions {

		// replay only needed transaction if specified
		if txHash == nil || *txHash == tx.Hash() {

			msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
			if err != nil {
				return nil, fmt.Errorf("cannot get message from transaction %s, error %s", tx.Hash().String(), err)
			}

			if len(receipts) <= i || receipts[i] == nil {
				return nil, fmt.Errorf("no receipt found for transaction %s", tx.Hash().String())
			}

			txTraces, err := s.traceTx(ctx, s.b, block.Header(), msg, state, block, tx, uint64(receipts[i].TransactionIndex), receipts[i].Status)
			if err != nil {
				return nil, fmt.Errorf("cannot get transaction trace for transaction %s, error %s", tx.Hash().String(), err)
			} else {
				callTrace.AddTraces(txTraces, traceIndex)
			}

			// already replayed specified transaction so end loop
			if txHash != nil {
				break
			}

		} else {

			// Replay transaction without tracing to prepare state for next transaction
			log.Debug("Replaying transaction without trace", "txHash", tx.Hash().String())
			msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
			if err != nil {
				return nil, fmt.Errorf("cannot get message from transaction %s, error %s", tx.Hash().String(), err)
			}

			state.SetTxContext(tx.Hash(), i)
			vmConfig := opera.DefaultVMConfig
			vmConfig.NoBaseFee = true
			vmConfig.Tracer = nil

			vmenv, _, err := s.b.GetEVM(ctx, state, block.Header(), &vmConfig, nil)
			if err != nil {
				return nil, fmt.Errorf("cannot initialize vm for transaction %s, error: %s", tx.Hash().String(), err.Error())
			}

			if vmenv.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
				evmcore.ProcessParentBlockHash(block.ParentHash, vmenv)
			}

			res, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.GasLimit))
			failed := false
			if err != nil {
				failed = true
				log.Error("Cannot replay transaction", "txHash", tx.Hash().String(), "err", err.Error())
			}

			if res != nil && res.Err != nil {
				failed = true
				log.Debug("Error replaying transaction", "txHash", tx.Hash().String(), "err", res.Err.Error())
			}

			state.EndTransaction()

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
		} else {
			return getEmptyBlockTrace(block.Hash, *block.Number), nil
		}
	}

	return &callTrace.Actions, nil
}

// traceTx trace transaction with EVM replay and return processed result
func (s *PublicTxTraceAPI) traceTx(
	ctx context.Context, b Backend, header *evmcore.EvmHeader, msg *core.Message,
	state state.StateDB, block *evmcore.EvmBlock, tx *types.Transaction, index uint64,
	status uint64) (*[]txtrace.ActionTrace, error) {

	// Providing default config with tracer
	cfg := opera.DefaultVMConfig
	txTracer := txtrace.NewTraceStructLogger(block, uint(index))
	cfg.Tracer = txTracer.Hooks()
	cfg.NoBaseFee = true

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var timeout time.Duration = 5 * time.Second
	if s.b.RPCEVMTimeout() > 0 {
		timeout = s.b.RPCEVMTimeout()
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)

	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	vmenv, _, err := b.GetEVM(ctx, state, header, &cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize vm for transaction %s, error: %s", tx.Hash().String(), err.Error())
	}

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		vmenv.Cancel()
	}()

	// Setup the gas pool and stateDB
	gp := new(core.GasPool).AddGas(msg.GasLimit)
	state.SetTxContext(tx.Hash(), int(index))
	resultReceipt, err := evmcore.ApplyTransactionWithEVM(msg, b.ChainConfig(), gp, state, header.Number, block.Hash, tx, &index, vmenv)

	traceActions := txTracer.GetResult()
	state.EndTransaction()

	// err is error occured before EVM execution
	if err != nil {
		errTrace := txtrace.GetErrorTraceFromMsg(msg, block.Hash, *block.Number, tx.Hash(), index, err)
		at := make([]txtrace.ActionTrace, 0)
		at = append(at, *errTrace)
		// check correct replay state
		if status == 1 {
			return nil, fmt.Errorf("invalid transaction replay state at %s", tx.Hash().String())
		}
		return &at, nil
	}
	// If the timer caused an abort, return an appropriate error message
	if vmenv.Cancelled() {
		return nil, fmt.Errorf("EVM was cancelled when replaying tx")
	}

	// check correct replay state
	if status != resultReceipt.Status {
		return nil, fmt.Errorf("invalid transaction replay state at %s, want %v but got %v", tx.Hash().String(), status, resultReceipt.Status)
	}
	return traceActions, nil
}

// getEmptyBlockTrace returns trace for empty block
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

// FilterArgs represents the arguments for specifiing trace targets
type FilterArgs struct {
	FromAddress *[]common.Address      `json:"fromAddress"`
	ToAddress   *[]common.Address      `json:"toAddress"`
	FromBlock   *rpc.BlockNumberOrHash `json:"fromBlock"`
	ToBlock     *rpc.BlockNumberOrHash `json:"toBlock"`
	After       uint                   `json:"after"`
	Count       uint                   `json:"count"`
}

// Filter is function for trace_filter rpc call
func (s *PublicTxTraceAPI) Filter(ctx context.Context, args FilterArgs) (json.RawMessage, error) {
	// add log after execution
	defer func(start time.Time) {
		data := getLogData(args, start)
		log.Debug("Executing trace_filter call finished", data...)
	}(time.Now())

	if args.Count == 0 && args.After == 0 {
		// count and order of traces doesn't matter so filter blocks in parallel
		return filterBlocksInParallel(ctx, s, args)
	} else {
		// filter blocks in series
		return filterBlocks(ctx, s, args)
	}
}

// Filter specified block range in series
func filterBlocks(ctx context.Context, s *PublicTxTraceAPI, args FilterArgs) (json.RawMessage, error) {

	var traceAdded, traceCount uint

	// resultBuffer is buffer for collecting result traces
	resultBuffer, err := NewJsonResultBuffer(s.maxResponseSize)
	if err != nil {
		return nil, err
	}

	// parse arguments
	fromBlock, toBlock, fromAddresses, toAddresses := parseFilterArguments(s.b, args)

	// loop trhu all blocks
	for i := fromBlock; i <= toBlock; i++ {
		traces, err := getTracesForBlock(s, ctx, i, fromAddresses, toAddresses)
		if err != nil {
			return nil, err
		}

		// check if traces have to be added
		for _, trace := range traces {

			if traceCount >= args.After {
				err := resultBuffer.AddObject(&trace)
				if err != nil {
					return nil, err
				}
				traceAdded++
			}
			if traceAdded >= args.Count {
				return resultBuffer.GetResult()
			}
			traceCount++
		}

		// when context ended return error
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return resultBuffer.GetResult()
}

// Filter specified block range in parallel
func filterBlocksInParallel(ctx context.Context, s *PublicTxTraceAPI, args FilterArgs) (json.RawMessage, error) {

	// resultBuffer is buffer for collecting result traces
	resultBuffer, err := NewJsonResultBuffer(s.maxResponseSize)
	if err != nil {
		return nil, err
	}
	// parse arguments
	fromBlock, toBlock, fromAddresses, toAddresses := parseFilterArguments(s.b, args)
	// add context cancel function
	ctx, cancelFunc := context.WithCancelCause(ctx)

	// number of workers
	workerCount := runtime.NumCPU()

	blocks := make(chan rpc.BlockNumber, 1)
	results := make(chan traceWorkerResult, 1)

	// make goroutine for results processing
	var wgResult sync.WaitGroup
	wgResult.Add(1)
	go func() {
		defer wgResult.Done()
		for {
			select {
			case res, ok := <-results:
				if !ok {
					return
				}
				if res.err != nil {
					cancelFunc(res.err)
				} else {
					for _, trace := range res.trace {
						err := resultBuffer.AddObject(&trace)
						if err != nil {
							cancelFunc(err)
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// make workers to process blocks
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			replayBlockWorker(s, ctx, blocks, results, fromAddresses, toAddresses)
		}()
	}

	// fill blocks channel with blocks to process
	addBlocksForProcessing(ctx, fromBlock, toBlock, blocks)

	// wait for workers to be done and then close results channel
	wg.Wait()
	close(results)
	wgResult.Wait()

	// check if context expired or had another error
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout when replaying tx")
		} else {
			return nil, context.Cause(ctx)
		}
	}
	return resultBuffer.GetResult()
}

// Fills blocks into provided channel for processing and close the channel in the end
// or if the context was canceled
func addBlocksForProcessing(ctx context.Context, fromBlock rpc.BlockNumber, toBlock rpc.BlockNumber, blocks chan<- rpc.BlockNumber) {
	defer close(blocks)
	for i := fromBlock; i <= toBlock; i++ {
		select {
		case blocks <- i:
		case <-ctx.Done():
			return
		}
	}
}

// Parses rpc call arguments
func parseFilterArguments(b Backend, args FilterArgs) (fromBlock rpc.BlockNumber, toBlock rpc.BlockNumber, fromAddresses map[common.Address]struct{}, toAddresses map[common.Address]struct{}) {

	blockHead := rpc.BlockNumber(b.CurrentBlock().NumberU64())

	if args.FromBlock != nil {
		fromBlock = *args.FromBlock.BlockNumber
		if fromBlock == rpc.LatestBlockNumber || fromBlock == rpc.PendingBlockNumber {
			fromBlock = blockHead
		}
	}

	if args.ToBlock != nil {
		toBlock = *args.ToBlock.BlockNumber
		if toBlock == rpc.LatestBlockNumber || toBlock == rpc.PendingBlockNumber {
			toBlock = blockHead
		}
	} else {
		toBlock = blockHead
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
	return fromBlock, toBlock, fromAddresses, toAddresses
}

type traceWorkerResult struct {
	trace []txtrace.ActionTrace
	err   error
}

// Worker for replaying blocks in parallel and filter replayed traces
func replayBlockWorker(
	s *PublicTxTraceAPI,
	ctx context.Context,
	blocks <-chan rpc.BlockNumber,
	results chan<- traceWorkerResult,
	fromAddresses map[common.Address]struct{},
	toAddresses map[common.Address]struct{}) {

	for i := range blocks {

		// check context before block processing
		// error is not propagated as it is checked
		// from context in the main goroutine
		if ctx.Err() != nil {
			return
		}

		traces, err := getTracesForBlock(s, ctx, i, fromAddresses, toAddresses)
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

// Replay block transactions and filter out useable traces
func getTracesForBlock(
	s *PublicTxTraceAPI,
	ctx context.Context,
	blockNumber rpc.BlockNumber,
	fromAddresses map[common.Address]struct{},
	toAddresses map[common.Address]struct{},
) (
	[]txtrace.ActionTrace,
	error,
) {
	resultTraces := make([]txtrace.ActionTrace, 0)

	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", blockNumber.Int64(), err.Error())
	}

	if block == nil {
		return nil, fmt.Errorf("cannot get block from db %v", blockNumber.Int64())
	}

	if block.Transactions.Len() == 0 {
		return resultTraces, nil
	}

	// when block has any transaction, then process it
	traces, err := s.replayBlock(ctx, block, nil, nil)
	if err != nil {
		return nil, err
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

// Check if from or to address is contained in the map
func containsAddress(addressFrom *common.Address, addressTo *common.Address, fromAddresses map[common.Address]struct{}, toAddresses map[common.Address]struct{}) bool {

	if len(fromAddresses) > 0 {
		if addressFrom == nil {
			return false
		} else {
			if _, ok := fromAddresses[*addressFrom]; !ok {
				return false
			}
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

// Creates log record according to request arguments
func getLogData(args FilterArgs, start time.Time) []interface{} {

	var data []interface{}

	if args.FromBlock != nil {
		data = append(data, "fromBlock", args.FromBlock.BlockNumber.Int64())
	}

	if args.ToBlock != nil {
		data = append(data, "toBlock", args.ToBlock.BlockNumber.Int64())
	}

	if args.FromAddress != nil {
		adresses := make([]string, 0)
		for _, addr := range *args.FromAddress {
			adresses = append(adresses, addr.String())
		}
		data = append(data, "fromAddr", adresses)
	}

	if args.ToAddress != nil {
		adresses := make([]string, 0)
		for _, addr := range *args.ToAddress {
			adresses = append(adresses, addr.String())
		}
		data = append(data, "toAddr", adresses)
	}
	data = append(data, "time", time.Since(start))
	return data
}
