// Copyright 2025 Sonic Operations Ltd
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

package txtrace

import (
	"errors"
	"math/big"
	"strings"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

const (
	CALL         = "call"
	CREATE       = "create"
	SELFDESTRUCT = "suicide"
)

// TraceStructLogger is a transaction trace data collector
type TraceStructLogger struct {
	blockHash   common.Hash
	tx          common.Hash
	txIndex     uint
	blockNumber big.Int

	gasLimit  uint64
	rootTrace *CallTrace
}

// CallTrace is struct for holding tracing results
type CallTrace struct {
	Actions []ActionTrace  `json:"result"`
	Stack   []*ActionTrace `json:"-"`
}

// ActionTrace represents single interaction with blockchain
type ActionTrace struct {
	childTraces         []*ActionTrace     `json:"-"`
	Action              *AddressAction     `json:"action"`
	BlockHash           common.Hash        `json:"blockHash"`
	BlockNumber         big.Int            `json:"blockNumber"`
	Result              *TraceActionResult `json:"result,omitempty"`
	Error               string             `json:"error,omitempty"`
	Subtraces           uint64             `json:"subtraces"`
	TraceAddress        []uint32           `json:"traceAddress"`
	TransactionHash     common.Hash        `json:"transactionHash"`
	TransactionPosition uint64             `json:"transactionPosition"`
	TraceType           string             `json:"type"`
}

// AddressAction represents more specific information about
// account interaction
type AddressAction struct {
	CallType      *string         `json:"callType,omitempty"`
	From          *common.Address `json:"from"`
	To            *common.Address `json:"to,omitempty"`
	Value         hexutil.Big     `json:"value"`
	Gas           hexutil.Uint64  `json:"gas"`
	Init          *hexutil.Bytes  `json:"init,omitempty"`
	Input         *hexutil.Bytes  `json:"input,omitempty"`
	Address       *common.Address `json:"address,omitempty"`
	RefundAddress *common.Address `json:"refund_address,omitempty"`
	Balance       *hexutil.Big    `json:"balance,omitempty"`
}

// TraceActionResult holds information related to result of the
// processed transaction
type TraceActionResult struct {
	GasUsed hexutil.Uint64  `json:"gasUsed"`
	Output  *hexutil.Bytes  `json:"output,omitempty"`
	Code    *hexutil.Bytes  `json:"code,omitempty"`
	Address *common.Address `json:"address,omitempty"`
}

// NewTraceStructLogger creates new instance of trace creator
func NewTraceStructLogger(block *evmcore.EvmBlock, index uint) *TraceStructLogger {
	traceStructLogger := TraceStructLogger{
		blockHash:   block.Hash,
		blockNumber: *block.Number,
		txIndex:     index,
	}
	return &traceStructLogger
}

// NewActionTrace creates new instance of type ActionTrace
func (tr *TraceStructLogger) NewActionTrace(tType string) *ActionTrace {
	return &ActionTrace{
		BlockHash:           tr.blockHash,
		BlockNumber:         tr.blockNumber,
		TransactionHash:     tr.tx,
		TransactionPosition: uint64(tr.txIndex),
		TraceType:           tType,
		Result:              &TraceActionResult{},
	}
}

// NewAddressAction creates specific information about trace addresses
func NewAddressAction(from common.Address, gas uint64, data []byte, to *common.Address, value hexutil.Big, callType *string) *AddressAction {
	action := AddressAction{
		From:     &from,
		To:       to,
		Gas:      hexutil.Uint64(gas),
		Value:    value,
		CallType: callType,
	}
	inputHex := hexutil.Bytes(common.CopyBytes(data))
	if callType == nil {
		action.Init = &inputHex
	} else {
		action.Input = &inputHex
	}
	return &action
}

func (tr *TraceStructLogger) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart: tr.OnTxStart,
		OnTxEnd:   tr.OnTxEnd,
		OnEnter:   tr.OnEnter,
		OnExit:    tr.OnExit,
	}
}

// OnTxStart implements the tracer interface to initialize the tracing operation.
func (tr *TraceStructLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Tracer OnTxStart failed", r)
		}
	}()

	tr.tx = tx.Hash()
	tr.gasLimit = tx.Gas()

	// Create main trace holder
	txTrace := CallTrace{
		Actions: make([]ActionTrace, 0),
	}

	// Add root object into Tracer
	tr.rootTrace = &txTrace
}

