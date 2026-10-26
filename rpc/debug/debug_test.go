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

package debug_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newDebug(t *testing.T, mutate ...tests.Mutate) (*tests.ReplayChain, *tests.Harness) {
	t.Helper()
	r := tests.NewReplayChain(t)
	return r, tests.NewHarness(t, r.Core, func(h *tests.Harness) { debug.Register(h.D, h.Env) }, mutate...)
}

func TestDebug_TraceTransactionWithTheStructLogger(t *testing.T) {
	r, h := newDebug(t)
	var res struct {
		Gas         uint64            `json:"gas"`
		Failed      bool              `json:"failed"`
		StructLogs  []json.RawMessage `json:"structLogs"`
		ReturnValue string            `json:"returnValue"`
	}
	h.MustCall(&res, "debug_traceTransaction", r.Call.Hash())
	require.False(t, res.Failed)
	require.Len(t, res.StructLogs, len(tests.StorageWritingCode)-2, "one entry per operation")

	h.MustCall(&res, "debug_traceTransaction", r.Transfer.Hash())
	require.Empty(t, res.StructLogs)
	require.Equal(t, uint64(21000), res.Gas)
}

func TestDebug_TraceWithANamedTracer(t *testing.T) {
	r, h := newDebug(t)
	var call struct {
		Type string         `json:"type"`
		From common.Address `json:"from"`
		To   common.Address `json:"to"`
	}
	h.MustCall(&call, "debug_traceTransaction", r.Call.Hash(), map[string]any{"tracer": "callTracer"})
	require.Equal(t, "CALL", call.Type)
	require.Equal(t, r.Wallet.Address, call.From)
	require.Equal(t, r.Writer, call.To)

	var block []struct {
		TxHash common.Hash     `json:"txHash"`
		Result json.RawMessage `json:"result"`
	}
	h.MustCall(&block, "debug_traceBlockByNumber", "0x1", map[string]any{"tracer": "callTracer"})
	require.Len(t, block, 2)
	require.Equal(t, r.Call.Hash(), block[1].TxHash)
	h.MustCall(&block, "debug_traceBlockByHash", r.Block.Hash, map[string]any{"tracer": "prestateTracer"})
	require.Len(t, block, 2)

	h.MustCall(&call, "debug_traceCall", map[string]any{"from": r.Wallet.Address, "to": r.Returner}, "latest", map[string]any{"tracer": "callTracer"})
	require.Equal(t, r.Returner, call.To)
}

// TestDebug_TracerChecks is V6: the tracer is checked after its subject is
// found, and a timeout above the EVM timeout is refused.
func TestDebug_TracerChecks(t *testing.T) {
	r, h := newDebug(t)
	e := h.Fails("debug_traceTransaction", r.Call.Hash(), map[string]any{"tracer": "function() {}"})
	require.Equal(t, "custom tracer is not permitted", e.Message)
	e = h.Fails("debug_traceTransaction", common.Hash{1}, map[string]any{"tracer": "function() {}"})
	require.Equal(t, "transaction 0x0100000000000000000000000000000000000000000000000000000000000000 not found", e.Message, "the subject is found first")

	e = h.Fails("debug_traceTransaction", r.Call.Hash(), map[string]any{"timeout": "1h"})
	require.Equal(t, -32602, e.Code)
	require.Equal(t, "tracer timeout 1h0m0s is above the limit 5s", e.Message)
	e = h.Fails("debug_traceTransaction", r.Call.Hash(), map[string]any{"timeout": "soon"})
	require.Equal(t, -32602, e.Code)
	require.True(t, strings.HasPrefix(e.Message, `invalid tracer timeout "soon"`))
}

// TestDebug_MissingBlocks holds the wordings of F3.
func TestDebug_MissingBlocks(t *testing.T) {
	r, h := newDebug(t)
	require.Equal(t, "block #9 not found", h.Fails("debug_traceBlockByNumber", "0x9").Message)
	require.Equal(t, "block 0x0100000000000000000000000000000000000000000000000000000000000000 not found", h.Fails("debug_traceBlockByHash", common.Hash{1}).Message)
	require.Equal(t, "genesis is not traceable", h.Fails("debug_traceBlockByNumber", "0x0").Message)
	require.Equal(t, "block #9 not found", h.Fails("debug_getBlockRlp", 9).Message)
	require.Equal(t, "block #9 not found", h.Fails("debug_printBlock", 9).Message)
	require.Equal(t, "tracing on top of pending is not supported", h.Fails("debug_traceCall", map[string]any{"to": r.Returner}, "pending").Message)
	require.Equal(t, "header not found", h.Fails("debug_traceCall", map[string]any{"to": r.Returner}, "0x9").Message)
}

func TestDebug_PrintBlock(t *testing.T) {
	r, h := newDebug(t)
	var printed string
	h.MustCall(&printed, "debug_printBlock", 1)
	require.Contains(t, printed, "EvmBlock")
	require.Contains(t, printed, r.Block.Hash.Hex()[2:10])
	// The reference encodes the block with a field RLP does not take.
	require.Contains(t, h.Fails("debug_getBlockRlp", 1).Message, "rlp: type time.Duration is not RLP-serializable")
}

// TestDebug_ABlockTraceIsBoundedByTheResponseSize is V12 for debug_*.
func TestDebug_ABlockTraceIsBoundedByTheResponseSize(t *testing.T) {
	r, h := newDebug(t, func(c *tests.Configs) { c.Exec.MaxResponseSize = 100 })
	e := h.Fails("debug_traceBlockByNumber", "0x1")
	require.Equal(t, "response size exceeded", e.Message)
	_ = r
}

func TestDebug_ProfilingIsNotEnabled(t *testing.T) {
	_, h := newDebug(t)
	shadows := map[string][]any{
		"debug_verbosity":               {3},
		"debug_vmodule":                 {"p2p=5"},
		"debug_memStats":                {},
		"debug_gcStats":                 {},
		"debug_cpuProfile":              {"file", 1},
		"debug_startCPUProfile":         {"file"},
		"debug_stopCPUProfile":          {},
		"debug_goTrace":                 {"file", 1},
		"debug_startGoTrace":            {"file"},
		"debug_stopGoTrace":             {},
		"debug_blockProfile":            {"file", 1},
		"debug_setBlockProfileRate":     {1},
		"debug_writeBlockProfile":       {"file"},
		"debug_mutexProfile":            {"file", 1},
		"debug_setMutexProfileFraction": {1},
		"debug_writeMutexProfile":       {"file"},
		"debug_writeMemProfile":         {"file"},
		"debug_stacks":                  {},
		"debug_freeOSMemory":            {},
		"debug_setGCPercent":            {1},
		"debug_setMemoryLimit":          {1},
	}
	for method, params := range shadows {
		require.Equal(t, "this method is not enabled", h.Fails(method, params...).Message, method)
	}
	// The arguments are decoded first, as by go-ethereum's server.
	require.Equal(t, "parameter 0 (file): missing value", h.Fails("debug_cpuProfile").Message)
}
