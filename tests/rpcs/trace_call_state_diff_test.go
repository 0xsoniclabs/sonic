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
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	geth_crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// traceCallResult represents the JSON response of a trace_call RPC invocation.
type traceCallResult struct {
	Output    string                          `json:"output"`
	StateDiff map[common.Address]*accountDiff `json:"stateDiff"`
	Trace     []json.RawMessage               `json:"trace"`
}

// accountDiff holds all per-account state changes returned in a StateDiff response.
// Each field encodes one of four change markers:
//   - "=" (plain string) — no change
//   - {"*": ...}         — persisting account, field changed
//   - {"+": ...}         — account/slot was born (created)
//   - {"-": ...}         — account died (self-destructed)
type accountDiff struct {
	Balance json.RawMessage            `json:"balance"`
	Code    json.RawMessage            `json:"code"`
	Nonce   json.RawMessage            `json:"nonce"`
	Storage map[string]json.RawMessage `json:"storage"`
}

// diffMarker extracts the change marker from a stateDiff field value.
// Returns "=" when the field is the plain unchanged string, or the single
// JSON object key ("*", "+", or "-") for all other cases.
func diffMarker(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &m), "unexpected stateDiff field format")
	for k := range m {
		return k
	}
	return ""
}

// diffValue extracts the value from a stateDiff field.
// Returns nil when the field is the plain unchanged string.
func diffValue(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return nil
	}
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), "unexpected stateDiff field format")
	for _, v := range m {
		return v
	}
	return nil
}

// senderOverride returns a trace_call config that pre-sets addr's balance to
// senderBalance so that state-changing hooks fire correctly regardless of the
// block state that trace_call uses as its starting point.
//
// trace_call evaluates the call against the state at the beginning of the
// "latest" block (parent-block state). Because the integration test may call
// trace_call immediately after the block that first endowed the sender, the
// sender's balance can be zero in that parent-block state. A state override
// ensures the simulation always has enough funds and that the balance-change
// hook provides a non-zero "before" value, allowing StateDiffLogger to
// correctly classify the sender as a pre-existing (persisting) account.
func senderOverride(addr common.Address, senderBalance *big.Int) map[string]any {
	return map[string]any{
		// JSON key matches TraceCallConfig.StateOverrides field name (Go's default encoding).
		"StateOverrides": map[string]any{
			addr.Hex(): map[string]any{
				"balance": hexutil.EncodeBig(senderBalance),
			},
		},
	}
}

// waitForBlockAfter polls eth_blockNumber until the network produces a block
// strictly after afterBlock. This guarantees that the deployment or state
// change that happened in afterBlock is part of the parent-block state used
// by a subsequent trace_call "latest" call.
func waitForBlockAfter(t *testing.T, client *tests.PooledEhtClient, afterBlock *big.Int) {
	t.Helper()
	err := tests.WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		var current hexutil.Uint64
		if err := client.Client().Call(&current, "eth_blockNumber"); err != nil {
			return false, err
		}
		return new(big.Int).SetUint64(uint64(current)).Cmp(afterBlock) > 0, nil
	})
	require.NoError(t, err, "timed out waiting for a new block after block %s", afterBlock)
}

// TestStateDiff_TraceCall_Transfer verifies that a simulated native token transfer
// via trace_call produces correct stateDiff entries:
//   - the sender appears as a persisting account whose balance and nonce changed
//   - the recipient appears as a newly born account with an increased balance
//
// A state override is used to pre-fund the sender so that both the transfer and
// the balance-change hook succeed regardless of the current block-state timing.
func TestStateDiff_TraceCall_Transfer(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()
	recipient := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	txArgs := map[string]any{
		"from":  sender.Address(),
		"to":    recipient.Address(),
		"value": (*hexutil.Big)(big.NewInt(1e17)), // 0.1 S
	}

	var result traceCallResult
	err = client.Client().Call(
		&result, "trace_call",
		txArgs, []string{"stateDiff"}, "latest",
		senderOverride(sender.Address(), big.NewInt(2e18)),
	)
	require.NoError(t, err)
	require.NotNil(t, result.StateDiff, "stateDiff must not be nil")

	// Sender is pre-existing (balance override ensures it); its balance decreases
	// by the transferred value and its nonce is incremented.
	senderDiff, ok := result.StateDiff[sender.Address()]
	require.True(t, ok, "sender must appear in stateDiff")
	require.Equal(t, "*", diffMarker(t, senderDiff.Balance), "sender balance should be marked as changed (*)")
	require.Equal(t, "*", diffMarker(t, senderDiff.Nonce), "sender nonce should be marked as changed (*)")

	// Recipient had no prior balance/nonce/code — it is born in this transaction.
	recipientDiff, ok := result.StateDiff[recipient.Address()]
	require.True(t, ok, "recipient must appear in stateDiff")
	require.Equal(t, "+", diffMarker(t, recipientDiff.Balance), "recipient balance should be marked as born (+)")

	// Verify the recipient's new balance is 0.1 S.
	recipientBalance := diffValue(t, recipientDiff.Balance)
	if balance, ok := recipientBalance.(string); ok {
		newBalance, err := hexutil.DecodeBig(balance)
		require.NoError(t, err, "failed to decode recipient balance")
		require.Equal(t, big.NewInt(1e17), newBalance, "recipient balance should be 0.1 S")
	} else {
		t.Fatalf("unexpected recipient balance format: %T", recipientBalance)
	}
}

