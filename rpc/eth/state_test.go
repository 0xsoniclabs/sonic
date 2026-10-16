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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newStateHarness(t *testing.T, blocks int) (*tests.Harness, *tests.FakeCore, *tests.CarmenState) {
	t.Helper()
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(blocks))
	h := tests.NewHarness(t, fc, tests.Default(), func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
		env := exec.NewEnv(chain, oracle.New(oracle.Config{}, fc))
		Register(d, env, "eth")
	})
	return h, fc, cs
}

var account = common.HexToAddress("0x1000000000000000000000000000000000000001")

func TestState_Getters(t *testing.T) {
	h, _, cs := newStateHarness(t, 2)
	slot := common.HexToHash("0x05")
	cs.SetBalance(account, big.NewInt(12345)).SetNonce(account, 7).
		SetCode(account, []byte{0x60, 0x01}).SetStorage(account, slot, common.HexToHash("0xabcd"))

	var balance hexutil.Big
	h.Into(&balance, "eth_getBalance", account, "latest")
	require.Equal(t, int64(12345), balance.ToInt().Int64())

	// The block selector is optional.
	h.Into(&balance, "eth_getBalance", account)
	require.Equal(t, int64(12345), balance.ToInt().Int64())

	var nonce hexutil.Uint64
	h.Into(&nonce, "eth_getTransactionCount", account, "0x1")
	require.EqualValues(t, 7, nonce)

	var code hexutil.Bytes
	h.Into(&code, "eth_getCode", account, "earliest")
	require.Equal(t, hexutil.Bytes{0x60, 0x01}, code)

	var value hexutil.Bytes
	h.Into(&value, "eth_getStorageAt", account, slot.Hex(), "latest")
	require.Equal(t, common.HexToHash("0xabcd").Bytes(), []byte(value))
}

func TestState_BlockSelectors(t *testing.T) {
	h, fc, _ := newStateHarness(t, 3)
	block2 := fc.BlockAt(2)

	for _, ref := range []any{
		"latest", "pending", "safe", "finalized", "earliest", "0x0", "0x3",
		block2.Hash,
		map[string]any{"blockHash": block2.Hash},
		map[string]any{"blockNumber": "0x1"},
	} {
		h.Result("eth_getBalance", account, ref)
	}

	for _, ref := range []any{"0x4", common.Hash{9}} {
		err := h.Fails("eth_getBalance", account, ref)
		require.Equal(t, dispatch.CodeDefault, err.Code, ref)
		require.Equal(t, "header not found", err.Message, ref)
	}
	// An empty reference object names no block.
	err := h.Fails("eth_getBalance", account, map[string]any{})
	require.Equal(t, "unknown header selector", err.Message)
}

func TestState_PendingNonceComesFromThePool(t *testing.T) {
	h, fc, cs := newStateHarness(t, 1)
	cs.SetNonce(account, 3)
	fc.SetPoolNonce(account, 9)

	var nonce hexutil.Uint64
	h.Into(&nonce, "eth_getTransactionCount", account, "pending")
	require.EqualValues(t, 9, nonce)
	h.Into(&nonce, "eth_getTransactionCount", account, "latest")
	require.EqualValues(t, 3, nonce)
}

func TestState_EveryStateIsReleasedExactlyOnce(t *testing.T) {
	h, fc, _ := newStateHarness(t, 1)
	tracker := fc.TrackStates()
	for _, call := range [][]any{
		{"eth_getBalance", account},
		{"eth_getCode", account},
		{"eth_getStorageAt", account, "0x1"},
		{"eth_getTransactionCount", account},
		{"eth_call", map[string]any{"to": account}},
		{"eth_estimateGas", map[string]any{"to": account, "value": "0x0"}},
	} {
		h.Result(call[0].(string), call[1:]...)
	}
	tracker.Require(t)
}

func TestState_StatesAreReleasedWhenTheHandlerPanics(t *testing.T) {
	h, fc, _ := newStateHarness(t, 1)
	tracker := fc.TrackStates()
	// A panic inside the handler, after the state is open, still releases it.
	fc.Inject("OracleInputs", tests.Panic)
	fc.Inject("PendingHeads", tests.Panic)
	_, _ = h.Call("eth_createAccessList", map[string]any{"to": account})
	tracker.Require(t)
}

func TestState_GetAccountAndProofUseTheStateRoot(t *testing.T) {
	// The fake chain has no real state roots, so the proofs cannot be checked
	// against a root; the integration tests cover the values.
	h, _, cs := newStateHarness(t, 1)
	cs.SetBalance(account, big.NewInt(5))
	require.Equal(t, dispatch.CodeInvalidParams, h.Fails("eth_getProof", account, make([]string, 1025)).Code)
	require.Equal(t, dispatch.CodeInvalidParams, h.Fails("eth_getProof", account, []string{"0xzz"}).Code)
	require.Equal(t, dispatch.CodeInvalidParams, h.Fails("eth_getProof", account, []string{"0x" + string(make([]byte, 0)) + "11223344556677889900112233445566778899001122334455667788990011223344"}).Code)
}
