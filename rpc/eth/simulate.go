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
	// maxSimulateBlocks bounds the blocks of one simulation, as reth and
	// nethermind do.
	maxSimulateBlocks = 256
	// maxSimulateCallsPerBlock bounds the calls of one simulated block.
	maxSimulateCallsPerBlock = 5000
	// maxSimulateTotalCalls bounds the calls of one simulation.
	maxSimulateTotalCalls = 10000
	// timestampIncrement is the default time between simulated blocks.
	timestampIncrement = 12
)

// JSON-RPC error codes of eth_simulateV1.
const (
	codeReverted                = -32000
	codeVMError                 = -32015
	codeNonceTooHigh            = -38011
	codeNonceTooLow             = -38010
	codeIntrinsicGas            = -38013
	codeInsufficientFunds       = -38014
	codeBlockGasLimitReached    = -38015
	codeBlockNumberInvalid      = -38020
	codeBlockTimestampInvalid   = -38021
	codeSenderIsNotEOA          = -38024
	codeMaxInitCodeSizeExceeded = -38025
	codeClientLimitExceeded     = -38026
)

func simError(code int, msg string) *dispatch.Error {
	return &dispatch.Error{Code: code, Message: msg}
}

// simTxValidationError maps a failed message to its eth_simulateV1 code.
func simTxValidationError(err error) *dispatch.Error {
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
		code = codeMaxInitCodeSizeExceeded
	}
	return simError(code, err.Error())
}

type simBlockOverrides struct {
	Number        *hexutil.Big       `json:"blockNumber"`
	Time          *hexutil.Uint64    `json:"timestamp"`
	GasLimit      *hexutil.Uint64    `json:"gasLimit"`
	FeeRecipient  *common.Address    `json:"feeRecipient"`
	PrevRandao    *common.Hash       `json:"prevRandao"`
	BaseFeePerGas *hexutil.Big       `json:"baseFeePerGas"`
	Withdrawals   *types.Withdrawals `json:"withdrawals"`
}

func (o *simBlockOverrides) applyTo(header *evmcore.EvmHeader) *evmcore.EvmHeader {
	h := *header
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

type simBlock struct {
	BlockOverrides *simBlockOverrides  `json:"blockOverrides"`
	StateOverrides *exec.StateOverride `json:"stateOverrides"`
	Calls          []exec.CallArgs     `json:"calls"`
}

type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
}

type simCallResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

func (r *simCallResult) MarshalJSON() ([]byte, error) {
	type alias simCallResult
	if r.Logs == nil {
		r.Logs = []*types.Log{}
	}
	return json.Marshal((*alias)(r))
}

type simBlockResult struct {
	block    *evmcore.EvmBlock
	calls    []simCallResult
	chainID  *big.Int
	fullTx   bool
	senders  map[common.Hash]common.Address
	receipts []*types.Receipt
}

func (r *simBlockResult) MarshalJSON() ([]byte, error) {
	enc := format.Block(r.block, r.receipts, r.fullTx, r.chainID)
	if r.fullTx {
		for _, raw := range enc.Txs {
			if tx, ok := raw.(*format.Transaction); ok {
				if sender, found := r.senders[tx.Hash]; found {
					tx.From = sender
				}
			}
		}
	}
	blockBytes, err := json.Marshal(enc)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(blockBytes, &fields); err != nil {
		return nil, err
	}
	calls := r.calls
	if calls == nil {
		calls = []simCallResult{}
	}
	callsBytes, err := json.Marshal(calls)
	if err != nil {
		return nil, err
	}
	fields["calls"] = callsBytes
	return json.Marshal(fields)
}

type simOpts struct {
	BlockStateCalls        []simBlock `json:"blockStateCalls"`
	TraceTransfers         bool       `json:"traceTransfers"`
	Validation             bool       `json:"validation"`
	ReturnFullTransactions bool       `json:"returnFullTransactions"`
}

type simulateArgs struct {
	Opts  simOpts         `rpc:"opts"`
	Block *chain.BlockRef `rpc:"block"`
}

