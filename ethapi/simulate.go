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

package ethapi

// Implements eth_simulateV1, adapted from go-ethereum's
// internal/ethapi/simulate.go to work with sonic's Backend interface and types.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	interState "github.com/0xsoniclabs/sonic/inter/state"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/inter"
)

const (
	// maxSimulateBlocks is the maximum number of blocks that can be simulated
	// in a single request.
	maxSimulateBlocks = 256

	// timestampIncrement is the default increment between block timestamps.
	timestampIncrement = 12

	errCodeReverted                = -32000
	errCodeVMError                 = -32015
	errCodeInvalidParams           = -32602
	errCodeInternalError           = -32603
	errCodeNonceTooHigh            = -38011
	errCodeNonceTooLow             = -38010
	errCodeIntrinsicGas            = -38013
	errCodeInsufficientFunds       = -38014
	errCodeBlockGasLimitReached    = -38015
	errCodeBlockNumberInvalid      = -38020
	errCodeBlockTimestampInvalid   = -38021
	errCodeSenderIsNotEOA          = -38024
	errCodeMaxInitCodeSizeExceeded = -38025
	errCodeClientLimitExceeded     = -38026
)

type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
}

type simInvalidParamsError struct{ message string }

func (e *simInvalidParamsError) Error() string  { return e.message }
func (e *simInvalidParamsError) ErrorCode() int { return errCodeInvalidParams }

type simClientLimitExceededError struct{ message string }

func (e *simClientLimitExceededError) Error() string  { return e.message }
func (e *simClientLimitExceededError) ErrorCode() int { return errCodeClientLimitExceeded }

type simInvalidBlockNumberError struct{ message string }

func (e *simInvalidBlockNumberError) Error() string  { return e.message }
func (e *simInvalidBlockNumberError) ErrorCode() int { return errCodeBlockNumberInvalid }

type simInvalidBlockTimestampError struct{ message string }

func (e *simInvalidBlockTimestampError) Error() string  { return e.message }
func (e *simInvalidBlockTimestampError) ErrorCode() int { return errCodeBlockTimestampInvalid }

type simBlockGasLimitReachedError struct{ message string }

func (e *simBlockGasLimitReachedError) Error() string  { return e.message }
func (e *simBlockGasLimitReachedError) ErrorCode() int { return errCodeBlockGasLimitReached }

type simInvalidTxError struct {
	Message string
	Code    int
}

func (e *simInvalidTxError) Error() string  { return e.Message }
func (e *simInvalidTxError) ErrorCode() int { return e.Code }

func simTxValidationError(err error) *simInvalidTxError {
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeSenderIsNotEOA}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeIntrinsicGas}
	case errors.Is(err, core.ErrMaxInitCodeSizeExceeded):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeMaxInitCodeSizeExceeded}
	default:
		return &simInvalidTxError{
			Message: err.Error(),
			Code:    errCodeInternalError,
		}
	}
}

var (
	// keccak256("Transfer(address,address,uint256)")
	simTransferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// ERC-7528 canonical address for native ETH/Sonic events
	simTransferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

// simTracer collects logs
type simTracer struct {
	logs           []*types.Log
	count          int
	traceTransfers bool
	blockNumber    uint64
	blockTimestamp uint64
	blockHash      common.Hash
	txHash         common.Hash
	txIdx          uint
}

func newSimTracer(traceTransfers bool, blockNumber, blockTimestamp uint64, blockHash, txHash common.Hash, txIndex uint) *simTracer {
	return &simTracer{
		traceTransfers: traceTransfers,
		blockNumber:    blockNumber,
		blockTimestamp: blockTimestamp,
		blockHash:      blockHash,
		txHash:         txHash,
		txIdx:          txIndex,
	}
}

func (t *simTracer) Hooks() *tracing.Hooks {
	if !t.traceTransfers {
		return nil
	}
	return &tracing.Hooks{
		OnEnter: t.onEnter,
		OnExit:  t.onExit,
	}
}

func (t *simTracer) onEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.traceTransfers &&
		vm.OpCode(typ) != vm.DELEGATECALL &&
		value != nil && value.Cmp(common.Big0) > 0 {

		t.captureTransfer(from, to, value)
	}
}

func (t *simTracer) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth == 0 && reverted {
		t.logs = nil
		t.count = 0
		return
	}
}