// TestStateDiff_TraceCall_ContractStorageModification verifies that a simulated
// call to a state-modifying contract function produces correct stateDiff entries:
//   - the counter contract appears with at least one storage slot marked as changed ("*")
func TestStateDiff_TraceCall_ContractStorageModification(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	_, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	counterAddr := receipt.ContractAddress

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// trace_call evaluates against the parent-block state of "latest". If the
	// deployment was included in the current latest block, the counter contract
	// would not yet exist in that parent state. Wait until a new block is produced
	// so the deployment is visible in the next call's parent-block state.
	waitForBlockAfter(t, client, receipt.BlockNumber)

	sender := tests.NewAccount()

	// Function selector for incrementCounter() → keccak256("incrementCounter()")[0:4] = 0x5b34b966
	txArgs := map[string]any{
		"from": sender.Address(),
		"to":   counterAddr,
		"data": hexutil.Bytes([]byte{0x5b, 0x34, 0xb9, 0x66}),
	}

	var result traceCallResult
	err = client.Client().Call(&result, "trace_call", txArgs, []string{"stateDiff"}, "latest")
	require.NoError(t, err)
	require.NotNil(t, result.StateDiff, "stateDiff must not be nil")

	// The counter contract modifies storage slot 0 (count: 0 → 1).
	// The contract already existed before this call, so each changed slot is
	// marked with the persisting-change operator "*".
	contractDiff, ok := result.StateDiff[counterAddr]
	require.True(t, ok, "counter contract must appear in stateDiff")
	require.NotEmpty(t, contractDiff.Storage, "counter contract must have at least one changed storage slot")
	for slot, slotDiff := range contractDiff.Storage {
		require.Equal(t, "*", diffMarker(t, slotDiff),
			"storage slot %s should be marked as changed (*)", slot)
	}
}

// TestStateDiff_TraceCall_ContractDeployment verifies that a simulated contract
// deployment via trace_call produces correct stateDiff entries:
//   - the newly deployed contract address appears as born ("+") with non-trivial
//     code and an incremented nonce
//
// gasPrice is not set (defaults to zero) so that the simulation succeeds even
// when the sender has no pre-existing on-chain balance.
func TestStateDiff_TraceCall_ContractDeployment(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// The deployed contract address is computed from the sender address and its
	// nonce at the time evm.Create runs. state_transition does NOT pre-increment
	// the sender nonce for creation transactions; evm.Create reads the nonce from
	// state before incrementing it internally. For a fresh (never-used) sender
	// with nonce=0, the address is crypto.CreateAddress(sender, 0).
	deployedAddr := geth_crypto.CreateAddress(sender.Address(), 0)

	counterBin, err := hexutil.Decode(counter.CounterMetaData.Bin)
	require.NoError(t, err)

	txArgs := map[string]any{
		"from": sender.Address(),
		"gas":  hexutil.Uint64(500_000),
		"data": hexutil.Bytes(counterBin),
	}

	var result traceCallResult
	err = client.Client().Call(&result, "trace_call", txArgs, []string{"stateDiff"}, "latest")
	require.NoError(t, err)
	require.NotNil(t, result.StateDiff, "stateDiff must not be nil")

	// The deployed contract is brand new: it must appear with the "+" (born) marker
	// for its nonce (set to 1 by EIP-161) and code (the deployed runtime bytecode).
	deployedDiff, ok := result.StateDiff[deployedAddr]
	require.True(t, ok, "deployed contract must appear in stateDiff at %s", deployedAddr)
	require.Equal(t, "+", diffMarker(t, deployedDiff.Nonce), "deployed contract nonce should be marked as born (+)")
	require.Equal(t, "+", diffMarker(t, deployedDiff.Code), "deployed contract code should be marked as born (+)")
}
