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

package eth

// eth_simulateV1, ported from the reference's api/ethapi/simulate.go, itself
// adapted from go-ethereum's internal/ethapi/simulate.go. It does not run on
// the calls of rpc/exec: its blocks are above the head, so their rules are
// not in Core, BLOCKHASH must see the simulated blocks before them, and the
// simulator brings its own precompiles and EVM configuration.

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
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

const (
	// maxSimulateBlocks bounds the blocks of one request, counted from the
	// base block to the last simulated one (ADR6).
	maxSimulateBlocks = 256
	// maxSimulateCallsPerBlock bounds the calls of one simulated block.
	maxSimulateCallsPerBlock = 5000
	// maxSimulateTotalCalls bounds the calls of one request.
	maxSimulateTotalCalls = 10000
	// timestampIncrement is the time between simulated blocks without one.
	timestampIncrement = 12
)

// simulateOptions are the argument of eth_simulateV1.
type simulateOptions struct {
	BlockStateCalls        []simulatedBlock `json:"blockStateCalls"`
	TraceTransfers         bool             `json:"traceTransfers"`
	Validation             bool             `json:"validation"`
	ReturnFullTransactions bool             `json:"returnFullTransactions"`
}

// simulatedBlock is a batch of calls run in one simulated block.
type simulatedBlock struct {
	BlockOverrides *simulatedHeaderOverrides `json:"blockOverrides"`
	StateOverrides *exec.StateOverride       `json:"stateOverrides"`
	Calls          []exec.CallArguments      `json:"calls"`
}

// simulatedHeaderOverrides are the header fields a simulated block sets.
type simulatedHeaderOverrides struct {
	Number        *hexutil.Big       `json:"blockNumber"`
	Time          *hexutil.Uint64    `json:"timestamp"`
	GasLimit      *hexutil.Uint64    `json:"gasLimit"`
	FeeRecipient  *common.Address    `json:"feeRecipient"`
	PrevRandao    *common.Hash       `json:"prevRandao"`
	BaseFeePerGas *hexutil.Big       `json:"baseFeePerGas"`
	Withdrawals   *types.Withdrawals `json:"withdrawals"`
}

// applyTo returns a copy of the header with the overrides.
func (overrides *simulatedHeaderOverrides) applyTo(header *evmcore.EvmHeader) *evmcore.EvmHeader {
	changed := *header
	if overrides == nil {
		return &changed
	}
	if overrides.Number != nil {
		changed.Number = new(big.Int).Set(overrides.Number.ToInt())
	}
	if overrides.Time != nil {
		changed.Time = inter.FromUnix(int64(*overrides.Time))
	}
	if overrides.GasLimit != nil {
		changed.GasLimit = uint64(*overrides.GasLimit)
	}
	if overrides.FeeRecipient != nil {
		changed.Coinbase = *overrides.FeeRecipient
	}
	if overrides.PrevRandao != nil {
		changed.PrevRandao = *overrides.PrevRandao
	}
	if overrides.BaseFeePerGas != nil {
		changed.BaseFee = overrides.BaseFeePerGas.ToInt()
	}
	return &changed
}

// callError is the error of one simulated call.
type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
}

// callResult is the result of one simulated call.
type callResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

// MarshalJSON writes no logs as [], not null.
func (result *callResult) MarshalJSON() ([]byte, error) {
	type plain callResult
	if result.Logs == nil {
		result.Logs = []*types.Log{}
	}
	return json.Marshal((*plain)(result))
}

// blockResult is a simulated block with the results of its calls.
type blockResult struct {
	block            *evmcore.EvmBlock
	calls            []callResult
	chainID          *big.Int
	fullTransactions bool
	senders          map[common.Hash]common.Address
	receipts         types.Receipts
}

