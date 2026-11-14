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

package eth_test

import (
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestCall_AnswersTheReturnDataOrTheRevert(t *testing.T) {
	h := newStateHarness(t)
	cases := map[string]struct {
		params []any
		result string
		err    *dispatch.Error
	}{
		"return data": {params: []any{map[string]any{"to": contract}}, result: `"0x000000000000000000000000000000000000000000000000000000000000002a"`},
		"at a block":  {params: []any{map[string]any{"to": contract}, "0x1"}, result: `"0x000000000000000000000000000000000000000000000000000000000000002a"`},
		"revert": {params: []any{map[string]any{"to": reverter}},
			err: &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}},
		"out of gas": {params: []any{map[string]any{"to": spinner, "gas": "0x10000"}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "execution unsuccessful"}},
		"missing block": {params: []any{map[string]any{"to": contract}, "0x2"},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "header not found"}},
		"both pricings": {params: []any{map[string]any{"to": contract, "gasPrice": "0x1", "maxFeePerGas": "0x1"}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"}},
		"state and state diff": {params: []any{map[string]any{"to": contract}, "latest",
			map[common.Address]any{contract: map[string]any{"state": map[string]string{}, "stateDiff": map[string]string{}}}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "account " + contract.Hex() + " has both 'state' and 'stateDiff'"}},
		"overridden code": {params: []any{map[string]any{"to": reverter}, "latest",
			map[common.Address]any{reverter: map[string]any{"code": hexutil.Encode(tests.ReturnsFortyTwo)}}},
			result: `"0x000000000000000000000000000000000000000000000000000000000000002a"`},
		"insufficient funds": {params: []any{map[string]any{"from": common.Address{0x99}, "to": contract, "value": "0x1"}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "err: insufficient funds for gas * price + value: address 0x9900000000000000000000000000000000000000 have 0 want 1 (supplied gas 50000000)"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if test.err != nil {
				require.Equal(t, test.err, h.Fail(t, "eth_call", test.params...))
			} else {
				require.Equal(t, test.result, h.Result(t, "eth_call", test.params...))
			}
			require.Zero(t, h.Core.OpenStates())
		})
	}
}

func TestCall_ReadsHeadersOnly(t *testing.T) {
	h := newStateHarness(t)
	blocks := h.Core.Calls("Block")
	h.Result(t, "eth_call", map[string]any{"to": contract})
	require.Equal(t, blocks, h.Core.Calls("Block"))
}

// BLOCKHASH is one lookup of a header below the block of the call, and none
// at or above it, however far a block override puts the call's number. The
// EVM itself answers numbers 256 below the call's number and further with the
// zero hash.
func TestCall_AnswersBlockHashByOneLookup(t *testing.T) {
	h := newStateHarness(t)
	genesis, err := h.Core.Block(0)
	require.NoError(t, err)
	cases := map[string]struct {
		number   string
		override string
		hash     common.Hash
		lookups  int
	}{
		"below the block":                  {"0x0", "", genesis.Hash, 1},
		"the block itself":                 {"0x1", "", common.Hash{}, 0},
		"override above the head, below":   {"0x0", "0x10", genesis.Hash, 1},
		"override above the head, above":   {"0xf80", "0x1000", common.Hash{}, 0},
		"override far above, out of reach": {"0x0", "0x1000", common.Hash{}, 0},
		"largest override":                 {"0xfffffffffffffff0", "0xffffffffffffffff", common.Hash{}, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			params := []any{map[string]any{"to": hasher, "input": hexutil.Encode(common.HexToHash(test.number).Bytes())}, "latest", nil}
			if test.override != "" {
				params = append(params, map[string]any{"number": test.override})
			}
			headers := h.Core.Calls("Header")
			result := h.Result(t, "eth_call", params...)
			require.Equal(t, `"`+test.hash.Hex()+`"`, result)
			// One header is the block of the call itself.
			require.Equal(t, headers+1+test.lookups, h.Core.Calls("Header"))
		})
	}
}

func TestEstimateGas_SearchesTheLeastGasThatDoesNotFail(t *testing.T) {
	h := newStateHarness(t)
	cases := map[string]struct {
		params []any
		result string
		err    *dispatch.Error
	}{
		"transfer": {params: []any{map[string]any{"from": funded, "to": common.Address{0x42}, "value": "0x1"}}, result: `"0x5208"`},
		"call":     {params: []any{map[string]any{"to": contract}}, result: `"0x521a"`},
		"revert": {params: []any{map[string]any{"to": reverter}},
			err: &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}},
		"spinner": {params: []any{map[string]any{"to": spinner, "gas": "0x100000"}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "execution unsuccessful"}},
		"funds for the value": {params: []any{map[string]any{"from": common.Address{0x99}, "to": contract, "value": "0x1", "gasPrice": "0x1"}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "insufficient funds for transfer"}},
		"missing block": {params: []any{map[string]any{"to": contract}, "0x2"},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "block does not exists"}},
		"unknown hash": {params: []any{map[string]any{"to": contract}, common.Hash{0x01}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "block with hash " + common.Hash{0x01}.Hex() + " not found"}},
		"missing selector": {params: []any{map[string]any{"to": contract}, map[string]any{}},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "invalid block number or hash"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if test.err != nil {
				require.Equal(t, test.err, h.Fail(t, "eth_estimateGas", test.params...))
			} else {
				require.Equal(t, test.result, h.Result(t, "eth_estimateGas", test.params...))
			}
			require.Zero(t, h.Core.OpenStates())
		})
	}
}

func TestEstimateGas_OpensOneStateForAllRuns(t *testing.T) {
	h := newStateHarness(t)
	states := h.Core.Calls("State")
	h.Result(t, "eth_estimateGas", map[string]any{"to": contract})
	require.Equal(t, states+1, h.Core.Calls("State"))
}

func TestCreateAccessList_AnswersTheListAndTheGas(t *testing.T) {
	h := newStateHarness(t)
	var result struct {
		AccessList []any          `json:"accessList"`
		GasUsed    hexutil.Uint64 `json:"gasUsed"`
		Error      string         `json:"error"`
	}
	h.Call(t, &result, "eth_createAccessList", map[string]any{"from": funded, "to": contract})
	require.Empty(t, result.AccessList, "the recipient is left out")
	require.NotZero(t, result.GasUsed)
	require.Empty(t, result.Error)

	h.Call(t, &result, "eth_createAccessList", map[string]any{"from": funded, "to": reverter, "gas": "0x10000"})
	require.Equal(t, "execution reverted", result.Error, "with gas given the revert is the call's")
	require.Equal(t, &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"},
		h.Fail(t, "eth_createAccessList", map[string]any{"from": funded, "to": reverter}), "without, it fails the estimate")
	require.Zero(t, h.Core.OpenStates())

	require.Equal(t, "header not found", h.Fail(t, "eth_createAccessList", map[string]any{"to": contract}, "0x2").Message)
}
