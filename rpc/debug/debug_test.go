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
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// fixture serves the debug namespace over the replay chain.
type fixture struct {
	*tests.ReplayChain
	h *tests.Harness
}

func newFixture(t *testing.T, mutate ...func(*dispatch.Config)) *fixture {
	t.Helper()
	r := tests.NewReplayChain(t)
	return &fixture{ReplayChain: r, h: newHarness(t, r.Core, mutate...)}
}

func newHarness(t *testing.T, fc *tests.FakeCore, mutate ...func(*dispatch.Config)) *tests.Harness {
	t.Helper()
	return tests.NewEnvHarness(t, fc, func(d *dispatch.Dispatcher, env *exec.Env) { Register(d, env) }, mutate...)
}

func TestTraceTransaction_DefaultStructLogger(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Gas        uint64 `json:"gas"`
		Failed     bool   `json:"failed"`
		StructLogs []struct {
			Op string `json:"op"`
		} `json:"structLogs"`
	}
	f.h.Into(&res, "debug_traceTransaction", f.Store.Hash())
	require.False(t, res.Failed)
	var ops []string
	for _, l := range res.StructLogs {
		ops = append(ops, l.Op)
	}
	require.Contains(t, ops, "SSTORE")
	require.Len(t, ops, 4) // PUSH1 PUSH1 SSTORE STOP

	// The struct log is cut at the configured limit, which also caps a larger
	// limit of the client. In the go-ethereum version of the module the limit
	// is a size in bytes: 2 lets only the first log through.
	f = newFixture(t, func(c *dispatch.Config) { c.StructLogLimit = 2 })
	for _, config := range []string{`{}`, `{"limit":0}`, `{"limit":100000}`} {
		res.StructLogs = nil
		f.h.Into(&res, "debug_traceTransaction", f.Store.Hash(), json.RawMessage(config))
		require.Len(t, res.StructLogs, 1, config)
	}
}