// MarshalJSON writes the block as a block of the chain with its calls, the
// fields in the order of a map's keys, as the reference wrote them.
func (result *blockResult) MarshalJSON() ([]byte, error) {
	block := format.Block(result.block, result.receipts, result.fullTransactions, result.chainID)
	if result.fullTransactions {
		result.formatTransactions(block)
	}
	encoded, err := json.Marshal(block)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	calls := result.calls
	if calls == nil {
		calls = []callResult{}
	}
	if fields["calls"], err = json.Marshal(calls); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

// formatTransactions writes each full transaction as the reference did: at
// the first position of its hash in the block, which it looked the
// transaction up by, and with its sender, as it is not signed.
func (result *blockResult) formatTransactions(block *evmcore.EvmBlockJson) {
	first := map[common.Hash]uint64{}
	for index, transaction := range result.block.Transactions {
		if _, seen := first[transaction.Hash()]; !seen {
			first[transaction.Hash()] = uint64(index)
		}
	}
	for index, transaction := range result.block.Transactions {
		formatted := format.TransactionAt(result.block, first[transaction.Hash()], result.chainID)
		if sender, found := result.senders[transaction.Hash()]; found {
			formatted.From = sender
		}
		block.Txs[index] = formatted
	}
}

// simulateError is a failure of a simulation with the code the
// specification of eth_simulateV1 gives it.
type simulateError struct {
	message string
	code    int
}

func (err *simulateError) Error() string {
	return err.message
}

func (err *simulateError) ErrorCode() int {
	return err.code
}

const (
	codeReverted              = -32000
	codeVMError               = -32015
	codeNonceTooHigh          = -38011
	codeNonceTooLow           = -38010
	codeIntrinsicGas          = -38013
	codeInsufficientFunds     = -38014
	codeBlockGasLimitReached  = -38015
	codeBlockNumberInvalid    = -38020
	codeBlockTimestampInvalid = -38021
	codeSenderIsNotEOA        = -38024
	codeMaxInitCodeExceeded   = -38025
	codeClientLimitExceeded   = -38026
)

// validationError gives an error of a message the code the specification
// gives its kind.
func validationError(err error) *simulateError {
	codes := []struct {
		kind error
		code int
	}{
		{core.ErrNonceTooHigh, codeNonceTooHigh},
		{core.ErrNonceTooLow, codeNonceTooLow},
		{core.ErrSenderNoEOA, codeSenderIsNotEOA},
		{core.ErrFeeCapTooLow, dispatch.CodeInvalidParameters},
		{core.ErrTipAboveFeeCap, dispatch.CodeInvalidParameters},
		{core.ErrTipVeryHigh, dispatch.CodeInvalidParameters},
		{core.ErrFeeCapVeryHigh, dispatch.CodeInvalidParameters},
		{core.ErrInsufficientFunds, codeInsufficientFunds},
		{core.ErrIntrinsicGas, codeIntrinsicGas},
		{vm.ErrMaxInitCodeSizeExceeded, codeMaxInitCodeExceeded},
	}
	for _, entry := range codes {
		if errors.Is(err, entry.kind) {
			return &simulateError{message: err.Error(), code: entry.code}
		}
	}
	return &simulateError{message: err.Error(), code: dispatch.CodeInternalError}
}

// simulateV1 checks the counts of blocks and calls, then simulates on the
// state of the base block.
func (handler *handlers) simulateV1(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	options := dispatch.Argument[simulateOptions](arguments, 0)
	if err := checkSimulationSize(options.BlockStateCalls); err != nil {
		return nil, err
	}
	head, statedb, base, err := handler.stateAt(ctx, dispatch.ArgumentOr(arguments, 1, chain.Latest()))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	limits := handler.environment.Config()
	gasCap := limits.RPCGasCap
	if gasCap == 0 {
		gasCap = ^uint64(0)
	}
	simulation := &simulator{
		environment:      handler.environment,
		head:             head,
		state:            statedb,
		base:             base,
		chainConfig:      handler.environment.ChainConfig(base.Number.Uint64()),
		gasPool:          core.NewGasPool(gasCap),
		traceTransfers:   options.TraceTransfers,
		validate:         options.Validation,
		fullTransactions: options.ReturnFullTransactions,
	}
	return simulation.execute(ctx, options.BlockStateCalls, limits.RPCEVMTimeout)
}

func checkSimulationSize(blocks []simulatedBlock) error {
	if len(blocks) == 0 {
		return &simulateError{message: "empty input", code: dispatch.CodeInvalidParameters}
	}
	if len(blocks) > maxSimulateBlocks {
		return &simulateError{message: "too many blocks", code: codeClientLimitExceeded}
	}
	total := 0
	for _, block := range blocks {
		if len(block.Calls) > maxSimulateCallsPerBlock {
			return &simulateError{message: fmt.Sprintf("too many calls in block: %d > %d", len(block.Calls), maxSimulateCallsPerBlock), code: codeClientLimitExceeded}
		}
		total += len(block.Calls)
		if total > maxSimulateTotalCalls {
			return &simulateError{message: fmt.Sprintf("too many calls: %d > %d", total, maxSimulateTotalCalls), code: codeClientLimitExceeded}
		}
	}
	return nil
}

// simulatedChain answers the headers BLOCKHASH walks: the base block, the
// blocks simulated before, and the chain below the base. Its walk is the one
// left in the module (V11): a simulated block is at most 256 above the base.
type simulatedChain struct {
	environment *exec.Environment
	head        exec.Head
	base        *evmcore.EvmHeader
	simulated   []*evmcore.EvmHeader
}

func (simulated *simulatedChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	if simulated.base.Number.Uint64() == number && simulated.base.Hash == hash {
		return simulated.base
	}
	for _, header := range simulated.simulated {
		if header != nil && header.Number.Uint64() == number && header.Hash == hash {
			return header
		}
	}
	header, err := simulated.environment.HeaderAt(simulated.head, number)
	if err != nil || header.Hash != hash {
		return nil
	}
	return header
}

// simulator simulates a series of blocks on one state; it is not safe for
// concurrent use.
type simulator struct {
	environment      *exec.Environment
	head             exec.Head
	state            state.StateDB
	base             *evmcore.EvmHeader
	chainConfig      *params.ChainConfig
	gasPool          *core.GasPool
	traceTransfers   bool
	validate         bool
	fullTransactions bool
}

// execute simulates the blocks in order, each on top of the one before,
// within the EVM timeout.
func (simulation *simulator) execute(ctx context.Context, blocks []simulatedBlock, timeout time.Duration) ([]*blockResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := exec.WithOptionalTimeout(ctx, timeout)
	defer cancel()
	blocks, err := simulation.sanitizeChain(blocks)
	if err != nil {
		return nil, err
	}
	headers, err := simulation.makeHeaders(blocks)
	if err != nil {
		return nil, err
	}
	results := make([]*blockResult, len(blocks))
	parent := simulation.base
	for index := range blocks {
		result, err := simulation.processBlock(ctx, &blocks[index], headers[index], parent, headers[:index], timeout)
		if err != nil {
			return nil, err
		}
		headers[index] = result.block.Header()
		results[index] = result
		parent = result.block.Header()
	}
	return results, nil
}

// blockRun is the EVM of one simulated block with what its calls collect.
type blockRun struct {
	header   *evmcore.EvmHeader
	evm      *vm.EVM
	state    state.StateDB
	tracer   *simulationTracer
	gasUsed  uint64
	results  []callResult
	receipts types.Receipts
	senders  map[common.Hash]common.Address
	timeout  time.Duration
}

// processBlock simulates one block: its base fee, its state overrides, the
// history contract of EIP-2935, its calls, and the block they make.
func (simulation *simulator) processBlock(ctx context.Context, block *simulatedBlock, header, parent *evmcore.EvmHeader, previous []*evmcore.EvmHeader, timeout time.Duration) (*blockResult, error) {
	if parent == nil {
		return nil, errors.New("parent header is nil")
	}
	header.ParentHash = parent.Hash
	if header.BaseFee == nil {
		if simulation.validate && parent.BaseFee != nil {
			header.BaseFee = new(big.Int).Set(parent.BaseFee)
		} else {
			header.BaseFee = big.NewInt(0)
		}
	}
	run, err := simulation.startBlock(block, header, previous)
	if err != nil {
		return nil, err
	}
	run.timeout = timeout
	transactions := make([]*types.Transaction, len(block.Calls))
	for index := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if transactions[index], err = simulation.runCall(ctx, run, &block.Calls[index], index); err != nil {
			return nil, err
		}
	}
	return simulation.finishBlock(run, transactions), nil
}

