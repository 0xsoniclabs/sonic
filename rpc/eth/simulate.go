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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// maxSimulateBlocks bounds the blocks of one simulation, gaps included.
	maxSimulateBlocks = 256
	// maxSimulateCallsPerBlock bounds the calls of one simulated block.
	maxSimulateCallsPerBlock = 5000
	// maxSimulateTotalCalls bounds the calls of one simulation.
	maxSimulateTotalCalls = 10000
	// timestampIncrement is the time between simulated blocks that name none.
	timestampIncrement = 12
)

// The error codes of eth_simulateV1.
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
	codeMaxInitCodeSize       = -38025
	codeClientLimitExceeded   = -38026
)

type simulateParams struct {
	Options simulationOptions
	Block   *chain.BlockReference
}

type simulationOptions struct {
	BlockStateCalls        []simulatedBlock `json:"blockStateCalls"`
	TraceTransfers         bool             `json:"traceTransfers"`
	Validation             bool             `json:"validation"`
	ReturnFullTransactions bool             `json:"returnFullTransactions"`
}

// simulatedBlock is a block of calls run in order.
type simulatedBlock struct {
	BlockOverrides *headerOverrides     `json:"blockOverrides"`
	StateOverrides *exec.StateOverride  `json:"stateOverrides"`
	Calls          []exec.CallArguments `json:"calls"`
}

// headerOverrides are the fields of a simulated block's header a client sets.
type headerOverrides struct {
	Number        *hexutil.Big       `json:"blockNumber"`
	Time          *hexutil.Uint64    `json:"timestamp"`
	GasLimit      *hexutil.Uint64    `json:"gasLimit"`
	FeeRecipient  *common.Address    `json:"feeRecipient"`
	PrevRandao    *common.Hash       `json:"prevRandao"`
	BaseFeePerGas *hexutil.Big       `json:"baseFeePerGas"`
	Withdrawals   *types.Withdrawals `json:"withdrawals"`
}

// applyTo is a copy of the header with the overrides set.
func (o *headerOverrides) applyTo(header *evmcore.EvmHeader) *evmcore.EvmHeader {
	copied := *header
	if o.Number != nil {
		copied.Number = new(big.Int).Set(o.Number.ToInt())
	}
	if o.Time != nil {
		copied.Time = inter.FromUnix(int64(*o.Time))
	}
	if o.GasLimit != nil {
		copied.GasLimit = uint64(*o.GasLimit)
	}
	if o.FeeRecipient != nil {
		copied.Coinbase = *o.FeeRecipient
	}
	if o.PrevRandao != nil {
		copied.PrevRandao = *o.PrevRandao
	}
	if o.BaseFeePerGas != nil {
		copied.BaseFee = o.BaseFeePerGas.ToInt()
	}
	return &copied
}

// callError is the failure of a simulated call.
type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
}

// callResult is the outcome of a simulated call.
type callResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

// MarshalJSON answers no logs as an empty list.
func (r *callResult) MarshalJSON() ([]byte, error) {
	type plain callResult
	if r.Logs == nil {
		r.Logs = []*types.Log{}
	}
	return json.Marshal((*plain)(r))
}

// blockResult is a simulated block with the outcomes of its calls.
type blockResult struct {
	block    *evmcore.EvmBlock
	calls    []callResult
	chainID  *big.Int
	full     bool
	senders  map[common.Hash]common.Address
	receipts []*types.Receipt
}

