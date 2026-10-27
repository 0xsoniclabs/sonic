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
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func increment(chain traceChain) map[string]any {
	return map[string]any{"from": chain.sender, "to": counter, "gas": hexutil.Uint64(100_000)}
}

func TestCall_TracesACallAtAPositionOfTheBlock(t *testing.T) {
	chain := newTraceChain(t, nil)
	result := tests.Decode[map[string]any](t, chain.harness.Result("trace_call", increment(chain), []string{"trace", "stateDiff", "vmTrace"}, "latest"))
	require.Equal(t, output(1), result["output"])
	require.Len(t, result["trace"], 1)
	stateDiff := result["stateDiff"].(map[string]any)
	require.Contains(t, stateDiff, strings.ToLower(counter.Hex()))
	vmTrace := result["vmTrace"].(map[string]any)
	require.Equal(t, hexutil.Encode(counterCode), vmTrace["code"])
	require.Len(t, vmTrace["ops"], 12)

	bare := tests.Decode[map[string]any](t, chain.harness.Result("trace_call", increment(chain), []string{}, "latest"))
	require.Equal(t, map[string]any{"output": output(1), "stateDiff": nil, "trace": nil, "vmTrace": nil}, bare)

	second := tests.Decode[map[string]any](t, chain.harness.Result("trace_call", increment(chain), []string{"trace"}, "0x1", map[string]any{"txIndex": "0x1"}))
	require.Equal(t, output(2), second["output"], "the first increment of the block ran before")

	overridden := tests.Decode[map[string]any](t, chain.harness.Result("trace_call", increment(chain), []string{"trace"}, "latest",
		map[string]any{"stateOverrides": map[string]any{counter.Hex(): map[string]any{"state": map[string]any{common.Hash{}.Hex(): common.BigToHash(common.Big3).Hex()}}}}))
	require.Equal(t, output(4), overridden["output"])

	cases := map[string]struct {
		parameters []any
		message    string
	}{
		"an unknown trace type": {[]any{increment(chain), []string{"trace", "diff"}, "latest"}, "unrecognized trace type: diff"},
		"an unknown block":      {[]any{increment(chain), []string{"trace"}, map[string]any{"blockHash": common.Hash{1}}}, "header not found"},
		"no selector":           {[]any{increment(chain), []string{"trace"}, map[string]any{}}, "invalid arguments; neither block number nor hash specified"},
		"a position too far":    {[]any{increment(chain), []string{"trace"}, "0x1", map[string]any{"txIndex": "0x3"}}, "transaction index 3 out of range for block " + chain.hashes[0].Hex()},
	}
	for name, test := range cases {
		require.Equal(t, test.message, chain.harness.Failure("trace_call", test.parameters...).Message, name)
	}
}

func TestCallMany_RunsEveryCallOnTheStateThePreviousLeft(t *testing.T) {
	chain := newTraceChain(t, nil)
	overrides := map[string]any{"stateOverrides": map[string]any{counter.Hex(): map[string]any{"state": map[string]any{common.Hash{}.Hex(): common.BigToHash(common.Big3).Hex()}}}}
	results := tests.Decode[[]map[string]any](t, chain.harness.Result("trace_callMany", []any{
		[]any{increment(chain), []string{"trace"}},
		[]any{increment(chain), []string{"stateDiff"}},
	}, "0x1", overrides))
	require.Len(t, results, 2)
	require.Equal(t, output(4), results[0]["output"])
	require.Equal(t, output(5), results[1]["output"])
	require.Nil(t, results[1]["trace"])
	require.Contains(t, results[1]["stateDiff"], strings.ToLower(counter.Hex()))
	require.JSONEq(t, `[]`, string(chain.harness.Result("trace_callMany", []any{}, "latest")))

	cases := map[string]struct {
		calls   any
		message string
	}{
		"an unknown trace type": {[]any{[]any{increment(chain), []string{"trace"}}, []any{increment(chain), []string{"diff"}}}, "call 1: unrecognized trace type: diff"},
		"a call without types":  {[]any{[]any{increment(chain)}}, "cannot parse trace types: unexpected end of JSON input"},
		"a call not a pair":     {[]any{increment(chain)}, "each call must be [tx, traceTypes]"},
	}
	for name, test := range cases {
		require.Contains(t, chain.harness.Failure("trace_callMany", test.calls, "latest").Message, test.message, name)
	}
}

// TestCallMany_SharesOneBudget holds V12: the calls of trace_callMany share
// MaxResponseSize, so two calls that fit alone fail together.
func TestCallMany_SharesOneBudget(t *testing.T) {
	chain := newTraceChain(t, nil)
	alone := chain.harness.Result("trace_call", increment(chain), []string{"vmTrace"}, "latest")
	limited := newTraceChain(t, nil, func(configs *tests.Configs) {
		configs.Exec.MaxResponseSize = 2 * len(alone)
	})
	require.JSONEq(t, string(alone), string(limited.harness.Result("trace_call", increment(chain), []string{"vmTrace"}, "latest")))
	call := []any{increment(chain), []string{"vmTrace"}}
	require.Len(t, tests.Decode[[]any](t, limited.harness.Result("trace_callMany", []any{call}, "latest")), 1)
	require.Equal(t, "call 1: response too large", limited.harness.Failure("trace_callMany", []any{call, call}, "latest").Message)

	tight := newTraceChain(t, nil, func(configs *tests.Configs) {
		configs.Exec.MaxResponseSize = len(alone) - 1
	})
	require.Equal(t, "response too large", tight.harness.Failure("trace_call", increment(chain), []string{"vmTrace"}, "latest").Message)
	require.Equal(t, "response too large", tight.harness.Failure("trace_call", increment(chain), []string{"vmTrace", "trace"}, "latest").Message)

	tiny := newTraceChain(t, nil, func(configs *tests.Configs) {
		configs.Exec.MaxResponseSize = 10
	})
	require.Equal(t, "response too large", tiny.harness.Failure("trace_call", increment(chain), []string{"trace"}, "latest").Message, "a result without a vmTrace is checked once encoded")
}
