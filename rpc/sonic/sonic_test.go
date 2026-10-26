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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newSonic(t *testing.T) (*tests.ReplayChain, *tests.Harness) {
	t.Helper()
	r := tests.NewReplayChain(t)
	return r, tests.NewHarness(t, r.Core, func(h *tests.Harness) { Register(h.D, h.Env) })
}

func TestEstimateGasForTransactions_EachOnTheStateOfThoseBefore(t *testing.T) {
	r, h := newSonic(t)
	var res BundleGasLimits
	transfer := map[string]any{"from": r.Wallet.Address, "to": r.Recipient, "value": "0x1"}
	h.MustCall(&res, "sonic_estimateGasForTransactions", []any{transfer, transfer})
	marker := hexutil.Uint64(params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas)
	require.Equal(t, []hexutil.Uint64{21000 + marker, 21000 + marker}, res.GasLimits)

	poor := map[string]any{"from": common.HexToAddress("0x7"), "to": r.Recipient, "value": "0x1", "gasPrice": "0x3b9aca00"}
	e := h.Fails("sonic_estimateGasForTransactions", []any{transfer, poor})
	require.Equal(t, "failed to estimate gas for transaction 1: insufficient funds for transfer", e.Message)

	many := make([]any, MaxNumEstimableTransactions+1)
	for i := range many {
		many[i] = transfer
	}
	e = h.Fails("sonic_estimateGasForTransactions", many)
	require.Contains(t, e.Message, "too many transactions to estimate gas for")
}

func TestPrepareAndSubmitBundle(t *testing.T) {
	r, h := newSonic(t)
	var prepared RPCPreparedBundle
	h.MustCall(&prepared, "sonic_prepareBundle", map[string]any{
		"steps": []any{
			map[string]any{"from": r.Wallet.Address, "to": r.Recipient, "value": "0x1", "nonce": "0x2"},
			map[string]any{"from": r.Wallet.Address, "to": r.Writer, "nonce": "0x3"},
		},
	})
	require.Len(t, prepared.Transactions, 2)
	require.Equal(t, hexutil.Uint64(2), prepared.ExecutionPlan.BlockRange.First, "from the next block on")
	for _, tx := range prepared.Transactions {
		require.NotNil(t, tx.Gas)
		require.NotNil(t, tx.GasPrice)
		require.Equal(t, bundle.BundleOnly, (*tx.AccessList)[len(*tx.AccessList)-1].Address)
	}

	signed := make([]hexutil.Bytes, len(prepared.Transactions))
	for i, args := range prepared.Transactions {
		tx, err := args.ToTransaction()
		require.NoError(t, err)
		tx, err = types.SignTx(tx, tests.Signer(), r.Wallet.Key)
		require.NoError(t, err)
		signed[i] = tests.Encode(t, tx)
	}
	var planHash common.Hash
	h.MustCall(&planHash, "sonic_submitBundle", map[string]any{"signedTransactions": signed, "executionPlan": prepared.ExecutionPlan})
	require.Len(t, r.Core.Sent(), 1)
	envelope := r.Core.Sent()[0]
	require.Equal(t, bundle.BundleProcessor, *envelope.To())
	require.Equal(t, (*tx0Plan(t, prepared)).Hash(), planHash)

	e := h.Fails("sonic_submitBundle", map[string]any{"signedTransactions": signed[:1], "executionPlan": prepared.ExecutionPlan})
	require.Equal(t, "executionPlan steps count (2) must match signedTransactions count (1)", e.Message)
	e = h.Fails("sonic_submitBundle", map[string]any{"signedTransactions": []any{}})
	require.Equal(t, "signedTransactions must not be empty", e.Message)
}

func tx0Plan(t *testing.T, prepared RPCPreparedBundle) *bundle.ExecutionPlan {
	t.Helper()
	plan, err := ToBundleExecutionPlan(prepared.ExecutionPlan)
	require.NoError(t, err)
	return &plan
}

func TestPrepareBundle_RefusesWhatItCannotPrepare(t *testing.T) {
	r, h := newSonic(t)
	step := func(extra map[string]any) map[string]any {
		s := map[string]any{"from": r.Wallet.Address, "to": r.Recipient, "nonce": "0x2"}
		for k, v := range extra {
			s[k] = v
		}
		return s
	}
	cases := map[string]struct {
		proposal map[string]any
		msg      string
	}{
		"no nonce":  {map[string]any{"steps": []any{map[string]any{"from": r.Wallet.Address}}}, "failed to prepare bundle: failed to flatten transactions: proposed transaction is missing nonce"},
		"no sender": {map[string]any{"steps": []any{map[string]any{"nonce": "0x1"}}}, "failed to prepare bundle: failed to flatten transactions: proposed transaction is missing from field"},
		"both fees": {map[string]any{"steps": []any{step(map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"})}}, "failed to prepare bundle: failed to flatten transactions: proposed transaction cannot have both gasPrice and maxFeePerGas set"},
		"one of":    {map[string]any{"oneOf": true, "steps": []any{step(nil)}}, "failed to prepare bundle: gas estimation not supported when oneOf is set: only one branch executes, making state uncertain for sibling branches"},
		"tolerate":  {map[string]any{"steps": []any{step(map[string]any{"tolerateFailed": true})}}, "failed to prepare bundle: gas estimation not supported when tolerateFailed is set: failed transaction leaves state uncertain for subsequent transactions"},
		"range":     {map[string]any{"blockRange": map[string]any{"first": "0x1", "length": "0x100000"}, "steps": []any{step(nil)}}, "failed to prepare bundle: invalid block range: length 1048576 is too large; must be at most 1024 blocks"},
		"past":      {map[string]any{"blockRange": map[string]any{"first": "0x0", "length": "0x1"}, "steps": []any{step(map[string]any{"gas": "0x5208"})}}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := h.Call("sonic_prepareBundle", tc.proposal)
			if tc.msg == "" {
				require.NoError(t, err, string(raw))
				return
			}
			require.Error(t, err)
			require.Equal(t, tc.msg, err.Error())
		})
	}
}

func TestGetBundleInfo(t *testing.T) {
	r, h := newSonic(t)
	hash := common.Hash{7}
	require.Equal(t, "null", string(h.MustCall(nil, "sonic_getBundleInfo", hash)))
	r.Core.SetBundle(hash, core.BundleInfo{BlockNumber: 1, Offset: 1, Count: 1})
	require.JSONEq(t, `{"block":"0x1","position":"0x1","count":"0x1"}`, string(h.MustCall(nil, "sonic_getBundleInfo", hash)))
	r.Core.SetBundle(hash, core.BundleInfo{BlockNumber: 5, Count: 1})
	require.Equal(t, "null", string(h.MustCall(nil, "sonic_getBundleInfo", hash)), "a block not served yet")
}