// OnEnter implements tracer interface for entering call
func (tr *TraceStructLogger) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Tracer OnEnter failed", r)
		}
	}()
	var (
		trace *ActionTrace
	)

	if tr.rootTrace == nil {
		log.Debug("There is no root trace", "tx hash", tr.tx.String())
		return
	}

	if value == nil {
		value = big.NewInt(0)
	}

	// Match processed instruction and create trace based on it
	switch vm.OpCode(typ) {
	case vm.CREATE, vm.CREATE2:

		trace = tr.NewActionTrace(CREATE)
		traceAction := NewAddressAction(from, gas, input, &to, hexutil.Big(*value), nil)
		trace.Action = traceAction

	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:

		trace = tr.NewActionTrace(CALL)
		callType := strings.ToLower(vm.OpCode(typ).String())
		traceAction := NewAddressAction(from, gas, input, &to, hexutil.Big(*value), &callType)
		trace.Action = traceAction

	case vm.SELFDESTRUCT:

		trace = tr.NewActionTrace(SELFDESTRUCT)
		traceAction := NewAddressAction(from, gas, input, nil, hexutil.Big(*value), nil)
		traceAction.Address = &from
		traceAction.RefundAddress = &to
		traceAction.Balance = (*hexutil.Big)(value)
		trace.Action = traceAction
	}
	if depth == 0 {
		tr.rootTrace.Actions = append(tr.rootTrace.Actions, *trace)
		tr.rootTrace.Stack = append(tr.rootTrace.Stack, &tr.rootTrace.Actions[0])
	} else {
		tr.rootTrace.Stack = append(tr.rootTrace.Stack, trace)
	}

}

// OnExit is called when returning from a call
func (tr *TraceStructLogger) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Tracer OnExit failed", r)
		}
	}()

	if tr.rootTrace == nil {
		log.Debug("There is no root trace when OnExit", "tx hash", tr.tx.String())
		return
	}

	size := len(tr.rootTrace.Stack)
	if size < 1 {
		log.Debug("OnExit does not match with number of OnEnter", "tx hash", tr.tx.String())
		return
	}

	trace := tr.rootTrace.Stack[size-1]
	tr.rootTrace.Stack = tr.rootTrace.Stack[:size-1]

	if depth == 1 {
		tr.rootTrace.Actions[0].childTraces = append(tr.rootTrace.Actions[0].childTraces, trace)
	} else if depth > 1 {

		parent := tr.rootTrace.Stack[len(tr.rootTrace.Stack)-1]
		parent.childTraces = append(parent.childTraces, trace)

	}

	trace.processOutput(output, err, false)

	result := trace.Result
	if result != nil {
		result.GasUsed = hexutil.Uint64(gasUsed)
	}
}

// OnTxEnd is called after the call finishes to finalize the tracing.
func (tr *TraceStructLogger) OnTxEnd(receipt *types.Receipt, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Tracer OnTxEnd failed", r)
		}
	}()

	if tr.rootTrace != nil && tr.rootTrace.lastTrace() != nil {

		trace := tr.rootTrace.lastTrace()
		if trace.Result != nil {
			trace.Result.GasUsed = hexutil.Uint64(receipt.GasUsed)
		}
	}

	if tr.rootTrace != nil {

		tr.rootTrace.processTraces()
	}
}

// Handle output data and error
func (trace *ActionTrace) processOutput(output []byte, err error, rootTrace bool) {
	output = common.CopyBytes(output)
	if err == nil {
		switch trace.TraceType {
		case CREATE:
			trace.Result.Code = (*hexutil.Bytes)(&output)
			if !rootTrace {
				trace.Result.Address = trace.Action.To
				trace.Action.To = nil
			}
		case CALL:
			trace.Result.Output = (*hexutil.Bytes)(&output)
		default:
		}
		return
	} else {
		trace.Result = nil
	}

	trace.Error = err.Error()
	if trace.TraceType == CREATE {
		trace.Action.To = nil
	}
	if traceError, ok := traceErrorMapping[err.Error()]; ok {
		trace.Error = traceError
		return
	}

	switch err.(type) {
	case *vm.ErrStackOverflow:
		trace.Error = "Out of stack"
		return
	}

	if !errors.Is(err, vm.ErrExecutionReverted) || len(output) == 0 {
		return
	}
	if len(output) < 4 {
		return
	}
	if unpacked, err := abi.UnpackRevert(output); err == nil {
		trace.Error = unpacked
	}
}

// GetResult returns action traces after recording evm process
func (tr *TraceStructLogger) GetResult() *[]ActionTrace {
	if tr.rootTrace != nil {
		return &tr.rootTrace.Actions
	}
	empty := make([]ActionTrace, 0)
	return &empty
}

// AddTrace Append trace to call trace list
func (callTrace *CallTrace) AddTrace(blockTrace *ActionTrace) {
	if callTrace.Actions == nil {
		callTrace.Actions = make([]ActionTrace, 0)
	}
	callTrace.Actions = append(callTrace.Actions, *blockTrace)
}