// simulateV1 runs calls in simulated blocks on top of a block. It does not
// build on the module's EVM environment: a simulated block is usually above
// the head, BLOCKHASH sees simulated predecessors, and the simulation has a
// precompile set of its own.
func (a *api) simulateV1(ctx context.Context, p simulateArgs) (any, error) {
	opts := p.Opts
	if len(opts.BlockStateCalls) == 0 {
		return nil, simError(dispatch.CodeInvalidParams, "empty input")
	}
	if len(opts.BlockStateCalls) > maxSimulateBlocks {
		return nil, simError(codeClientLimitExceeded, "too many blocks")
	}
	var totalCalls int
	for _, block := range opts.BlockStateCalls {
		if len(block.Calls) > maxSimulateCallsPerBlock {
			return nil, simError(codeClientLimitExceeded, fmt.Sprintf("too many calls in block: %d > %d", len(block.Calls), maxSimulateCallsPerBlock))
		}
		totalCalls += len(block.Calls)
		if totalCalls > maxSimulateTotalCalls {
			return nil, simError(codeClientLimitExceeded, fmt.Sprintf("too many calls: %d > %d", totalCalls, maxSimulateTotalCalls))
		}
	}

	head := a.env.Latest(ctx)
	st, base, err := a.env.StateAt(head, chain.OrLatest(p.Block))
	if err != nil {
		return nil, exec.HeaderNotFound(err)
	}
	defer st.Release()

	gasCap := a.env.Config().GasCap
	if gasCap == 0 {
		gasCap = ^uint64(0)
	}
	sim := &simulator{
		env:            a.env,
		head:           head,
		state:          st,
		base:           base,
		chainConfig:    a.env.ChainConfig(base.Number.Uint64()),
		gp:             core.NewGasPool(gasCap),
		traceTransfers: opts.TraceTransfers,
		validate:       opts.Validation,
		fullTx:         opts.ReturnFullTransactions,
	}
	return sim.execute(ctx, opts.BlockStateCalls)
}

// simChain answers BLOCKHASH of a simulated block: the base block, the
// blocks simulated before, and the chain below. Its block number is that of
// the header it walks from, so it walks at most 256 headers.
type simChain struct {
	env       *exec.Env
	head      *exec.Head
	base      *evmcore.EvmHeader
	processed []*evmcore.EvmHeader
}

func (c *simChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	if c.base.Number.Uint64() == number && c.base.Hash == hash {
		return c.base
	}
	for _, h := range c.processed {
		if h != nil && h.Number.Uint64() == number && h.Hash == hash {
			return h
		}
	}
	header, err := c.env.HeaderAt(c.head, number)
	if err != nil || header.Hash != hash {
		return nil
	}
	return header
}

type simulator struct {
	env            *exec.Env
	head           *exec.Head
	state          state.StateDB
	base           *evmcore.EvmHeader
	chainConfig    *params.ChainConfig
	gp             *core.GasPool
	traceTransfers bool
	validate       bool
	fullTx         bool
}

func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]*simBlockResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	timeout := sim.env.Config().EVMTimeout
	ctx, cancel := exec.WithTimeout(ctx, timeout)
	defer cancel()

	blocks, err := sim.sanitizeChain(blocks)
	if err != nil {
		return nil, err
	}
	headers, err := sim.makeHeaders(blocks)
	if err != nil {
		return nil, err
	}
	results := make([]*simBlockResult, len(blocks))
	parent := sim.base
	for bi, block := range blocks {
		evmBlock, calls, senders, receipts, err := sim.processBlock(ctx, &block, headers[bi], parent, headers[:bi], timeout)
		if err != nil {
			return nil, err
		}
		headers[bi] = evmBlock.Header()
		results[bi] = &simBlockResult{
			block:    evmBlock,
			calls:    calls,
			chainID:  sim.chainConfig.ChainID,
			fullTx:   sim.fullTx,
			senders:  senders,
			receipts: receipts,
		}
		parent = evmBlock.Header()
	}
	return results, nil
}