// startBlock builds the EVM of a block on the state, with its overrides
// applied.
func (simulation *simulator) startBlock(block *simulatedBlock, header *evmcore.EvmHeader, previous []*evmcore.EvmHeader) (*blockRun, error) {
	headers := &simulatedChain{environment: simulation.environment, head: simulation.head, base: simulation.base, simulated: previous}
	blockContext := evmcore.NewEVMBlockContext(header, headers, nil)
	precompiles := simulation.activePrecompiles()
	if err := applySimulatedOverrides(block.StateOverrides, precompiles, simulation.state); err != nil {
		return nil, err
	}
	tracer := newSimulationTracer(simulation.traceTransfers, blockContext.BlockNumber.Uint64(), blockContext.Time)
	config := vm.Config{NoBaseFee: !simulation.validate, Tracer: tracer.Hooks()}
	traced := state.WrapWithLogger(simulation.state, tracer.Hooks())
	evm := vm.NewEVM(blockContext, traced, simulation.chainConfig, config)
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	if simulation.chainConfig.IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, traced)
	}
	return &blockRun{
		header:   header,
		evm:      evm,
		state:    traced,
		tracer:   tracer,
		results:  make([]callResult, len(block.Calls)),
		receipts: make(types.Receipts, len(block.Calls)),
		senders:  map[common.Hash]common.Address{},
	}, nil
}

