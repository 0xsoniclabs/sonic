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

package sonic_test

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

var (
	funded   = common.Address{0x0f}
	writer   = common.Address{0xc0}
	reverter = common.Address{0xc1}
)

func newHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		funded: {Balance: big.NewInt(1e18)},
		writer: {Code: tests.WritesStorage},
	})
	fake.AddBlock()
	fake.SetHead(1)
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		sonic.Register(h.Dispatcher, h.Environment, h.Core)
	})
}

// Each limit is the estimate on the state the transactions before it leave,
// with the gas of the bundle's markers: 2400 for the bundle-only address and
// 1900 for the hash of the execution plan.
func TestEstimateGasForTransactions_EstimatesEachAfterTheOnesBefore(t *testing.T) {
	h := newHarness(t)
	recipient := common.Address{0x42}
	write := map[string]any{"from": funded, "to": writer}
	var limits struct {
		GasLimits []hexutil.Uint64 `json:"gasLimits"`
	}
	h.Call(t, &limits, "sonic_estimateGasForTransactions", []any{write, write, map[string]any{"from": funded, "to": recipient}})
	require.Len(t, limits.GasLimits, 3)
	require.Greater(t, limits.GasLimits[0], limits.GasLimits[1], "the second write finds the slot set")
	require.Equal(t, hexutil.Uint64(21000+2400+1900), limits.GasLimits[2])
	require.Zero(t, h.Core.OpenStates())

	require.Equal(t, `{"gasLimits":[]}`, h.Result(t, "sonic_estimateGasForTransactions", []any{}))
}

func TestEstimateGasForTransactions_RefusesWhatItCannotEstimate(t *testing.T) {
	h := newHarness(t)
	call := map[string]any{"to": funded}
	cases := map[string]struct {
		params []any
		err    *dispatch.Error
	}{
		"too many": {params: []any{make([]any, 17)},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "too many transactions to estimate gas for: got 17, max is 16"}},
		"missing block": {params: []any{[]any{call}, "0x2"},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "failed to estimate gas for transaction 0: block does not exists"}},
		"a failing transaction": {params: []any{[]any{call, map[string]any{"to": funded, "gasPrice": "0x1", "maxFeePerGas": "0x1"}}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "failed to estimate gas for transaction 1: both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"}},
		"no transactions": {params: nil,
			err: &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "missing value for required argument 0"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "sonic_estimateGasForTransactions", test.params...))
			require.Zero(t, h.Core.OpenStates())
		})
	}
}