func (sim *simulator) processBlock(ctx context.Context, block *simBlock, header, parent *evmcore.EvmHeader, prevHeaders []*evmcore.EvmHeader, timeout time.Duration) (*evmcore.EvmBlock, []simCallResult, map[common.Hash]common.Address, []*types.Receipt, error) {
	if parent == nil {
		return nil, nil, nil, nil, errors.New("parent header is nil")
	}
	header.ParentHash = parent.Hash
	if header.BaseFee == nil {
		if sim.validate && parent.BaseFee != nil {
			header.BaseFee = new(big.Int).Set(parent.BaseFee)
		} else {
			header.BaseFee = big.NewInt(0)
		}
	}
	blockChain := &simChain{env: sim.env, head: sim.head, base: sim.base, processed: prevHeaders}
	blockContext := evmcore.NewEVMBlockContext(header, blockChain, nil)
	precompiles := sim.activePrecompiles(sim.base)
	if err := sim.applyStateOverrides(block.StateOverrides, precompiles, sim.state); err != nil {
		return nil, nil, nil, nil, err
	}

	var (
		gasUsed     uint64
		txes        = make([]*types.Transaction, len(block.Calls))
		callResults = make([]simCallResult, len(block.Calls))
		senders     = make(map[common.Hash]common.Address)
		receipts    = make([]*types.Receipt, len(block.Calls))
		vmConfig    = vm.Config{NoBaseFee: !sim.validate}
	)
	tracer := newSimTracer(sim.traceTransfers, blockContext.BlockNumber.Uint64(), blockContext.Time)
	vmConfig.Tracer = tracer.hooks()
	activeState := exec.WithStateHooks(sim.state, tracer.hooks())
	evm := vm.NewEVM(blockContext, activeState, sim.chainConfig, vmConfig)
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	// EIP-2935: the parent hash goes into the history contract.
	if sim.chainConfig.IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, activeState)
	}

	for i, call := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, nil, err
		}
		if err := sim.sanitizeCall(&call, activeState, header, &gasUsed); err != nil {
			return nil, nil, nil, nil, err
		}
		tx, err := call.ToTransaction()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		txHash := tx.Hash()
		txes[i] = tx
		senders[txHash] = call.Sender()
		tracer.reset(txHash, uint(i))
		activeState.SetTxContext(txHash, i)

		msg, err := call.ToMessage(sim.gp.Gas(), header.BaseFee)
		if err != nil {
			return nil, nil, nil, nil, simTxValidationError(err)
		}
		msg.SkipNonceChecks = !sim.validate
		msg.SkipTransactionChecks = !sim.validate
		result, err := applySimMessage(ctx, evm, msg, timeout, sim.gp)
		if err != nil {
			return nil, nil, nil, nil, simTxValidationError(err)
		}
		gasUsed += result.UsedGas

		callRes := simCallResult{ReturnValue: result.ReturnData, GasUsed: hexutil.Uint64(result.UsedGas), Logs: tracer.logs()}
		if result.Failed() {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusFailed)
			if errors.Is(result.Err, vm.ErrExecutionReverted) {
				revertErr := exec.NewRevertError(result)
				callRes.Error = &callError{Message: revertErr.Error(), Code: codeReverted, Data: revertErr.ErrorData().(string)}
			} else {
				callRes.Error = &callError{Message: result.Err.Error(), Code: codeVMError}
			}
		} else {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
		}
		callResults[i] = callRes
		receipt := &types.Receipt{Status: uint64(callRes.Status), Logs: callRes.Logs}
		receipt.Bloom = types.CreateBloom(receipt)
		receipts[i] = receipt
	}

	header.GasUsed = gasUsed
	header.Root = activeState.GetStateHash()
	evmBlock := evmcore.NewEvmBlock(header, txes)
	evmBlock.Hash = evmBlock.EthHeader().Hash()
	for i := range callResults {
		for _, l := range callResults[i].Logs {
			l.BlockHash = evmBlock.Hash
			l.BlockNumber = evmBlock.NumberU64()
			l.BlockTimestamp = uint64(evmBlock.Time.Unix())
		}
	}
	return evmBlock, callResults, senders, receipts, nil
}

