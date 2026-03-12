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

package rpcs

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// callManyConfig returns a trace_callMany config that pre-sets each address in
// balances to the corresponding amount, so simulations succeed regardless of
// on-chain state. This mirrors the senderOverride helper used in trace_call
// tests, but for the TraceCallConfig accepted by trace_callMany.
func callManyConfig(balances map[common.Address]*big.Int) map[string]any {
	stateOverrides := make(map[string]any, len(balances))
	for addr, balance := range balances {
		stateOverrides[addr.Hex()] = map[string]any{
			"balance": hexutil.EncodeBig(balance),
		}
	}
	return map[string]any{
		"StateOverrides": stateOverrides,
	}
}

// TestCallMany_ReturnsOneResultPerCall verifies that trace_callMany returns
// exactly one result per submitted call.
func TestCallMany_ReturnsOneResultPerCall(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()
	recipient1 := tests.NewAccount()
	recipient2 := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	calls := []any{
		[]any{
			map[string]any{"from": sender.Address(), "to": recipient1.Address(), "value": (*hexutil.Big)(big.NewInt(1e17))},
			[]string{"trace"},
		},
		[]any{
			map[string]any{"from": sender.Address(), "to": recipient2.Address(), "value": (*hexutil.Big)(big.NewInt(1e17))},
			[]string{"trace"},
		},
	}
	config := callManyConfig(map[common.Address]*big.Int{
		sender.Address(): big.NewInt(2e18),
	})

	var results []traceCallResult
	err = client.Client().Call(&results, "trace_callMany", calls, "latest", config)
	require.NoError(t, err)
	require.Len(t, results, 2, "must return one result per call")

	for i, result := range results {
		require.NotEmpty(t, result.Trace, "call %d must have a trace", i)
		require.Nil(t, result.StateDiff, "stateDiff must be nil when not requested (call %d)", i)
	}
}

// TestCallMany_StateAccumulatesAcrossCalls verifies the core trace_callMany
// invariant: each call is executed on top of the state produced by all previous
// calls, so call N sees the state changes made by calls 0..N-1.
//
// The test proceeds in two simulated steps:
//  1. Transfer 0.1 S from sender to recipient — recipient is born (+).
//  2. Transfer 0.05 S from recipient to another — this only succeeds if the
//     recipient's balance from step 1 was preserved in the accumulated state.
//     recipient is persisting (*) and another is born (+).
//
// State overrides pre-fund the sender so no on-chain balance is required.
func TestCallMany_StateAccumulatesAcrossCalls(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()
	recipient := tests.NewAccount()
	another := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	calls := []any{
		// Call 0: sender → recipient, 0.1 S
		[]any{
			map[string]any{
				"from":  sender.Address(),
				"to":    recipient.Address(),
				"value": (*hexutil.Big)(big.NewInt(1e17)),
			},
			[]string{"stateDiff"},
		},
		// Call 1: recipient → another, 0.05 S
		// Only succeeds if state from call 0 is visible (recipient has balance).
		[]any{
			map[string]any{
				"from":  recipient.Address(),
				"to":    another.Address(),
				"value": (*hexutil.Big)(big.NewInt(5e16)),
			},
			[]string{"stateDiff"},
		},
	}
	config := callManyConfig(map[common.Address]*big.Int{
		sender.Address(): big.NewInt(2e18),
	})

	var results []traceCallResult
	err = client.Client().Call(&results, "trace_callMany", calls, "latest", config)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// ── Call 0 stateDiff ──────────────────────────────────────────────────────
	diff0 := results[0].StateDiff
	require.NotNil(t, diff0, "call 0 must have a stateDiff")

	// Sender was given balance via state override — its balance and nonce change (*).
	senderDiff0, ok := diff0[sender.Address()]
	require.True(t, ok, "sender must appear in call 0 stateDiff")
	require.Equal(t, "*", diffMarker(t, senderDiff0.Balance), "sender balance must be marked as changed (*)")
	require.Equal(t, "*", diffMarker(t, senderDiff0.Nonce), "sender nonce must be marked as changed (*)")

	// Recipient is brand-new in the simulation — it is born (+).
	recipientDiff0, ok := diff0[recipient.Address()]
	require.True(t, ok, "recipient must appear in call 0 stateDiff")
	require.Equal(t, "+", diffMarker(t, recipientDiff0.Balance), "recipient balance must be born (+) in call 0")

	// ── Call 1 stateDiff ──────────────────────────────────────────────────────
	diff1 := results[1].StateDiff
	require.NotNil(t, diff1, "call 1 must have a stateDiff")

	// Recipient was created in call 0 — from call 1's perspective it is an
	// existing account whose balance decreases (*).
	recipientDiff1, ok := diff1[recipient.Address()]
	require.True(t, ok, "recipient must appear in call 1 stateDiff")
	require.Equal(t, "*", diffMarker(t, recipientDiff1.Balance),
		"recipient must be a persisting account (*) in call 1 — proves state accumulation")

	// Another is brand-new — born (+).
	anotherDiff1, ok := diff1[another.Address()]
	require.True(t, ok, "another must appear in call 1 stateDiff")
	require.Equal(t, "+", diffMarker(t, anotherDiff1.Balance), "another balance must be born (+) in call 1")
}

// TestCallMany_IndependentTraceTypesPerCall verifies that each call in a
// trace_callMany request can independently request different trace types, and
// that only the requested data is populated in each result.
func TestCallMany_IndependentTraceTypesPerCall(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()
	recipient := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	txArgs := map[string]any{
		"from":  sender.Address(),
		"to":    recipient.Address(),
		"value": (*hexutil.Big)(big.NewInt(1e16)),
	}
	config := callManyConfig(map[common.Address]*big.Int{
		sender.Address(): big.NewInt(2e18),
	})

	calls := []any{
		[]any{txArgs, []string{"trace"}},
		[]any{txArgs, []string{"stateDiff"}},
		[]any{txArgs, []string{"trace", "stateDiff"}},
	}

	var results []traceCallResult
	err = client.Client().Call(&results, "trace_callMany", calls, "latest", config)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Call 0: trace only
	require.NotEmpty(t, results[0].Trace, "call 0 must have trace actions")
	require.Nil(t, results[0].StateDiff, "call 0 must not have stateDiff")

	// Call 1: stateDiff only
	require.Empty(t, results[1].Trace, "call 1 must not have trace actions")
	require.NotNil(t, results[1].StateDiff, "call 1 must have stateDiff")

	// Call 2: both
	require.NotEmpty(t, results[2].Trace, "call 2 must have trace actions")
	require.NotNil(t, results[2].StateDiff, "call 2 must have stateDiff")
}