// runCall runs one call of a block and records its result and receipt.
func (simulation *simulator) runCall(ctx context.Context, run *blockRun, call *exec.CallArguments, index int) (*types.Transaction, error) {
	if err := simulation.sanitizeCall(call, run.state, run.header, run.gasUsed); err != nil {
		return nil, err
	}
	transaction, err := call.ToTransaction()
	if err != nil {
		return nil, err
	}
	hash := transaction.Hash()
	run.senders[hash] = call.Sender()
	run.tracer.reset(hash, uint(index))
	run.state.SetTxContext(hash, index)
	message, err := call.ToMessage(simulation.gasPool.Gas(), run.header.BaseFee)
	if err != nil {
		return nil, validationError(err)
	}
	message.SkipNonceChecks = !simulation.validate
	message.SkipTransactionChecks = !simulation.validate
	result, err := applySimulatedMessage(ctx, run.evm, message, run.timeout, simulation.gasPool)
	if err != nil {
		return nil, validationError(err)
	}
	run.gasUsed += result.UsedGas
	run.results[index] = newCallResult(result, run.tracer.Logs())
	receipt := &types.Receipt{Status: uint64(run.results[index].Status), Logs: run.results[index].Logs}
	receipt.Bloom = types.CreateBloom(receipt)
	run.receipts[index] = receipt
	return transaction, nil
}

func newCallResult(result *core.ExecutionResult, logs []*types.Log) callResult {
	answer := callResult{ReturnValue: result.ReturnData, GasUsed: hexutil.Uint64(result.UsedGas), Logs: logs}
	if !result.Failed() {
		answer.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
		return answer
	}
	answer.Status = hexutil.Uint64(types.ReceiptStatusFailed)
	if errors.Is(result.Err, vm.ErrExecutionReverted) {
		reverted := dispatch.ToError(exec.RevertError(result))
		answer.Error = &callError{Message: reverted.Message, Code: codeReverted, Data: reverted.Data.(string)}
	} else {
		answer.Error = &callError{Message: result.Err.Error(), Code: codeVMError}
	}
	return answer
}

// finishBlock completes the header with what the calls did and makes the
// block, whose hash the logs then take.
func (simulation *simulator) finishBlock(run *blockRun, transactions []*types.Transaction) *blockResult {
	header := run.header
	header.GasUsed = run.gasUsed
	header.Root = run.state.GetStateHash()
	// NewEvmBlock derives the transactions root.
	block := evmcore.NewEvmBlock(header, transactions)
	block.Hash = block.EthHeader().Hash()
	repairLogs(run.results, block)
	return &blockResult{
		block:            block,
		calls:            run.results,
		chainID:          simulation.chainConfig.ChainID,
		fullTransactions: simulation.fullTransactions,
		senders:          run.senders,
		receipts:         run.receipts,
	}
}