// AddTraces Append traces to call trace list
func (callTrace *CallTrace) AddTraces(traces *[]ActionTrace, traceIndex *[]hexutil.Uint) {
	for _, trace := range *traces {
		if traceIndex == nil || isIndexEqual(traceIndex, trace.TraceAddress) {
			callTrace.AddTrace(&trace)
		}
	}
}

// isIndexEqual tells whether index and traceIndex are the same
func isIndexEqual(index *[]hexutil.Uint, traceIndex []uint32) bool {
	if len(*index) != len(traceIndex) {
		return false
	}
	for i, v := range *index {
		if uint32(v) != traceIndex[i] {
			return false
		}
	}
	return true
}

// lastTrace Get last trace in call trace list
func (callTrace *CallTrace) lastTrace() *ActionTrace {
	if len(callTrace.Actions) > 0 {
		return &callTrace.Actions[len(callTrace.Actions)-1]
	}
	return nil
}

// processTraces initiates final information distribution
// accros result traces
func (callTrace *CallTrace) processTraces() {
	trace := &callTrace.Actions[len(callTrace.Actions)-1]
	callTrace.processTrace(trace, []uint32{})
}

// processTrace goes thru all trace results and sets info
func (callTrace *CallTrace) processTrace(trace *ActionTrace, traceAddress []uint32) {
	trace.TraceAddress = traceAddress
	trace.Subtraces = uint64(len(trace.childTraces))
	for i, childTrace := range trace.childTraces {
		childAddress := childTraceAddress(traceAddress, i)
		childTrace.TraceAddress = childAddress
		callTrace.AddTrace(childTrace)
		callTrace.processTrace(callTrace.lastTrace(), childAddress)
	}
}

func childTraceAddress(a []uint32, i int) []uint32 {
	child := make([]uint32, 0, len(a)+1)
	child = append(child, a...)
	child = append(child, uint32(i))
	return child
}

func CreateActionTrace(bHash common.Hash, bNumber big.Int, tHash common.Hash, tPos uint64, tType string) *ActionTrace {
	return &ActionTrace{
		BlockHash:           bHash,
		BlockNumber:         bNumber,
		TransactionHash:     tHash,
		TransactionPosition: tPos,
		TraceType:           tType,
		TraceAddress:        make([]uint32, 0),
		Result:              &TraceActionResult{},
	}
}

// GetErrorTraceFromMsg constructs filled error trace
func GetErrorTraceFromMsg(msg *core.Message, blockHash common.Hash, blockNumber big.Int, txHash common.Hash, index uint64, err error) *ActionTrace {
	if msg == nil {
		return createErrorTrace(blockHash, blockNumber, nil, &common.Address{}, txHash, 0, []byte{}, hexutil.Big{}, index, err)
	} else {
		from := msg.From
		return createErrorTrace(blockHash, blockNumber, &from, msg.To, txHash, msg.GasLimit, msg.Data, hexutil.Big(*msg.Value), index, err)
	}
}

// createErrorTrace constructs filled error trace
func createErrorTrace(blockHash common.Hash, blockNumber big.Int,
	from *common.Address, to *common.Address,
	txHash common.Hash, gas uint64, input []byte,
	value hexutil.Big,
	index uint64, err error) *ActionTrace {

	var blockTrace *ActionTrace
	var txAction *AddressAction

	if from == nil {
		from = &common.Address{}
	}

	callType := CALL
	if to != nil {
		blockTrace = CreateActionTrace(blockHash, blockNumber, txHash, index, CALL)
		txAction = NewAddressAction(*from, gas, input, to, value, &callType)
	} else {
		blockTrace = CreateActionTrace(blockHash, blockNumber, txHash, index, CREATE)
		txAction = NewAddressAction(*from, gas, input, nil, value, nil)
	}
	blockTrace.Action = txAction
	blockTrace.Result = nil
	if err != nil {
		blockTrace.Error = err.Error()
	} else {
		blockTrace.Error = "Reverted"
	}
	return blockTrace
}

var traceErrorMapping = map[string]string{
	vm.ErrCodeStoreOutOfGas.Error():     "Out of gas",
	vm.ErrOutOfGas.Error():              "Out of gas",
	vm.ErrGasUintOverflow.Error():       "Out of gas",
	vm.ErrMaxCodeSizeExceeded.Error():   "Out of gas",
	vm.ErrInvalidJump.Error():           "Bad jump destination",
	vm.ErrExecutionReverted.Error():     "Reverted",
	vm.ErrReturnDataOutOfBounds.Error(): "Out of bounds",
	"precompiled failed":                "Built-in failed",
	"invalid input length":              "Built-in failed",
}
