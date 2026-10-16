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

package sonic

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var bob = common.HexToAddress("0xb0b")

type fixture struct {
	h     *tests.Harness
	fc    *tests.FakeCore
	alice tests.Account
}

func newFixture(t *testing.T, blocks int) *fixture {
	t.Helper()
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(blocks))
	alice := tests.NewAccount(t, cs, big.NewInt(1e18))
	h := tests.NewHarness(t, fc, tests.Default(), func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
		Register(d, exec.NewEnv(chain, oracle.New(oracle.Config{}, fc)))
	})
	return &fixture{h: h, fc: fc, alice: alice}
}

func TestEstimateGasForTransactions(t *testing.T) {
	f := newFixture(t, 2)
	var res struct {
		GasLimits []hexutil.Uint64 `json:"gasLimits"`
	}
	f.h.Into(&res, "sonic_estimateGasForTransactions", []any{
		map[string]any{"from": f.alice.Addr, "to": bob, "value": "0x1"},
		map[string]any{"from": f.alice.Addr, "to": bob, "value": "0x2"},
	})
	require.Len(t, res.GasLimits, 2)
	// Each estimate is padded for the bundle-only address and the plan hash.
	for _, gas := range res.GasLimits {
		require.EqualValues(t, 21000+2400+1900, gas)
	}

	f.h.Into(&res, "sonic_estimateGasForTransactions", []any{}, "latest")
	require.Empty(t, res.GasLimits)

	err := f.h.Fails("sonic_estimateGasForTransactions", make([]map[string]any, MaxNumEstimableTransactions+1))
	require.Contains(t, err.Message, "too many transactions to estimate gas for")

	// A failing transaction names its position.
	err = f.h.Fails("sonic_estimateGasForTransactions", []any{map[string]any{"from": f.alice.Addr, "to": bob},
		map[string]any{"from": f.alice.Addr, "to": bob, "value": "0xffffffffffffffffffffffffff"}})
	require.Contains(t, err.Message, "failed to estimate gas for transaction 1")
}

func proposal(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{
		"steps": []any{
			map[string]any{"from": from, "to": to, "value": "0x1", "nonce": hexutil.Uint64(nonce), "chainId": "0xfa3"},
			map[string]any{"from": from, "to": to, "value": "0x2", "nonce": hexutil.Uint64(nonce + 1), "chainId": "0xfa3"},
		},
	}
}

func TestPrepareAndSubmitBundle(t *testing.T) {
	f := newFixture(t, 3)
	var prepared struct {
		Transactions  []map[string]any `json:"transactions"`
		ExecutionPlan json.RawMessage  `json:"executionPlan"`
	}
	f.h.Into(&prepared, "sonic_prepareBundle", proposal(f.alice.Addr, bob, 0))
	require.Len(t, prepared.Transactions, 2)
	for _, tx := range prepared.Transactions {
		// Every transaction is bound to the plan: its access list names the
		// bundle-only address.
		require.NotEmpty(t, tx["accessList"])
		require.NotNil(t, tx["gas"])
	}

	// The block range of the plan starts after the head.
	var plan struct {
		BlockRange struct {
			First hexutil.Uint64 `json:"first"`
		} `json:"blockRange"`
	}
	require.NoError(t, json.Unmarshal(prepared.ExecutionPlan, &plan))
	require.EqualValues(t, 4, plan.BlockRange.First)

	// Sign the prepared transactions and submit them with the plan.
	var signed []hexutil.Bytes
	for i, args := range prepared.Transactions {
		raw, err := json.Marshal(args)
		require.NoError(t, err)
		var a exec.CallArgs
		require.NoError(t, json.Unmarshal(raw, &a))
		a.Nonce = (*hexutil.Uint64)(ptr(uint64(i)))
		tx, err := a.ToTransaction()
		require.NoError(t, err)
		sig, err := types.SignTx(tx, tests.Signer(), f.alice.Key)
		require.NoError(t, err)
		bin, err := sig.MarshalBinary()
		require.NoError(t, err)
		signed = append(signed, bin)
	}
	var planHash common.Hash
	f.h.Into(&planHash, "sonic_submitBundle", map[string]any{"signedTransactions": signed, "executionPlan": prepared.ExecutionPlan})
	require.NotZero(t, planHash)
	// The bundle went to the pool as one envelope transaction.
	pending, _ := f.fc.PoolStats()
	require.Equal(t, 1, pending)

	err := f.h.Fails("sonic_submitBundle", map[string]any{"signedTransactions": []hexutil.Bytes{}, "executionPlan": prepared.ExecutionPlan})
	require.Equal(t, "signedTransactions must not be empty", err.Message)
	err = f.h.Fails("sonic_submitBundle", map[string]any{"signedTransactions": signed[:1], "executionPlan": prepared.ExecutionPlan})
	require.Contains(t, err.Message, "must match signedTransactions count")
}

func ptr[T any](v T) *T { return &v }

func TestPrepareBundle_Errors(t *testing.T) {
	f := newFixture(t, 5)
	// A block range in the past.
	p := proposal(f.alice.Addr, bob, 0)
	p["blockRange"] = map[string]any{"first": "0x1", "length": "0x2"}
	err := f.h.Fails("sonic_prepareBundle", p)
	require.Contains(t, err.Message, "is not in the future relative to current block")

	// A step without nonce or sender.
	err = f.h.Fails("sonic_prepareBundle", map[string]any{"steps": []any{map[string]any{"from": f.alice.Addr, "to": bob}}})
	require.Contains(t, err.Message, "missing nonce")
	err = f.h.Fails("sonic_prepareBundle", map[string]any{"steps": []any{map[string]any{"to": bob, "nonce": "0x0"}}})
	require.Contains(t, err.Message, "missing from")

	// A block range that is too long.
	p = proposal(f.alice.Addr, bob, 0)
	p["blockRange"] = map[string]any{"first": "0x5", "length": "0xffffff"}
	err = f.h.Fails("sonic_prepareBundle", p)
	require.Contains(t, err.Message, "is too large")
}

func TestGetBundleInfo(t *testing.T) {
	f := newFixture(t, 3)
	hash := common.Hash{7}
	require.JSONEq(t, `null`, string(f.h.Result("sonic_getBundleInfo", hash)))

	f.fc.SetBundleInfo(hash, core.BundleInfo{BlockNumber: 2, Offset: 3, Count: 4})
	var info map[string]string
	f.h.Into(&info, "sonic_getBundleInfo", hash)
	require.Equal(t, map[string]string{"block": "0x2", "position": "0x3", "count": "0x4"}, info)

	// The block of a bundle that is not served yet is not reported.
	late := common.Hash{8}
	f.fc.SetBundleInfo(late, core.BundleInfo{BlockNumber: 50})
	require.JSONEq(t, `null`, string(f.h.Result("sonic_getBundleInfo", late)))
}