func applySimMessage(ctx context.Context, evm *vm.EVM, msg *core.Message, timeout time.Duration, gp *core.GasPool) (*core.ExecutionResult, error) {
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()
	result, err := core.ApplyMessage(evm, msg, gp)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

// applyStateOverrides applies the overrides of a simulated block; an
// overridden precompile becomes an ordinary account.
func (sim *simulator) applyStateOverrides(overrides *exec.StateOverride, precompiles vm.PrecompiledContracts, st state.StateDB) error {
	if overrides == nil {
		return nil
	}
	for addr, account := range *overrides {
		delete(precompiles, addr)
		if account.Nonce != nil {
			st.SetNonce(addr, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		if account.Code != nil {
			st.SetCode(addr, *account.Code, tracing.CodeChangeUnspecified)
		}
		if account.Balance != nil && *account.Balance != nil {
			st.SetBalance(addr, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		if account.State != nil {
			st.SetStorage(addr, *account.State)
		}
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				st.SetState(addr, key, value)
			}
		}
	}
	st.Finalise(false)
	return nil
}

func (sim *simulator) sanitizeCall(call *exec.CallArgs, st state.StateDB, header *evmcore.EvmHeader, gasUsed *uint64) error {
	if call.Nonce == nil {
		nonce := st.GetNonce(call.Sender())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if call.Gas == nil {
		remaining := header.GasLimit - *gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	if *gasUsed+uint64(*call.Gas) > header.GasLimit {
		return simError(codeBlockGasLimitReached, fmt.Sprintf("block gas limit reached: %d >= %d", *gasUsed+uint64(*call.Gas), header.GasLimit))
	}
	// A blob or set-code call without a recipient cannot be a transaction.
	if call.To == nil {
		if call.BlobHashes != nil || call.BlobFeeCap != nil {
			return core.ErrBlobTxCreate
		}
		if call.AuthorizationList != nil {
			return core.ErrSetCodeTxCreate
		}
	}
	return sim.setCallPriceDefaults(call, header.BaseFee)
}

func (sim *simulator) setCallPriceDefaults(call *exec.CallArgs, baseFee *big.Int) error {
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
		// Zero fees by default, so that calls succeed without funds.
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

func (sim *simulator) activePrecompiles(base *evmcore.EvmHeader) vm.PrecompiledContracts {
	rules := sim.chainConfig.Rules(base.Number, base.PrevRandao != (common.Hash{}), uint64(base.Time.Unix()))
	return vm.ActivePrecompiledContracts(rules)
}

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
			return nil, simError(codeBlockNumberInvalid, fmt.Sprintf("block numbers must be in order: %d <= %d",
				block.BlockOverrides.Number.ToInt().Uint64(), prevNumber.Uint64()))
		}
		if total := new(big.Int).Sub(block.BlockOverrides.Number.ToInt(), base.Number); total.Cmp(big.NewInt(maxSimulateBlocks)) > 0 {
			return nil, simError(codeClientLimitExceeded, "too many blocks")
		}
		// Fill a gap with empty blocks.
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
				return nil, simError(codeBlockTimestampInvalid, fmt.Sprintf("block timestamps must be in order: %d <= %d", t, prevTimestamp))
			}
			prevTimestamp = t
		}
		res = append(res, block)
	}
	return res, nil
}

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
		var withdrawalsHash *common.Hash
		if sim.chainConfig.IsShanghai(number, timestamp) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		template := &evmcore.EvmHeader{
			Number:          number,
			Time:            inter.FromUnix(int64(timestamp)),
			Coinbase:        prev.Coinbase,
			GasLimit:        prev.GasLimit,
			BaseFee:         prev.BaseFee,
			PrevRandao:      prev.PrevRandao,
			WithdrawalsHash: withdrawalsHash,
		}
		header := overrides.applyTo(template)
		res[bi] = header
		prev = header
	}
	return res, nil
}

var (
	// simTransferTopic is keccak256("Transfer(address,address,uint256)").
	simTransferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// simTransferAddress is the ERC-7528 address of native transfers.
	simTransferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

// simTracer collects the logs of a simulated call, and with traceTransfers
// one log per native transfer; the logs of a reverted frame are dropped.
type simTracer struct {
	calls          [][]*types.Log
	count          int
	traceTransfers bool
	blockNumber    uint64
	blockTimestamp uint64
	txHash         common.Hash
	txIdx          uint
}

func newSimTracer(traceTransfers bool, blockNumber, blockTimestamp uint64) *simTracer {
	return &simTracer{traceTransfers: traceTransfers, blockNumber: blockNumber, blockTimestamp: blockTimestamp}
}

func (t *simTracer) hooks() *tracing.Hooks {
	return &tracing.Hooks{OnEnter: t.onEnter, OnExit: t.onExit, OnLog: t.onLog}
}

func (t *simTracer) onEnter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.calls = append(t.calls, nil)
	if vm.OpCode(typ) != vm.DELEGATECALL && value != nil && value.Cmp(common.Big0) > 0 && t.traceTransfers {
		topics := []common.Hash{simTransferTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(to.Bytes())}
		t.captureLog(simTransferAddress, topics, common.BigToHash(value).Bytes())
	}
}

func (t *simTracer) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if reverted {
		t.calls = t.calls[:len(t.calls)-1]
	}
}

func (t *simTracer) onLog(log *types.Log) {
	t.captureLog(log.Address, log.Topics, log.Data)
}

func (t *simTracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	last := len(t.calls) - 1
	t.calls[last] = append(t.calls[last], &types.Log{
		Address:        address,
		Topics:         topics,
		Data:           data,
		BlockNumber:    t.blockNumber,
		BlockTimestamp: t.blockTimestamp,
		TxHash:         t.txHash,
		TxIndex:        t.txIdx,
		Index:          uint(t.count),
	})
	t.count++
}

func (t *simTracer) reset(txHash common.Hash, txIdx uint) {
	t.calls = nil
	t.txHash = txHash
	t.txIdx = txIdx
}

func (t *simTracer) logs() []*types.Log {
	logs := []*types.Log{}
	for _, call := range t.calls {
		logs = append(logs, call...)
	}
	return logs
}