// MarshalJSON forms the block as a block of the chain, with the senders of
// its unsigned transactions and the outcomes of its calls.
func (r *blockResult) MarshalJSON() ([]byte, error) {
	block := format.Block(r.block, r.receipts, r.full, r.chainID)
	if r.full {
		for _, formed := range block.Txs {
			transaction := formed.(*format.Transaction)
			if sender, found := r.senders[transaction.Hash]; found {
				transaction.From = sender
			}
		}
	}
	encoded, err := json.Marshal(block)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	calls := r.calls
	if calls == nil {
		calls = []callResult{}
	}
	if fields["calls"], err = json.Marshal(calls); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

// simulateV1 runs blocks of calls on top of a block, the head if it names
// none, as go-ethereum's eth_simulateV1 does on the network's rules.
func (h *handlers) simulateV1(ctx context.Context, params simulateParams) ([]*blockResult, error) {
	if err := checkSimulationSize(params.Options.BlockStateCalls); err != nil {
		return nil, err
	}
	head := h.environment.Head()
	stateDB, base, err := h.environment.StateAt(head, orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	gasCap := h.environment.Config().RPCGasCap
	if gasCap == 0 {
		gasCap = math.MaxUint64
	}
	vmConfig, err := h.environment.VMConfig(base.Number.Uint64())
	if err != nil {
		return nil, err
	}
	simulation := &simulator{
		environment: h.environment,
		head:        head,
		state:       stateDB,
		base:        base,
		chainConfig: h.environment.ChainConfig(base.Number.Uint64()),
		vmConfig:    vmConfig,
		gasPool:     core.NewGasPool(gasCap),
		options:     params.Options,
	}
	return simulation.run(ctx)
}

func checkSimulationSize(blocks []simulatedBlock) error {
	if len(blocks) == 0 {
		return dispatch.InvalidParams("empty input")
	}
	if len(blocks) > maxSimulateBlocks {
		return limitExceeded("too many blocks")
	}
	total := 0
	for _, block := range blocks {
		if len(block.Calls) > maxSimulateCallsPerBlock {
			return limitExceeded(fmt.Sprintf("too many calls in block: %d > %d", len(block.Calls), maxSimulateCallsPerBlock))
		}
		if total += len(block.Calls); total > maxSimulateTotalCalls {
			return limitExceeded(fmt.Sprintf("too many calls: %d > %d", total, maxSimulateTotalCalls))
		}
	}
	return nil
}

func limitExceeded(message string) error {
	return &dispatch.Error{Code: codeClientLimitExceeded, Message: message}
}

// simulator runs the blocks of a simulation on one state.
type simulator struct {
	environment *exec.Environment
	head        *evmcore.EvmBlock
	state       state.StateDB
	base        *evmcore.EvmHeader
	chainConfig *params.ChainConfig
	vmConfig    vm.Config
	gasPool     *core.GasPool
	options     simulationOptions
}

func (s *simulator) run(ctx context.Context) ([]*blockResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	timeout := s.environment.Config().RPCEVMTimeout
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	blocks, err := s.sanitizeChain(s.options.BlockStateCalls)
	if err != nil {
		return nil, err
	}
	headers := s.makeHeaders(blocks)
	results := make([]*blockResult, len(blocks))
	parent := s.base
	for i := range blocks {
		result, err := s.runBlock(ctx, &blocks[i], headers[i], parent, headers[:i], timeout)
		if err != nil {
			return nil, err
		}
		headers[i] = result.block.Header()
		results[i] = result
		parent = result.block.Header()
	}
	return results, nil
}

// sanitizeChain numbers and times the blocks, each after the one before, and
// fills a gap between two numbers with empty blocks.
func (s *simulator) sanitizeChain(blocks []simulatedBlock) ([]simulatedBlock, error) {
	sanitized := make([]simulatedBlock, 0, len(blocks))
	previousNumber := new(big.Int).Set(s.base.Number)
	previousTime := uint64(s.base.Time.Unix())
	for _, block := range blocks {
		if block.BlockOverrides == nil {
			block.BlockOverrides = new(headerOverrides)
		}
		if block.BlockOverrides.Number == nil {
			block.BlockOverrides.Number = (*hexutil.Big)(new(big.Int).Add(previousNumber, big.NewInt(1)))
		}
		if block.BlockOverrides.Withdrawals == nil {
			block.BlockOverrides.Withdrawals = &types.Withdrawals{}
		}
		number := block.BlockOverrides.Number.ToInt()
		difference := new(big.Int).Sub(number, previousNumber)
		if difference.Sign() <= 0 {
			return nil, &dispatch.Error{Code: codeBlockNumberInvalid,
				Message: fmt.Sprintf("block numbers must be in order: %d <= %d", number.Uint64(), previousNumber.Uint64())}
		}
		if total := new(big.Int).Sub(number, s.base.Number); total.Cmp(big.NewInt(maxSimulateBlocks)) > 0 {
			return nil, limitExceeded("too many blocks")
		}
		for i := uint64(1); i < difference.Uint64(); i++ {
			previousTime += timestampIncrement
			gapTime := previousTime
			sanitized = append(sanitized, simulatedBlock{BlockOverrides: &headerOverrides{
				Number:      (*hexutil.Big)(new(big.Int).Add(previousNumber, new(big.Int).SetUint64(i))),
				Time:        (*hexutil.Uint64)(&gapTime),
				Withdrawals: &types.Withdrawals{},
			}})
		}
		previousNumber = new(big.Int).Set(number)
		if block.BlockOverrides.Time == nil {
			previousTime += timestampIncrement
			blockTime := previousTime
			block.BlockOverrides.Time = (*hexutil.Uint64)(&blockTime)
		} else {
			blockTime := uint64(*block.BlockOverrides.Time)
			if blockTime <= previousTime {
				return nil, &dispatch.Error{Code: codeBlockTimestampInvalid,
					Message: fmt.Sprintf("block timestamps must be in order: %d <= %d", blockTime, previousTime)}
			}
			previousTime = blockTime
		}
		sanitized = append(sanitized, block)
	}
	return sanitized, nil
}

// makeHeaders forms the headers of the blocks, each taking what a block
// does not override from the one before. Gas, roots and hashes follow from
// the runs.
func (s *simulator) makeHeaders(blocks []simulatedBlock) []*evmcore.EvmHeader {
	headers := make([]*evmcore.EvmHeader, len(blocks))
	previous := s.base
	for i, block := range blocks {
		overrides := block.BlockOverrides
		number, timestamp := overrides.Number.ToInt(), uint64(*overrides.Time)
		var withdrawalsHash *common.Hash
		if s.chainConfig.IsShanghai(number, timestamp) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		headers[i] = overrides.applyTo(&evmcore.EvmHeader{
			Number:          number,
			Time:            inter.FromUnix(int64(timestamp)),
			Coinbase:        previous.Coinbase,
			GasLimit:        previous.GasLimit,
			BaseFee:         previous.BaseFee,
			PrevRandao:      previous.PrevRandao,
			WithdrawalsHash: withdrawalsHash,
		})
		previous = headers[i]
	}
	return headers
}

// runBlock runs the calls of a block on the state and assembles the block.
func (s *simulator) runBlock(ctx context.Context, block *simulatedBlock, header, parent *evmcore.EvmHeader, previous []*evmcore.EvmHeader, timeout time.Duration) (*blockResult, error) {
	header.ParentHash = parent.Hash
	if header.BaseFee == nil {
		header.BaseFee = new(big.Int)
		if s.options.Validation && parent.BaseFee != nil {
			header.BaseFee.Set(parent.BaseFee)
		}
	}
	blockContext := evmcore.NewEVMBlockContext(header, &simulatedChain{simulator: s, previous: previous}, nil)
	rules := s.chainConfig.Rules(s.base.Number, s.base.PrevRandao != (common.Hash{}), uint64(s.base.Time.Unix()))
	precompiles := vm.ActivePrecompiledContracts(rules)
	statePrecompiles := maps.Clone(s.vmConfig.StatePrecompiles)
	if err := applyOverrides(block.StateOverrides, precompiles, statePrecompiles, s.state); err != nil {
		return nil, err
	}

	collector := newLogCollector(s.options.TraceTransfers, blockContext.BlockNumber.Uint64(), blockContext.Time)
	vmConfig := s.vmConfig
	vmConfig.NoBaseFee = !s.options.Validation
	vmConfig.StatePrecompiles = statePrecompiles
	vmConfig.Tracer = collector.hooks()
	logging := exec.WithHooks(s.state, vmConfig.Tracer)
	evm := vm.NewEVM(blockContext, logging, s.chainConfig, vmConfig)
	evm.SetPrecompiles(precompiles)
	if s.chainConfig.IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, logging)
	}

	result := &blockResult{
		chainID:  s.chainConfig.ChainID,
		full:     s.options.ReturnFullTransactions,
		senders:  map[common.Hash]common.Address{},
		calls:    make([]callResult, len(block.Calls)),
		receipts: make([]*types.Receipt, len(block.Calls)),
	}
	transactions := make([]*types.Transaction, len(block.Calls))
	gasUsed := uint64(0)
	for i := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		call := &block.Calls[i]
		if err := s.sanitizeCall(call, logging, header, gasUsed); err != nil {
			return nil, err
		}
		transaction, err := call.ToTransaction()
		if err != nil {
			return nil, err
		}
		transactions[i] = transaction
		result.senders[transaction.Hash()] = call.Sender()
		collector.startTransaction(transaction.Hash(), uint(i))
		logging.SetTxContext(transaction.Hash(), i)

		message, err := call.ToMessage(s.gasPool.Gas(), header.BaseFee)
		if err != nil {
			return nil, validationError(err)
		}
		message.SkipNonceChecks = !s.options.Validation
		outcome, err := applyMessage(ctx, evm, message, timeout, s.gasPool)
		if err != nil {
			return nil, validationError(err)
		}
		logging.EndTransaction()
		gasUsed += outcome.UsedGas
		result.calls[i] = newCallResult(outcome, collector.logs())
		receipt := &types.Receipt{Status: uint64(result.calls[i].Status), Logs: result.calls[i].Logs}
		receipt.Bloom = types.CreateBloom(receipt)
		result.receipts[i] = receipt
	}

	header.GasUsed = gasUsed
	header.Root = logging.GetStateHash()
	header.TxHash = types.EmptyRootHash
	if len(transactions) > 0 {
		header.TxHash = types.DeriveSha(types.Transactions(transactions), trie.NewStackTrie(nil))
	}
	result.block = evmcore.NewEvmBlock(header, transactions)
	result.block.Hash = result.block.EthHeader().Hash()
	for _, call := range result.calls {
		for _, log := range call.Logs {
			log.BlockHash = result.block.Hash
			log.BlockNumber = result.block.NumberU64()
			log.BlockTimestamp = uint64(result.block.Time.Unix())
		}
	}
	return result, nil
}