func (t *simTracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	t.logs = append(t.logs, &types.Log{
		Address:        address,
		Topics:         topics,
		Data:           data,
		BlockNumber:    t.blockNumber,
		BlockTimestamp: t.blockTimestamp,
		BlockHash:      t.blockHash,
		TxHash:         t.txHash,
		TxIndex:        t.txIdx,
		Index:          uint(t.count),
	})
	t.count++
}

func (t *simTracer) captureTransfer(from, to common.Address, value *big.Int) {
	if !t.traceTransfers {
		return
	}
	topics := []common.Hash{
		simTransferTopic,
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	t.captureLog(simTransferAddress, topics, common.BigToHash(value).Bytes())
}

func (t *simTracer) reset(txHash common.Hash, txIdx uint) {
	t.logs = nil
	t.txHash = txHash
	t.txIdx = txIdx
}

func (t *simTracer) Logs() []*types.Log {
	return t.logs
}

// simBlockOverrides contains block header fields that can be overridden per
// simulated block.
type simBlockOverrides struct {
	Number        *hexutil.Big       `json:"blockNumber"`
	Time          *hexutil.Uint64    `json:"timestamp"`
	GasLimit      *hexutil.Uint64    `json:"gasLimit"`
	FeeRecipient  *common.Address    `json:"feeRecipient"`
	PrevRandao    *common.Hash       `json:"prevRandao"`
	BaseFeePerGas *hexutil.Big       `json:"baseFeePerGas"`
	Withdrawals   *types.Withdrawals `json:"withdrawals"`
}

// makeEvmHeader creates a new EvmHeader based on a template and applies the
// overrides. Fields not provided in the overrides remain from the template.
func (o *simBlockOverrides) makeEvmHeader(template *evmcore.EvmHeader) *evmcore.EvmHeader {
	h := *template // copy
	if o == nil {
		return &h
	}
	if o.Number != nil {
		h.Number = new(big.Int).Set(o.Number.ToInt())
	}
	if o.Time != nil {
		h.Time = inter.FromUnix(int64(*o.Time))
	}
	if o.GasLimit != nil {
		h.GasLimit = uint64(*o.GasLimit)
	}
	if o.FeeRecipient != nil {
		h.Coinbase = *o.FeeRecipient
	}
	if o.PrevRandao != nil {
		h.PrevRandao = *o.PrevRandao
	}
	if o.BaseFeePerGas != nil {
		h.BaseFee = o.BaseFeePerGas.ToInt()
	}
	return &h
}

// simBlock is a batch of calls to be simulated sequentially.
type simBlock struct {
	BlockOverrides *simBlockOverrides `json:"blockOverrides"`
	StateOverrides *StateOverride     `json:"stateOverrides"`
	Calls          []TransactionArgs  `json:"calls"`
}

// simCallResult is the result of a single simulated call.
type simCallResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

func (r *simCallResult) MarshalJSON() ([]byte, error) {
	type alias simCallResult
	// Ensure logs serialize as [] instead of null when empty.
	if r.Logs == nil {
		r.Logs = []*types.Log{}
	}
	return json.Marshal((*alias)(r))
}

// simBlockResult is the result of a simulated block.
type simBlockResult struct {
	block   *evmcore.EvmBlock
	calls   []simCallResult
	chainId *big.Int
	fullTx  bool
	senders map[common.Hash]common.Address
}

func (r *simBlockResult) MarshalJSON() ([]byte, error) {
	blockJson, err := RPCMarshalBlock(r.block, nil, true, r.fullTx, r.chainId)
	if err != nil {
		return nil, err
	}

	// Fix up the "from" field for full transaction objects.
	if r.fullTx && blockJson.Txs != nil {
		for _, txRaw := range blockJson.Txs {
			if tx, ok := txRaw.(*RPCTransaction); ok {
				if sender, found := r.senders[tx.Hash]; found {
					tx.From = sender
				}
			}
		}
	}

	// Marshal the block to a map to inject the "calls" field.
	blockBytes, err := json.Marshal(blockJson)
	if err != nil {
		return nil, err
	}
	var blockMap map[string]json.RawMessage
	if err := json.Unmarshal(blockBytes, &blockMap); err != nil {
		return nil, err
	}

	// Ensure calls marshal as [] when empty.
	calls := r.calls
	if calls == nil {
		calls = []simCallResult{}
	}
	callsBytes, err := json.Marshal(calls)
	if err != nil {
		return nil, err
	}
	blockMap["calls"] = callsBytes

	return json.Marshal(blockMap)
}

// simOpts are the inputs to eth_simulateV1.
type simOpts struct {
	BlockStateCalls        []simBlock `json:"blockStateCalls"`
	TraceTransfers         bool       `json:"traceTransfers"`
	Validation             bool       `json:"validation"`
	ReturnFullTransactions bool       `json:"returnFullTransactions"`
}

// simDummyChain implements evmcore.DummyChain so the EVM's BLOCKHASH opcode
// can look up headers from the real chain or from already-simulated blocks.
type simDummyChain struct {
	ctx     context.Context
	backend Backend
	base    *evmcore.EvmHeader
	headers []*evmcore.EvmHeader // previously simulated headers in this request
}

func (c *simDummyChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	// Check the base (real) block.
	if c.base.Number.Uint64() == number && c.base.Hash == hash {
		return c.base
	}
	// Check already-assembled simulated headers.
	for _, h := range c.headers {
		if h != nil && h.Number.Uint64() == number && h.Hash == hash {
			return h
		}
	}
	// Fall back to the real chain.
	header, err := c.backend.HeaderByNumber(c.ctx, rpc.BlockNumber(number))
	if err != nil || header == nil || header.Hash != hash {
		return nil
	}
	return header
}

// simulator is a stateful object that simulates a series of blocks.
// It is NOT safe for concurrent use.
type simulator struct {
	b              Backend
	state          interState.StateDB
	base           *evmcore.EvmHeader
	chainConfig    *params.ChainConfig
	gp             *core.GasPool
	traceTransfers bool
	validate       bool
	fullTx         bool
}

// execute runs the simulation over all provided blocks and returns the results.
func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]*simBlockResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var (
		cancel  context.CancelFunc
		timeout = sim.b.RPCEVMTimeout()
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var err error
	blocks, err = sim.sanitizeChain(blocks)
	if err != nil {
		return nil, err
	}
	headers, err := sim.makeHeaders(blocks)
	if err != nil {
		return nil, err
	}

	var (
		results = make([]*simBlockResult, len(blocks))
		parent  = sim.base
	)
	for bi, block := range blocks {
		evmBlock, callResults, senders, err := sim.processBlock(ctx, &block, headers[bi], parent, headers[:bi], timeout)
		if err != nil {
			return nil, err
		}
		// Update the header slot with the assembled block's header (including hash).
		headers[bi] = evmBlock.Header()
		results[bi] = &simBlockResult{
			block:   evmBlock,
			calls:   callResults,
			chainId: sim.chainConfig.ChainID,
			fullTx:  sim.fullTx,
			senders: senders,
		}
		parent = evmBlock.Header()
	}
	return results, nil
}