// repairLogs gives the logs the hash, number and time of their block, which
// are known once the block is made.
func repairLogs(calls []callResult, block *evmcore.EvmBlock) {
	for index := range calls {
		for _, log := range calls[index].Logs {
			log.BlockHash = block.Hash
			log.BlockNumber = block.NumberU64()
			log.BlockTimestamp = uint64(block.Time.Unix())
		}
	}
}

// applySimulatedMessage applies a message that ctx cancels, as the
// reference's DoCall did.
func applySimulatedMessage(ctx context.Context, evm *vm.EVM, message *core.Message, timeout time.Duration, gasPool *core.GasPool) (*core.ExecutionResult, error) {
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()
	result, err := core.ApplyMessage(evm, message, gasPool)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, message.GasLimit)
	}
	return result, nil
}

// applySimulatedOverrides applies the state overrides of a block. An
// overridden precompile stops being one. Unlike exec.StateOverride.Apply,
// the overrides are finalised, not ended as a transaction, as in the
// reference.
func applySimulatedOverrides(overrides *exec.StateOverride, precompiles vm.PrecompiledContracts, statedb state.StateDB) error {
	if overrides == nil {
		return nil
	}
	for address, account := range *overrides {
		delete(precompiles, address)
		if account.Nonce != nil {
			statedb.SetNonce(address, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		if account.Code != nil {
			statedb.SetCode(address, *account.Code, tracing.CodeChangeUnspecified)
		}
		if account.Balance != nil && *account.Balance != nil {
			statedb.SetBalance(address, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", address.Hex())
		}
		if account.State != nil {
			statedb.SetStorage(address, *account.State)
		}
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				statedb.SetState(address, key, value)
			}
		}
	}
	statedb.Finalise(false)
	return nil
}

// sanitizeCall fills in the nonce from the state, the gas left in the block
// and the fees, and refuses what cannot be a transaction.
func (simulation *simulator) sanitizeCall(call *exec.CallArguments, statedb state.StateDB, header *evmcore.EvmHeader, gasUsed uint64) error {
	if call.Nonce == nil {
		nonce := statedb.GetNonce(call.Sender())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if call.Gas == nil {
		remaining := header.GasLimit - gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	if gasUsed+uint64(*call.Gas) > header.GasLimit {
		return &simulateError{message: fmt.Sprintf("block gas limit reached: %d >= %d", gasUsed+uint64(*call.Gas), header.GasLimit), code: codeBlockGasLimitReached}
	}
	if call.Recipient == nil {
		if call.BlobHashes != nil || call.BlobFeeCap != nil {
			return core.ErrBlobTxCreate
		}
		if call.AuthorizationList != nil {
			return core.ErrSetCodeTxCreate
		}
	}
	return simulation.setCallPrices(call, header.BaseFee)
}

// setCallPrices defaults the fees to zero, so that calls succeed without
// funds, in the style the base fee calls for.
func (simulation *simulator) setCallPrices(call *exec.CallArguments, baseFee *big.Int) error {
	if call.GasPrice != nil && (call.MaxFeePerGas != nil || call.MaxPriorityFeePerGas != nil) {
		return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	if call.ChainID == nil {
		call.ChainID = (*hexutil.Big)(simulation.chainConfig.ChainID)
	}
	if call.Value == nil {
		call.Value = new(hexutil.Big)
	}
	switch {
	case baseFee != nil && call.GasPrice == nil:
		if call.MaxFeePerGas == nil {
			call.MaxFeePerGas = (*hexutil.Big)(new(big.Int))
		}
		if call.MaxPriorityFeePerGas == nil {
			call.MaxPriorityFeePerGas = (*hexutil.Big)(new(big.Int))
		}
	case call.GasPrice == nil:
		call.GasPrice = (*hexutil.Big)(new(big.Int))
	}
	return nil
}

// activePrecompiles returns the precompiles of the base block.
func (simulation *simulator) activePrecompiles() vm.PrecompiledContracts {
	base := simulation.base
	rules := simulation.chainConfig.Rules(base.Number, base.PrevRandao != (common.Hash{}), uint64(base.Time.Unix()))
	return vm.ActivePrecompiledContracts(rules)
}

// sanitizeChain numbers and times the blocks, refuses numbers or times out
// of order and a last block more than 256 above the base, and fills gaps
// with empty blocks.
func (simulation *simulator) sanitizeChain(blocks []simulatedBlock) ([]simulatedBlock, error) {
	sanitized := make([]simulatedBlock, 0, len(blocks))
	base := simulation.base
	previousNumber := new(big.Int).Set(base.Number)
	previousTime := uint64(base.Time.Unix())
	for _, block := range blocks {
		if block.BlockOverrides == nil {
			block.BlockOverrides = new(simulatedHeaderOverrides)
		}
		overrides := block.BlockOverrides
		if overrides.Number == nil {
			overrides.Number = (*hexutil.Big)(new(big.Int).Add(previousNumber, big.NewInt(1)))
		}
		if overrides.Withdrawals == nil {
			overrides.Withdrawals = &types.Withdrawals{}
		}
		difference := new(big.Int).Sub(overrides.Number.ToInt(), previousNumber)
		if difference.Cmp(common.Big0) <= 0 {
			return nil, &simulateError{message: fmt.Sprintf("block numbers must be in order: %d <= %d", overrides.Number.ToInt().Uint64(), previousNumber.Uint64()), code: codeBlockNumberInvalid}
		}
		if total := new(big.Int).Sub(overrides.Number.ToInt(), base.Number); total.Cmp(big.NewInt(maxSimulateBlocks)) > 0 {
			return nil, &simulateError{message: "too many blocks", code: codeClientLimitExceeded}
		}
		sanitized, previousTime = fillGap(sanitized, previousNumber, previousTime, difference)
		previousNumber = new(big.Int).Set(overrides.Number.ToInt())
		if overrides.Time == nil {
			next := previousTime + timestampIncrement
			overrides.Time = (*hexutil.Uint64)(&next)
		} else if uint64(*overrides.Time) <= previousTime {
			return nil, &simulateError{message: fmt.Sprintf("block timestamps must be in order: %d <= %d", uint64(*overrides.Time), previousTime), code: codeBlockTimestampInvalid}
		}
		previousTime = uint64(*overrides.Time)
		sanitized = append(sanitized, block)
	}
	return sanitized, nil
}

// fillGap appends the empty blocks between the previous block and one the
// given difference above it, and returns the time of the last.
func fillGap(blocks []simulatedBlock, previousNumber *big.Int, previousTime uint64, difference *big.Int) ([]simulatedBlock, uint64) {
	if difference.Cmp(big.NewInt(1)) <= 0 {
		return blocks, previousTime
	}
	gap := new(big.Int).Sub(difference, big.NewInt(1)).Uint64()
	for offset := range gap {
		number := new(big.Int).Add(previousNumber, new(big.Int).SetUint64(offset+1))
		next := previousTime + timestampIncrement
		blocks = append(blocks, simulatedBlock{BlockOverrides: &simulatedHeaderOverrides{
			Number:      (*hexutil.Big)(number),
			Time:        (*hexutil.Uint64)(&next),
			Withdrawals: &types.Withdrawals{},
		}})
		previousTime = next
	}
	return blocks, previousTime
}

// makeHeaders makes the first headers of the blocks, each from the one
// before with the block's overrides; what the calls determine comes later.
func (simulation *simulator) makeHeaders(blocks []simulatedBlock) ([]*evmcore.EvmHeader, error) {
	headers := make([]*evmcore.EvmHeader, len(blocks))
	previous := simulation.base
	for index, block := range blocks {
		if block.BlockOverrides == nil || block.BlockOverrides.Number == nil {
			return nil, errors.New("empty block number")
		}
		number := block.BlockOverrides.Number.ToInt()
		timestamp := uint64(*block.BlockOverrides.Time)
		var withdrawalsHash *common.Hash
		if simulation.chainConfig.IsShanghai(number, timestamp) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		template := &evmcore.EvmHeader{
			Number:          number,
			Time:            inter.FromUnix(int64(timestamp)),
			Coinbase:        previous.Coinbase,
			GasLimit:        previous.GasLimit,
			BaseFee:         previous.BaseFee,
			PrevRandao:      previous.PrevRandao,
			WithdrawalsHash: withdrawalsHash,
		}
		headers[index] = block.BlockOverrides.applyTo(template)
		previous = headers[index]
	}
	return headers, nil
}