// newCallResult is the outcome of a call: its return data, also of a revert,
// its logs and gas, and why it failed.
func newCallResult(outcome *core.ExecutionResult, logs []*types.Log) callResult {
	result := callResult{
		ReturnValue: outcome.ReturnData,
		GasUsed:     hexutil.Uint64(outcome.UsedGas),
		Logs:        logs,
		Status:      hexutil.Uint64(types.ReceiptStatusSuccessful),
	}
	if !outcome.Failed() {
		return result
	}
	result.Status = hexutil.Uint64(types.ReceiptStatusFailed)
	if errors.Is(outcome.Err, vm.ErrExecutionReverted) {
		reverted := exec.NewRevertError(outcome)
		result.Error = &callError{Message: reverted.Error(), Code: codeReverted, Data: reverted.ErrorData().(string)}
	} else {
		result.Error = &callError{Message: outcome.Err.Error(), Code: codeVMError}
	}
	return result
}

// applyMessage runs a message, stopped when the context ends.
func applyMessage(ctx context.Context, evm *vm.EVM, message *core.Message, timeout time.Duration, gasPool *core.GasPool) (*core.ExecutionResult, error) {
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

// applyOverrides applies the state overrides of a block. An overridden
// precompile is a precompile no more, so that its code and state replace it.
func applyOverrides(overrides *exec.StateOverride, precompiles vm.PrecompiledContracts, statePrecompiles map[common.Address]vm.PrecompiledStateContract, stateDB state.StateDB) error {
	if overrides == nil {
		return nil
	}
	for address := range *overrides {
		delete(precompiles, address)
		delete(statePrecompiles, address)
	}
	return overrides.Apply(stateDB)
}

// sanitizeCall defaults the nonce to the sender's, the gas to what the block
// has left, and the fees to zero, and refuses a call over the block's gas.
func (s *simulator) sanitizeCall(call *exec.CallArguments, stateDB state.StateDB, header *evmcore.EvmHeader, gasUsed uint64) error {
	if call.Nonce == nil {
		nonce := stateDB.GetNonce(call.Sender())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if call.Gas == nil {
		remaining := header.GasLimit - gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	if gasUsed+uint64(*call.Gas) > header.GasLimit {
		return &dispatch.Error{Code: codeBlockGasLimitReached,
			Message: fmt.Sprintf("block gas limit reached: %d >= %d", gasUsed+uint64(*call.Gas), header.GasLimit)}
	}
	if call.To == nil {
		if call.BlobHashes != nil || call.BlobFeeCap != nil {
			return core.ErrBlobTxCreate
		}
		if call.AuthorizationList != nil {
			return core.ErrSetCodeTxCreate
		}
	}
	return s.setPriceDefaults(call, header.BaseFee)
}

func (s *simulator) setPriceDefaults(call *exec.CallArguments, baseFee *big.Int) error {
	if call.GasPrice != nil && (call.MaxFeePerGas != nil || call.MaxPriorityFeePerGas != nil) {
		return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	if call.ChainID == nil {
		call.ChainID = (*hexutil.Big)(s.chainConfig.ChainID)
	}
	if call.Value == nil {
		call.Value = new(hexutil.Big)
	}
	switch {
	case baseFee != nil && call.GasPrice == nil:
		if call.MaxFeePerGas == nil {
			call.MaxFeePerGas = new(hexutil.Big)
		}
		if call.MaxPriorityFeePerGas == nil {
			call.MaxPriorityFeePerGas = new(hexutil.Big)
		}
	case call.GasPrice == nil:
		call.GasPrice = new(hexutil.Big)
	}
	return nil
}

// validationError gives the refusal of a simulated call its code.
func validationError(err error) error {
	code := dispatch.CodeInternal
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		code = codeNonceTooHigh
	case errors.Is(err, core.ErrNonceTooLow):
		code = codeNonceTooLow
	case errors.Is(err, core.ErrSenderNoEOA):
		code = codeSenderIsNotEOA
	case errors.Is(err, core.ErrFeeCapTooLow), errors.Is(err, core.ErrTipAboveFeeCap),
		errors.Is(err, core.ErrTipVeryHigh), errors.Is(err, core.ErrFeeCapVeryHigh):
		code = dispatch.CodeInvalidParams
	case errors.Is(err, core.ErrInsufficientFunds):
		code = codeInsufficientFunds
	case errors.Is(err, core.ErrIntrinsicGas):
		code = codeIntrinsicGas
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		code = codeMaxInitCodeSize
	}
	return &dispatch.Error{Code: code, Message: err.Error()}
}

// simulatedChain answers BLOCKHASH with the headers of the simulation and of
// the chain. BLOCKHASH walks back from a block's parent one header at a time,
// at most 256 headers, as the EVM asks for no older one.
type simulatedChain struct {
	simulator *simulator
	previous  []*evmcore.EvmHeader
}

func (c *simulatedChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	base := c.simulator.base
	if base.Number.Uint64() == number && base.Hash == hash {
		return base
	}
	for _, header := range c.previous {
		if header != nil && header.Number.Uint64() == number && header.Hash == hash {
			return header
		}
	}
	header, err := c.simulator.environment.HeaderAt(c.simulator.head, number)
	if err != nil || header.Hash != hash {
		return nil
	}
	return header
}