// processBlock simulates a single block and returns the assembled EvmBlock,
// per-call results, and a map of tx-hash to sender for full-transaction output.
func (sim *simulator) processBlock(
	ctx context.Context,
	block *simBlock,
	header *evmcore.EvmHeader,
	parent *evmcore.EvmHeader,
	prevHeaders []*evmcore.EvmHeader,
	timeout time.Duration,
) (*evmcore.EvmBlock, []simCallResult, map[common.Hash]common.Address, error) {

	// Resolve base fee.
	header.ParentHash = parent.Hash
	if header.BaseFee == nil {
		if sim.validate && parent.BaseFee != nil {
			header.BaseFee = new(big.Int).Set(parent.BaseFee)
		} else {
			header.BaseFee = big.NewInt(0)
		}
	}

	// Build block context.
	chain := &simDummyChain{ctx: ctx, backend: sim.b, base: sim.base, headers: prevHeaders}
	blockContext := evmcore.NewEVMBlockContext(header, chain, nil)

	precompiles := sim.activePrecompiles(sim.base)

	if err := sim.applyStateOverrides(block.StateOverrides, precompiles, sim.state); err != nil {
		return nil, nil, nil, err
	}

	var (
		gasUsed     uint64
		txes        = make([]*types.Transaction, len(block.Calls))
		callResults = make([]simCallResult, len(block.Calls))
		senders     = make(map[common.Hash]common.Address)
		vmConfig    = vm.Config{
			NoBaseFee: !sim.validate,
		}
	)

	tracer := newSimTracer(sim.traceTransfers, blockContext.BlockNumber.Uint64(), blockContext.Time,
		common.Hash{}, common.Hash{}, 0)
	if hooks := tracer.Hooks(); hooks != nil {
		vmConfig.Tracer = hooks
	}

	evm := vm.NewEVM(blockContext, sim.state, sim.chainConfig, vmConfig)
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}

	// EIP-2935: store parent block hash in history contract.
	if sim.chainConfig.IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, sim.state)
	}

	var allContractLogs []*types.Log

	for i, call := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}

		if err := sim.sanitizeCall(&call, sim.state, header, &gasUsed); err != nil {
			return nil, nil, nil, err
		}

		tx := call.ToTransaction()
		txHash := tx.Hash()
		txes[i] = tx
		senders[txHash] = call.from()

		tracer.reset(txHash, uint(i))
		sim.state.SetTxContext(txHash, i)

		msg, err := call.ToMessage(sim.gp.Gas(), header.BaseFee, log.Root())
		if err != nil {
			return nil, nil, nil, simTxValidationError(err)
		}
		msg.SkipNonceChecks = !sim.validate
		msg.SkipTransactionChecks = !sim.validate

		result, applyErr := applySimMessage(ctx, evm, msg, timeout, sim.gp)
		if applyErr != nil {
			return nil, nil, nil, simTxValidationError(applyErr)
		}

		contractLogs := sim.state.GetLogs(txHash, common.Hash{})
		allContractLogs = append(allContractLogs, contractLogs...)

		sim.state.EndTransaction()

		gasUsed += result.UsedGas

		// Build the per-call result.
		// Use result.ReturnData directly so that revert payloads are included
		callRes := simCallResult{
			ReturnValue: result.ReturnData,
			GasUsed:     hexutil.Uint64(result.UsedGas),
		}

		// Combine contract logs with transfer pseudo-logs (if traceTransfers).
		var txLogs []*types.Log
		if sim.traceTransfers {
			txLogs = append(txLogs, tracer.Logs()...)
		}
		txLogs = append(txLogs, contractLogs...)
		callRes.Logs = txLogs

		if result.Failed() {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusFailed)
			if errors.Is(result.Err, vm.ErrExecutionReverted) {
				revertErr := newRevertError(result)
				callRes.Error = &callError{
					Message: revertErr.Error(),
					Code:    errCodeReverted,
					Data:    revertErr.ErrorData().(string),
				}
			} else {
				callRes.Error = &callError{Message: result.Err.Error(), Code: errCodeVMError}
			}
		} else {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
		}
		callResults[i] = callRes
	}

	header.GasUsed = gasUsed
	header.Root = sim.state.GetStateHash()

	if len(txes) == 0 {
		header.TxHash = types.EmptyRootHash
	} else {
		header.TxHash = types.DeriveSha(types.Transactions(txes), trie.NewStackTrie(nil))
	}
	evmBlock := evmcore.NewEvmBlock(header, txes)
	ethHdr := evmBlock.EthHeader()
	blockHash := ethHdr.Hash()
	evmBlock.Hash = blockHash

	// Repair all log entries with the now-known block hash.
	repairSimLogs(callResults, allContractLogs, blockHash)

	return evmBlock, callResults, senders, nil
}

