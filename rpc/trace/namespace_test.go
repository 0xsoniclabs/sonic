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

package trace_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/trace"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var (
	writer  = common.Address{0xc0}
	spinner = common.Address{0xc1}
)

// traceChain serves block 1 with a call of a contract that writes storage
// and block 2 without transactions; block 2 is the head.
type traceChain struct {
	harness *tests.Harness
	account *tests.Account
	call    *types.Transaction
}

func newTraceChain(t *testing.T, mutate ...func(*tests.Settings)) *traceChain {
	t.Helper()
	account := tests.NewAccount()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		account.Address(): {Balance: big.NewInt(1e18)},
		writer:            {Code: tests.WritesStorage},
		spinner:           {Code: tests.Spins},
	})
	call := account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), To: &writer, Gas: 100000,
		GasFeeCap: big.NewInt(2e9), GasTipCap: big.NewInt(1)})
	fake.AddBlock(call)
	fake.AddBlock()
	fake.SetHead(2)
	h := tests.NewHarness(t, fake, func(h *tests.Harness) { trace.Register(h.Dispatcher, h.Environment) }, mutate...)
	return &traceChain{harness: h, account: account, call: call}
}

type actionTrace struct {
	Action struct {
		From *common.Address `json:"from"`
		To   *common.Address `json:"to"`
	} `json:"action"`
	BlockNumber     uint64      `json:"blockNumber"`
	TransactionHash common.Hash `json:"transactionHash"`
	Type            string      `json:"type"`
	Error           string      `json:"error"`
}

func TestTraceTransaction_AnswersTheTracesOfATransaction(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	var traces []actionTrace
	h.Call(t, &traces, "trace_transaction", chain.call.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, "call", traces[0].Type)
	require.Equal(t, chain.account.Address(), *traces[0].Action.From)
	require.Equal(t, writer, *traces[0].Action.To)

	var at []actionTrace
	h.Call(t, &at, "trace_get", chain.call.Hash(), []hexutil.Uint{})
	require.Equal(t, traces, at, "the root trace")
	require.Equal(t, "null", h.Result(t, "trace_get", chain.call.Hash(), []hexutil.Uint{5}))
	unknown := common.Hash{0x01}
	require.Equal(t, "transaction "+unknown.String()+" not found", h.Fail(t, "trace_transaction", unknown).Message)
	require.Zero(t, h.Core.OpenStates())
}

func TestTraceBlock_AnswersEveryTransactionOrTheEmptyBlock(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	var traces []actionTrace
	h.Call(t, &traces, "trace_block", "0x1")
	require.Len(t, traces, 1)
	h.Call(t, &traces, "trace_block", "latest")
	require.Len(t, traces, 1)
	require.Equal(t, "empty", traces[0].Type)
	require.Equal(t, "Empty block", traces[0].Error)
	require.Equal(t, "cannot trace block 0 got genesis block is not traceable", h.Fail(t, "trace_block", "0x0").Message)
	require.Equal(t, "cannot trace block 5 got invalid block for tracing", h.Fail(t, "trace_block", "0x5").Message)
	require.Zero(t, h.Core.OpenStates())
}

func TestTraceFilter_SelectsTracesByAddressAndWindow(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	cases := map[string]struct {
		filter map[string]any
		count  int
	}{
		"all in parallel":   {map[string]any{"fromBlock": "0x1"}, 1},
		"by sender":         {map[string]any{"fromBlock": "0x1", "fromAddress": []common.Address{chain.account.Address()}}, 1},
		"by another target": {map[string]any{"fromBlock": "0x1", "toAddress": []common.Address{spinner}}, 0},
		"in order":          {map[string]any{"fromBlock": "0x1", "count": 1}, 1},
		"after the first":   {map[string]any{"fromBlock": "0x1", "after": 1}, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var traces []actionTrace
			h.Call(t, &traces, "trace_filter", test.filter)
			require.Len(t, traces, test.count)
		})
	}
	limited := newTraceChain(t, func(settings *tests.Settings) { settings.Exec.MaxTraceFilterRange = 1 })
	require.Equal(t, "too wide blocks range, the limit is 1", limited.harness.Fail(t, "trace_filter", map[string]any{"fromBlock": "0x0"}).Message)
	require.Equal(t, "block not found", h.Fail(t, "trace_filter", map[string]any{"fromBlock": map[string]any{"blockHash": common.Hash{0x01}}}).Message)
}

