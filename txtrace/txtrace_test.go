package txtrace

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	blockNumber     = big.NewInt(123)
	blockHash       = common.HexToHash("0x123")
	from            = common.HexToAddress("1")
	to              = common.HexToAddress("2")
	toInner         = common.HexToAddress("3")
	inputData       = common.Hex2Bytes("2f7468610000000000000000000000000000000000000000000000000000000000000008")
	inputDataInner  = common.Hex2Bytes("2f7468610000000000000000000000000000000000000000000000000000000000000002")
	outputData      = common.Hex2Bytes("456")
	outputDataInner = common.Hex2Bytes("789")

	txIndex  uint     = 3
	nonce    uint64   = 0
	gaslimit uint64   = 3000000
	gasUsed  uint64   = 2000
	gasprice *big.Int = big.NewInt(100)
	value    *big.Int = big.NewInt(5)
)

func TestTracerSimpleCall(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CALL), from, to, inputData, gaslimit, value)
	tracer.OnExit(0, outputData, 100, nil, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, nil)

	want := `[
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000001",
            "to": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x2dc6c0",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000008"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x7d0",
            "output": "0x45"
        },
        "subtraces": 0,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func TestTracerSimpleCreate(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CREATE), from, to, inputData, gaslimit, value)
	tracer.OnExit(0, outputData, 100, nil, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, nil)

	want := `[
    {
        "action": {
            "from": "0x0000000000000000000000000000000000000001",
            "value": "0x5",
            "gas": "0x2dc6c0",
            "init": "0x2f7468610000000000000000000000000000000000000000000000000000000000000008"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x7d0",
            "code": "0x45",
            "address": "0x0000000000000000000000000000000000000002"
        },
        "subtraces": 0,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "create"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func TestTracerComplexCall(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CALL), from, to, inputData, gaslimit, value)

	tracer.OnEnter(1, byte(vm.CREATE2), to, toInner, inputDataInner, 600, value)

	tracer.OnEnter(2, byte(vm.CALL), to, toInner, inputDataInner, 601, value)
	tracer.OnExit(2, outputDataInner, 401, nil, false)

	tracer.OnEnter(2, byte(vm.STATICCALL), to, toInner, inputDataInner, 602, value)
	tracer.OnExit(2, outputDataInner, 402, nil, false)

	tracer.OnEnter(2, byte(vm.DELEGATECALL), to, toInner, inputDataInner, 603, value)
	tracer.OnExit(2, outputDataInner, 403, nil, false)

	tracer.OnEnter(2, byte(vm.SELFDESTRUCT), to, toInner, inputDataInner, 604, value)
	tracer.OnExit(2, outputDataInner, 404, nil, false)

	tracer.OnExit(1, outputDataInner, 400, nil, false)

	tracer.OnEnter(1, byte(vm.CREATE), to, toInner, inputDataInner, 600, value)
	tracer.OnExit(1, outputDataInner, 406, nil, false)

	tracer.OnExit(0, outputData, 100, nil, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, nil)

	want := `[
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000001",
            "to": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x2dc6c0",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000008"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x7d0",
            "output": "0x45"
        },
        "subtraces": 2,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    },
    {
        "action": {
            "from": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x258",
            "init": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x190",
            "code": "0x78",
            "address": "0x0000000000000000000000000000000000000003"
        },
        "subtraces": 4,
        "traceAddress": [
            0
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "create"
    },
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000002",
            "to": "0x0000000000000000000000000000000000000003",
            "value": "0x5",
            "gas": "0x259",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x191",
            "output": "0x78"
        },
        "subtraces": 0,
        "traceAddress": [
            0,
            0
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    },
    {
        "action": {
            "callType": "staticcall",
            "from": "0x0000000000000000000000000000000000000002",
            "to": "0x0000000000000000000000000000000000000003",
            "value": "0x5",
            "gas": "0x25a",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x192",
            "output": "0x78"
        },
        "subtraces": 0,
        "traceAddress": [
            0,
            1
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    },
    {
        "action": {
            "callType": "delegatecall",
            "from": "0x0000000000000000000000000000000000000002",
            "to": "0x0000000000000000000000000000000000000003",
            "value": "0x5",
            "gas": "0x25b",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x193",
            "output": "0x78"
        },
        "subtraces": 0,
        "traceAddress": [
            0,
            2
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    },
    {
        "action": {
            "from": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x25c",
            "init": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002",
            "address": "0x0000000000000000000000000000000000000002",
            "refund_address": "0x0000000000000000000000000000000000000003",
            "balance": "0x5"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x194"
        },
        "subtraces": 0,
        "traceAddress": [
            0,
            3
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "suicide"
    },
    {
        "action": {
            "from": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x258",
            "init": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x196",
            "code": "0x78",
            "address": "0x0000000000000000000000000000000000000003"
        },
        "subtraces": 0,
        "traceAddress": [
            1
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "create"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func TestTracerZeroValues(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CALL), from, to, []byte{}, gaslimit, nil)
	tracer.OnExit(0, []byte{}, 100, nil, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, nil)

	want := `[
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000001",
            "to": "0x0000000000000000000000000000000000000002",
            "value": "0x0",
            "gas": "0x2dc6c0",
            "input": "0x"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x7d0",
            "output": "0x"
        },
        "subtraces": 0,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func TestTracerSimpleErrorCall(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CALL), from, to, inputData, gaslimit, value)
	tracer.OnExit(0, outputData, 100, vm.ErrExecutionReverted, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, vm.ErrExecutionReverted)

	want := `[
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000001",
            "to": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x2dc6c0",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000008"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "error": "Reverted",
        "subtraces": 0,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func TestTracerInnerErrorCall(t *testing.T) {

	block, tx := getDefaultBlockTxMessage()
	tracer := NewTraceStructLogger(block, txIndex)

	tracer.OnTxStart(nil, tx, from)
	tracer.OnEnter(0, byte(vm.CALL), from, to, inputData, gaslimit, value)
	tracer.OnEnter(1, byte(vm.CALL), to, toInner, inputDataInner, 200, value)
	tracer.OnExit(1, outputDataInner, 201, vm.ErrExecutionReverted, false)
	tracer.OnExit(0, outputData, 100, nil, false)
	tracer.OnTxEnd(&types.Receipt{GasUsed: gasUsed}, nil)

	want := `[
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000001",
            "to": "0x0000000000000000000000000000000000000002",
            "value": "0x5",
            "gas": "0x2dc6c0",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000008"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "result": {
            "gasUsed": "0x7d0",
            "output": "0x45"
        },
        "subtraces": 1,
        "traceAddress": [],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    },
    {
        "action": {
            "callType": "call",
            "from": "0x0000000000000000000000000000000000000002",
            "to": "0x0000000000000000000000000000000000000003",
            "value": "0x5",
            "gas": "0xc8",
            "input": "0x2f7468610000000000000000000000000000000000000000000000000000000000000002"
        },
        "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000123",
        "blockNumber": 123,
        "error": "Reverted",
        "subtraces": 0,
        "traceAddress": [
            0
        ],
        "transactionHash": "0xb3a9e46933c0c55b3e9facb9d291b1c606ffa59acbdc9b58540130155b0699ec",
        "transactionPosition": 3,
        "type": "call"
    }
]`
	checkResult(t, tracer.GetResult(), want)
}

func getDefaultBlockTxMessage() (*evmcore.EvmBlock, *types.Transaction) {

	// create transaction with default values
	tx := types.NewTransaction(nonce, to, value, gaslimit, gasprice, inputData)

	// create block
	block := evmcore.NewEvmBlock(&evmcore.EvmHeader{
		Number: blockNumber,
		Hash:   blockHash}, types.Transactions{tx})

	return block, tx
}

func checkResult(t *testing.T, traces *[]ActionTrace, expectedTraces string) {
	result, err := json.MarshalIndent(&traces, "", "    ")
	if err != nil {
		t.Errorf("problem with formating result, got error: %v", err)
	}

	if expectedTraces != string(result) {
		t.Errorf("expected result is not the same as output got: %v, want: %v", string(result), expectedTraces)
	}
}