// repairSimLogs updates the BlockHash field in all collected logs now that the
// block hash is known. It updates both the logs stored in callResults and the
// shared allContractLogs slice (which shares pointers with callResults).
func repairSimLogs(calls []simCallResult, allLogs []*types.Log, blockHash common.Hash) {
	for _, l := range allLogs {
		l.BlockHash = blockHash
	}
	// Also repair transfer pseudo-logs, which are only in callResults.
	for i := range calls {
		for j := range calls[i].Logs {
			calls[i].Logs[j].BlockHash = blockHash
		}
	}
}

// applySimMessage executes a message on the EVM, cancelling on context
// expiry, and wraps timeout errors appropriately.
func applySimMessage(ctx context.Context, evm *vm.EVM, msg *core.Message, timeout time.Duration, gp *core.GasPool) (*core.ExecutionResult, error) {
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()
	result, err := core.ApplyMessage(evm, msg, gp)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

// applyStateOverrides applies account overrides to the state before a block's transactions run
func (sim *simulator) applyStateOverrides(overrides *StateOverride, precompiles vm.PrecompiledContracts, state interState.StateDB) error {
	if overrides == nil {
		return nil
	}
	for addr, account := range *overrides {
		// If this address is a precompile, remove it (the account override takes
		// precedence and the code/state can be replaced below).
		delete(precompiles, addr)

		if account.Nonce != nil {
			state.SetNonce(addr, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		if account.Code != nil {
			state.SetCode(addr, *account.Code, tracing.CodeChangeUnspecified)
		}
		if account.Balance != nil && *account.Balance != nil {
			state.SetBalance(addr, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		if account.State != nil {
			state.SetStorage(addr, *account.State)
		}
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				state.SetState(addr, key, value)
			}
		}
	}
	// Finalise the overrides as if they were a transaction.
	state.Finalise(false)
	return nil
}

// sanitizeCall fills in defaults for a single simulated call.
func (sim *simulator) sanitizeCall(call *TransactionArgs, state interState.StateDB, header *evmcore.EvmHeader, gasUsed *uint64) error {
	// Default nonce to the sender's current nonce in state.
	if call.Nonce == nil {
		nonce := state.GetNonce(call.from())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	// Default gas to the remaining block gas.
	if call.Gas == nil {
		remaining := header.GasLimit - *gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	if *gasUsed+uint64(*call.Gas) > header.GasLimit {
		return &simBlockGasLimitReachedError{
			fmt.Sprintf("block gas limit reached: %d >= %d", *gasUsed+uint64(*call.Gas), header.GasLimit),
		}
	}
	// Set price-related defaults (no-backend equivalent of setDefaults).
	if err := sim.setCallPriceDefaults(call, header.BaseFee); err != nil {
		return err
	}
	return nil
}

// setCallPriceDefaults fills in gas-price fields based on the simulated base fee.
func (sim *simulator) setCallPriceDefaults(call *TransactionArgs, baseFee *big.Int) error {
	if call.GasPrice != nil && (call.MaxFeePerGas != nil || call.MaxPriorityFeePerGas != nil) {
		return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	if call.ChainID == nil {
		call.ChainID = (*hexutil.Big)(sim.chainConfig.ChainID)
	}
	if call.Value == nil {
		call.Value = new(hexutil.Big)
	}
	if baseFee != nil && call.GasPrice == nil {
		// EIP-1559 style: default to zero fees so calls succeed without funds.
		if call.MaxFeePerGas == nil {
			call.MaxFeePerGas = (*hexutil.Big)(new(big.Int))
		}
		if call.MaxPriorityFeePerGas == nil {
			call.MaxPriorityFeePerGas = (*hexutil.Big)(new(big.Int))
		}
	} else if call.GasPrice == nil {
		call.GasPrice = (*hexutil.Big)(new(big.Int))
	}
	return nil
}

// activePrecompiles returns the precompiled contracts active at the base block.
func (sim *simulator) activePrecompiles(base *evmcore.EvmHeader) vm.PrecompiledContracts {
	rules := sim.chainConfig.Rules(base.Number, base.PrevRandao != (common.Hash{}), uint64(base.Time.Unix()))
	return vm.ActivePrecompiledContracts(rules)
}

// sanitizeChain validates and fills gaps in the sequence of simulated blocks.
// Blocks must be in strictly increasing order; gaps are filled with empty blocks.
func (sim *simulator) sanitizeChain(blocks []simBlock) ([]simBlock, error) {
	var (
		res           = make([]simBlock, 0, len(blocks))
		base          = sim.base
		prevNumber    = new(big.Int).Set(base.Number)
		prevTimestamp = uint64(base.Time.Unix())
	)
	for _, block := range blocks {
		if block.BlockOverrides == nil {
			block.BlockOverrides = new(simBlockOverrides)
		}
		if block.BlockOverrides.Number == nil {
			n := new(big.Int).Add(prevNumber, big.NewInt(1))
			block.BlockOverrides.Number = (*hexutil.Big)(n)
		}
		if block.BlockOverrides.Withdrawals == nil {
			block.BlockOverrides.Withdrawals = &types.Withdrawals{}
		}

		diff := new(big.Int).Sub(block.BlockOverrides.Number.ToInt(), prevNumber)
		if diff.Cmp(common.Big0) <= 0 {
			return nil, &simInvalidBlockNumberError{
				fmt.Sprintf("block numbers must be in order: %d <= %d",
					block.BlockOverrides.Number.ToInt().Uint64(), prevNumber.Uint64()),
			}
		}
		if total := new(big.Int).Sub(block.BlockOverrides.Number.ToInt(), base.Number); total.Cmp(big.NewInt(maxSimulateBlocks)) > 0 {
			return nil, &simClientLimitExceededError{message: "too many blocks"}
		}

		// Fill any gap with empty blocks.
		if diff.Cmp(big.NewInt(1)) > 0 {
			gap := new(big.Int).Sub(diff, big.NewInt(1))
			for i := uint64(0); i < gap.Uint64(); i++ {
				n := new(big.Int).Add(prevNumber, big.NewInt(int64(i+1)))
				t := prevTimestamp + timestampIncrement
				res = append(res, simBlock{
					BlockOverrides: &simBlockOverrides{
						Number:      (*hexutil.Big)(n),
						Time:        (*hexutil.Uint64)(&t),
						Withdrawals: &types.Withdrawals{},
					},
				})
				prevTimestamp = t
			}
		}

		prevNumber = new(big.Int).Set(block.BlockOverrides.Number.ToInt())
		if block.BlockOverrides.Time == nil {
			t := prevTimestamp + timestampIncrement
			block.BlockOverrides.Time = (*hexutil.Uint64)(&t)
			prevTimestamp = t
		} else {
			t := uint64(*block.BlockOverrides.Time)
			if t <= prevTimestamp {
				return nil, &simInvalidBlockTimestampError{
					fmt.Sprintf("block timestamps must be in order: %d <= %d", t, prevTimestamp),
				}
			}
			prevTimestamp = t
		}

		res = append(res, block)
	}
	return res, nil
}

// makeHeaders creates preliminary EvmHeader objects for each simulated block.
// Some fields (GasUsed, Root, TxHash, Hash) are filled in later, after execution.
func (sim *simulator) makeHeaders(blocks []simBlock) ([]*evmcore.EvmHeader, error) {
	res := make([]*evmcore.EvmHeader, len(blocks))
	prev := sim.base

	for bi, block := range blocks {
		if block.BlockOverrides == nil || block.BlockOverrides.Number == nil {
			return nil, errors.New("empty block number")
		}
		overrides := block.BlockOverrides
		number := overrides.Number.ToInt()
		timestamp := uint64(*overrides.Time)

		// Determine whether to set a withdrawals hash (Shanghai+).
		var withdrawalsHash *common.Hash
		if sim.chainConfig.IsShanghai(number, timestamp) {
			h := types.EmptyWithdrawalsHash
			withdrawalsHash = &h
		}

		// Template header inheriting fields from the previous block.
		template := &evmcore.EvmHeader{
			Number:          number,
			Time:            inter.FromUnix(int64(timestamp)),
			Coinbase:        prev.Coinbase,
			GasLimit:        prev.GasLimit,
			BaseFee:         prev.BaseFee,
			PrevRandao:      prev.PrevRandao,
			WithdrawalsHash: withdrawalsHash,
		}
		// Apply user overrides.
		header := overrides.makeEvmHeader(template)
		res[bi] = header
		prev = header
	}
	return res, nil
}

// SimulateV1 executes a series of transactions on top of a base state.
// Transactions are packed into simulated blocks; each block can override header
// fields and apply state overrides before execution. The function never commits
// anything to the blockchain.
func (s *PublicBlockChainAPI) SimulateV1(ctx context.Context, opts simOpts, blockNrOrHash *rpc.BlockNumberOrHash) ([]*simBlockResult, error) {
	if len(opts.BlockStateCalls) == 0 {
		return nil, &simInvalidParamsError{message: "empty input"}
	}
	if len(opts.BlockStateCalls) > maxSimulateBlocks {
		return nil, &simClientLimitExceededError{message: "too many blocks"}
	}

	if blockNrOrHash == nil {
		n := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &n
	}

	state, base, err := s.b.StateAndBlockByNumberOrHash(ctx, *blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()

	gasCap := s.b.RPCGasCap()
	if gasCap == 0 {
		gasCap = ^uint64(0) // MaxUint64
	}

	chainConfig := s.b.ChainConfig(idx.Block(base.Number.Uint64()))

	sim := &simulator{
		b:              s.b,
		state:          state,
		base:           base.Header(),
		chainConfig:    chainConfig,
		gp:             new(core.GasPool).AddGas(gasCap),
		traceTransfers: opts.TraceTransfers,
		validate:       opts.Validation,
		fullTx:         opts.ReturnFullTransactions,
	}
	return sim.execute(ctx, opts.BlockStateCalls)
}