type callResult struct {
	Output    hexutil.Bytes             `json:"output"`
	StateDiff map[string]map[string]any `json:"stateDiff"`
	Trace     []actionTrace             `json:"trace"`
	VmTrace   *struct {
		Ops []map[string]any `json:"ops"`
	} `json:"vmTrace"`
}

func TestTraceCall_AnswersTheKindsOfTracesAskedFor(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	call := map[string]any{"from": chain.account.Address(), "to": writer}
	var result callResult
	h.Call(t, &result, "trace_call", call, []string{"trace", "stateDiff", "vmTrace"}, "0x1")
	require.Len(t, result.Trace, 1)
	require.Contains(t, result.StateDiff, "0xc000000000000000000000000000000000000000")
	require.Len(t, result.VmTrace.Ops, 4)
	h.Call(t, &result, "trace_call", call, []string{}, "0x1")
	require.Nil(t, result.VmTrace)
	require.Equal(t, "unrecognized trace type: other", h.Fail(t, "trace_call", call, []string{"other"}, "0x1").Message)
	require.Equal(t, "header not found", h.Fail(t, "trace_call", call, []string{"trace"}, "0x5").Message)
	require.Zero(t, h.Core.OpenStates())
}

// Each call of a trace_callMany runs on the state the ones before it left.
func TestTraceCallMany_RunsTheCallsOneAfterTheOther(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	call := []any{map[string]any{"from": chain.account.Address(), "to": writer}, []string{"stateDiff"}}
	var results []callResult
	h.Call(t, &results, "trace_callMany", []any{call, call}, "0x1")
	require.Len(t, results, 2)
	require.Contains(t, results[0].StateDiff["0xc000000000000000000000000000000000000000"], "storage")
	require.NotContains(t, results[1].StateDiff, "0xc000000000000000000000000000000000000000", "the slot was written before")
	require.Equal(t, "call 1: unrecognized trace type: other",
		h.Fail(t, "trace_callMany", []any{call, []any{call[0], []string{"other"}}}, "0x1").Message)
	require.Contains(t, h.Fail(t, "trace_callMany", []any{[]any{1}}, "0x1").Message, "cannot parse transaction args")
}

// The answer of a traced call is bounded as a whole: a vmTrace stops at what
// is left of the bound, and calls that fit alone fail together.
func TestTraceCalls_AreBoundedAsAWhole(t *testing.T) {
	chain := newTraceChain(t, func(settings *tests.Settings) {
		settings.Exec.MaxResponseSize = 1000
		settings.Exec.RPCGasCap = 1 << 40
	})
	h := chain.harness
	start := time.Now()
	spin := map[string]any{"from": chain.account.Address(), "to": spinner, "gas": hexutil.Uint64(1 << 30)}
	require.Equal(t, "vmTrace exceeds size limit", h.Fail(t, "trace_call", spin, []string{"vmTrace"}, "0x1").Message)
	require.Less(t, time.Since(start), 5*time.Second, "the vmTrace stops before the run ends")

	cases := map[string]struct {
		kinds []string
		err   string
	}{
		"traces":   {[]string{"trace", "stateDiff"}, "response too large"},
		"vmTraces": {[]string{"vmTrace"}, "call 1: vmTrace exceeds size limit"},
	}
	for name, test := range cases {
		write := []any{map[string]any{"from": chain.account.Address(), "to": writer}, test.kinds}
		var alone []json.RawMessage
		h.Call(t, &alone, "trace_callMany", []any{write}, "0x1")
		require.Less(t, len(alone[0]), 1000, "one call fits")
		require.Greater(t, 2*len(alone[0]), 1000, "two do not")
		require.Equal(t, test.err, h.Fail(t, "trace_callMany", []any{write, write}, "0x1").Message, name)
	}
}