func TestTraceTransaction_CallTracer(t *testing.T) {
	f := newFixture(t)
	var res struct {
		From  common.Address `json:"from"`
		To    common.Address `json:"to"`
		Value hexutil.Big    `json:"value"`
		Type  string         `json:"type"`
	}
	f.h.Into(&res, "debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "callTracer"})
	require.Equal(t, f.Alice.Addr, res.From)
	require.Equal(t, f.Bob, res.To)
	require.Equal(t, int64(5), res.Value.ToInt().Int64())
	require.Equal(t, "CALL", res.Type)
}

func TestTraceTransaction_Errors(t *testing.T) {
	f := newFixture(t)
	err := f.h.Fails("debug_traceTransaction", common.Hash{9})
	require.Equal(t, "transaction "+common.Hash{9}.Hex()+" not found", err.Message)

	// Only built-in tracers run unless the operator allows others.
	err = f.h.Fails("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "{result: function() { return 1 }, fault: function() {}}"})
	require.Equal(t, "custom tracer is not permitted", err.Message)
	err = f.h.Fails("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "nope"})
	require.Equal(t, "custom tracer is not permitted", err.Message)
}

func TestTraceTransaction_TimeoutIsCheckedWhereTheTracerIsBuilt(t *testing.T) {
	f := newFixture(t, func(c *dispatch.Config) { c.EVMTimeout = time.Minute })
	err := f.h.Fails("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "callTracer", "timeout": "5m"})
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
	require.Contains(t, err.Message, "exceeds the limit")
	err = f.h.Fails("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "callTracer", "timeout": "soon"})
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
	f.h.Result("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "callTracer", "timeout": "30s"})

	// The subject is found first: an unknown transaction is reported as such.
	err = f.h.Fails("debug_traceTransaction", common.Hash{9}, map[string]any{"timeout": "5m"})
	require.Contains(t, err.Message, "not found")
	// ... and the tracer name before the timeout.
	err = f.h.Fails("debug_traceTransaction", f.Transfer.Hash(), map[string]any{"tracer": "nope", "timeout": "5m"})
	require.Equal(t, "custom tracer is not permitted", err.Message)
}

func TestTraceTransaction_ResponseSizeIsLimited(t *testing.T) {
	f := newFixture(t, func(c *dispatch.Config) { c.MaxResponseSize = 10 })
	err := f.h.Fails("debug_traceTransaction", f.Store.Hash())
	require.Equal(t, ErrMaxResponseSize.Error(), err.Message)
}

func TestTraceBlock(t *testing.T) {
	f := newFixture(t)
	for _, call := range [][]any{
		{"debug_traceBlockByNumber", "latest", map[string]any{"tracer": "callTracer"}},
		{"debug_traceBlockByNumber", "0x2"},
		{"debug_traceBlockByHash", f.Core.BlockAt(2).Hash, map[string]any{"tracer": "callTracer"}},
	} {
		var res []struct {
			TxHash common.Hash `json:"txHash"`
			Result any         `json:"result"`
		}
		f.h.Into(&res, call[0].(string), call[1:]...)
		require.Len(t, res, 2, call[0])
		require.Equal(t, f.Transfer.Hash(), res[0].TxHash)
		require.Equal(t, f.Store.Hash(), res[1].TxHash)
	}

	err := f.h.Fails("debug_traceBlockByNumber", "0x0")
	require.Equal(t, "genesis is not traceable", err.Message)
	err = f.h.Fails("debug_traceBlockByNumber", "0x99")
	require.Equal(t, "block #153 not found", err.Message)
	err = f.h.Fails("debug_traceBlockByHash", common.Hash{9})
	require.Equal(t, "block "+common.Hash{9}.Hex()+" not found", err.Message)
}

func TestTraceCall(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Type string         `json:"type"`
		To   common.Address `json:"to"`
	}
	f.h.Into(&res, "debug_traceCall", map[string]any{"from": f.Alice.Addr, "to": f.Contract, "gas": "0x30000"}, "latest",
		map[string]any{"tracer": "callTracer"})
	require.Equal(t, f.Contract, res.To)

	f.h.Result("debug_traceCall", map[string]any{"from": f.Alice.Addr, "to": f.Bob}, f.Core.BlockAt(1).Hash)

	err := f.h.Fails("debug_traceCall", map[string]any{"to": f.Bob}, "pending")
	require.Equal(t, "tracing on top of pending is not supported", err.Message)
	// Missing blocks are "header not found", whatever the call object holds.
	bad := map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"}
	err = f.h.Fails("debug_traceCall", bad, "0x99")
	require.Equal(t, "header not found", err.Message)
	err = f.h.Fails("debug_traceCall", bad, common.Hash{9})
	require.Equal(t, "header not found", err.Message)
	err = f.h.Fails("debug_traceCall", bad, "latest")
	require.Equal(t, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified", err.Message)
}

// BLOCKHASH of a traced call with a block number above the head reads no
// header: it does not walk the chain down to genesis.
func TestTraceCall_BlockHashUnderABlockOverrideReadsNoHeader(t *testing.T) {
	const blocks = 1000
	contract := common.HexToAddress("0xc0de")
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(blocks))
	// PUSH2 blocks BLOCKHASH STOP
	cs.SetCode(contract, []byte{0x61, byte(blocks >> 8), byte(blocks & 0xff), 0x40, 0x00})
	h := newHarness(t, fc)
	trace := func(config map[string]any) int {
		before := fc.Calls("Header")
		h.Result("debug_traceCall", map[string]any{"to": contract}, "latest", config)
		return fc.Calls("Header") - before
	}
	// Without an override the asked block is the block of the call: no hash.
	withoutOverride := trace(map[string]any{"tracer": "callTracer"})
	withOverride := trace(map[string]any{"tracer": "callTracer",
		"blockOverrides": map[string]any{"number": hexutil.Uint64(blocks + 1)}})
	require.Equal(t, withoutOverride, withOverride)
}

func TestBlockEncodings(t *testing.T) {
	f := newFixture(t)
	number := f.Block
	// The RLP of a block is that of the node's block type, which holds a
	// duration RLP cannot encode: the answer is the same error as ever.
	require.Contains(t, f.h.Fails("debug_getBlockRlp", number).Message, "is not RLP-serializable")
	require.Equal(t, "block #99 not found", f.h.Fails("debug_getBlockRlp", 99).Message)

	var printed string
	f.h.Into(&printed, "debug_printBlock", number)
	require.Contains(t, printed, "EvmBlock")
	require.Equal(t, "block #99 not found", f.h.Fails("debug_printBlock", 99).Message)

	// The raw encodings are the Ethereum ones.
	var raw hexutil.Bytes
	f.h.Into(&raw, "debug_getRawBlock", "latest")
	var block types.Block
	require.NoError(t, rlp.DecodeBytes(raw, &block))
	require.Len(t, block.Transactions(), 2)
	require.Equal(t, f.Transfer.Hash(), block.Transactions()[0].Hash())

	f.h.Into(&raw, "debug_getRawHeader", f.Core.BlockAt(number).Hash)
	var header types.Header
	require.NoError(t, rlp.DecodeBytes(raw, &header))
	require.EqualValues(t, number, header.Number.Uint64())

	var receipts []hexutil.Bytes
	f.h.Into(&receipts, "debug_getRawReceipts", map[string]any{"blockNumber": "0x2"})
	require.Len(t, receipts, 2)
	var receipt types.Receipt
	require.NoError(t, receipt.UnmarshalBinary(receipts[0]))
	require.EqualValues(t, types.ReceiptStatusSuccessful, receipt.Status)

	want, err3 := f.Store.MarshalBinary()
	require.NoError(t, err3)
	f.h.Into(&raw, "debug_getRawTransaction", f.Store.Hash())
	require.Equal(t, hexutil.Bytes(want), raw)
	require.JSONEq(t, `"0x"`, string(f.h.Result("debug_getRawTransaction", common.Hash{9})))

	require.Contains(t, f.h.Fails("debug_getRawBlock", "0x99").Message, "not found")
	require.Contains(t, f.h.Fails("debug_getRawHeader", common.Hash{9}).Message, "not found")
	require.Contains(t, f.h.Fails("debug_getRawReceipts", "0x99").Message, "not found")
}

func TestProfilingMethodsAreNotEnabled(t *testing.T) {
	f := newFixture(t)
	for _, call := range [][]any{
		{"debug_verbosity", 3}, {"debug_vmodule", "x"}, {"debug_memStats"}, {"debug_gcStats"},
		{"debug_cpuProfile", "f", 1}, {"debug_startCPUProfile", "f"}, {"debug_stopCPUProfile"},
		{"debug_goTrace", "f", 1}, {"debug_startGoTrace", "f"}, {"debug_stopGoTrace"},
		{"debug_blockProfile", "f", 1}, {"debug_setBlockProfileRate", 1}, {"debug_writeBlockProfile", "f"},
		{"debug_mutexProfile", "f", 1}, {"debug_setMutexProfileFraction", 1}, {"debug_writeMutexProfile", "f"},
		{"debug_writeMemProfile", "f"}, {"debug_stacks"}, {"debug_freeOSMemory"}, {"debug_setGCPercent", 1},
		{"debug_setMemoryLimit", 1},
	} {
		err := f.h.Fails(call[0].(string), call[1:]...)
		require.Equal(t, "this method is not enabled", err.Message, call[0])
	}
}

func TestIsTracerWhitelisted(t *testing.T) {
	require.True(t, isTracerWhitelisted("callTracer"))
	require.True(t, isTracerWhitelisted("4byteTracer"))
	require.False(t, isTracerWhitelisted("muxTracer"))
	require.False(t, isTracerWhitelisted(""))
}
