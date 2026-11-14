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
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// bundleHarness serves a Carmen core of four blocks, the head the last, with
// a funded account that signs.
func bundleHarness(t *testing.T) (*tests.Harness, *tests.Account) {
	t.Helper()
	account := tests.NewAccount()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		account.Address(): {Balance: big.NewInt(1e18)},
		writer:            {Code: tests.WritesStorage},
		reverter:          {Code: tests.RevertsWithData},
	})
	for range 3 {
		fake.AddBlock()
	}
	fake.SetHead(3)
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		sonic.Register(h.Dispatcher, h.Environment, h.Core)
	}), account
}

type preparedForm struct {
	Transactions []struct {
		From         common.Address    `json:"from"`
		To           *common.Address   `json:"to"`
		Gas          *hexutil.Uint64   `json:"gas"`
		GasPrice     *hexutil.Big      `json:"gasPrice"`
		MaxFeePerGas *hexutil.Big      `json:"maxFeePerGas"`
		Nonce        hexutil.Uint64    `json:"nonce"`
		ChainID      *hexutil.Big      `json:"chainId"`
		AccessList   *types.AccessList `json:"accessList"`
	} `json:"transactions"`
	ExecutionPlan json.RawMessage `json:"executionPlan"`
}

type planForm struct {
	BlockRange struct {
		First  hexutil.Uint64 `json:"first"`
		Length hexutil.Uint64 `json:"length"`
	} `json:"blockRange"`
	OneOf bool `json:"oneOf"`
	Steps []struct {
		From  common.Address `json:"from"`
		Hash  common.Hash    `json:"hash"`
		Steps []any          `json:"steps"`
	} `json:"steps"`
}

func step(account *tests.Account, nonce uint64, fields map[string]any) map[string]any {
	transaction := map[string]any{"from": account.Address(), "to": writer, "nonce": hexutil.Uint64(nonce)}
	for key, value := range fields {
		transaction[key] = value
	}
	return transaction
}

// A prepared bundle names the blocks after the head, estimates the gas of
// each transaction with the ones before it run, prices the transactions with
// the suggestion, and marks every access list with the plan's hash, which
// submitting the signed transactions answers.
func TestBundles_PrepareThenSubmit(t *testing.T) {
	h, account := bundleHarness(t)
	var prepared preparedForm
	h.Call(t, &prepared, "sonic_prepareBundle", map[string]any{"steps": []any{
		step(account, 0, nil),
		map[string]any{"steps": []any{step(account, 1, map[string]any{"gas": "0x186a0"})}},
	}})
	require.Len(t, prepared.Transactions, 2)
	first, second := prepared.Transactions[0], prepared.Transactions[1]
	require.Greater(t, uint64(*first.Gas), uint64(21000+2400+1900), "the estimate with the marker gas")
	require.Equal(t, hexutil.Uint64(100000), *second.Gas, "a given gas stays")
	suggested := h.Environment.SuggestedGasPrice(h.Environment.Head())
	require.Equal(t, suggested, first.GasPrice.ToInt())
	require.Equal(t, suggested, second.GasPrice.ToInt())

	var plan planForm
	require.NoError(t, json.Unmarshal(prepared.ExecutionPlan, &plan))
	require.Equal(t, hexutil.Uint64(4), plan.BlockRange.First)
	require.Equal(t, hexutil.Uint64(bundle.MaxBlockRangeLength), plan.BlockRange.Length)
	require.Len(t, plan.Steps, 2)
	require.Equal(t, account.Address(), plan.Steps[0].From)
	require.Len(t, plan.Steps[1].Steps, 1)

	marker := (*first.AccessList)[len(*first.AccessList)-1]
	require.Equal(t, bundle.BundleOnly, marker.Address)
	planHash := marker.StorageKeys[0]
	require.Equal(t, *second.AccessList, *first.AccessList)

	signed := []hexutil.Bytes{}
	for _, transaction := range prepared.Transactions {
		binary, err := account.Sign(&types.AccessListTx{ChainID: big.NewInt(tests.ChainID), Nonce: uint64(transaction.Nonce),
			To: transaction.To, Gas: uint64(*transaction.Gas), GasPrice: suggested, AccessList: *transaction.AccessList}).MarshalBinary()
		require.NoError(t, err)
		signed = append(signed, binary)
	}
	var hash common.Hash
	h.Call(t, &hash, "sonic_submitBundle", map[string]any{"signedTransactions": signed, "executionPlan": prepared.ExecutionPlan})
	require.Equal(t, planHash, hash)

	pending, _ := h.Core.PoolContent()
	require.Len(t, pending, 1)
	for _, transactions := range pending {
		require.Equal(t, bundle.BundleProcessor, *transactions[0].To())
	}
	require.Zero(t, h.Core.OpenStates())
}

func TestPrepareBundle_RefusesWhatItCannotPrepare(t *testing.T) {
	h, account := bundleHarness(t)
	nested := map[string]any{"steps": []any{step(account, 0, map[string]any{"gas": "0x5208"})}}
	for range 8 {
		nested = map[string]any{"steps": []any{nested}}
	}
	cases := map[string]struct {
		proposal map[string]any
		err      *dispatch.Error
	}{
		"too long a range": {map[string]any{"blockRange": map[string]any{"first": "0x0", "length": "0x401"}, "steps": []any{step(account, 0, nil)}},
			defaultError("failed to prepare bundle: invalid block range: length 1025 is too large; must be at most 1024 blocks")},
		"a past range": {map[string]any{"blockRange": map[string]any{"first": "0x1", "length": "0x2"}, "steps": []any{step(account, 0, nil)}},
			defaultError("failed to prepare bundle: invalid block range: the specified range (first: 1, length: 2) is not in the future relative to current block 3")},
		"no nonce": {map[string]any{"steps": []any{map[string]any{"from": account.Address()}}},
			defaultError("failed to prepare bundle: failed to flatten transactions: proposed transaction is missing nonce")},
		"no sender": {map[string]any{"steps": []any{map[string]any{"nonce": "0x0"}}},
			defaultError("failed to prepare bundle: failed to flatten transactions: proposed transaction is missing from field")},
		"both prices": {map[string]any{"steps": []any{step(account, 0, map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"})}},
			defaultError("failed to prepare bundle: failed to flatten transactions: proposed transaction cannot have both gasPrice and maxFeePerGas set")},
		"an empty group": {map[string]any{"steps": []any{map[string]any{"steps": []any{}}}},
			defaultError("failed to prepare bundle: failed to flatten transactions: invalid execution plan level: must have either executionStep or group")},
		"set code without recipient": {map[string]any{"steps": []any{map[string]any{"from": account.Address(), "nonce": "0x0", "gas": "0x5208",
			"authorizationList": []any{map[string]any{"chainId": "0x0", "address": writer, "nonce": "0x0", "yParity": "0x0", "r": "0x0", "s": "0x0"}}}}},
			defaultError("failed to prepare bundle: invalid execution plan level: invalid transaction in bundle: missing To in SetCodeTx")},
		"no steps": {map[string]any{"steps": []any{}},
			defaultError("failed to prepare bundle: proposed group must include at least one step")},
		"too deep": {map[string]any{"steps": []any{nested}},
			&dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "invalid argument 0: execution plan exceeds maximum nesting depth of 8"}},
		"estimation of a tolerated failure": {map[string]any{"steps": []any{step(account, 0, map[string]any{"tolerateFailed": true})}},
			defaultError("failed to prepare bundle: gas estimation not supported when tolerateFailed is set: failed transaction leaves state uncertain for subsequent transactions")},
		"estimation of a tolerated invalid": {map[string]any{"steps": []any{step(account, 0, map[string]any{"tolerateInvalid": true})}},
			defaultError("failed to prepare bundle: gas estimation not supported when tolerateInvalid is set: invalid transaction may be skipped, leaving state uncertain for subsequent transactions")},
		"estimation of tolerated failures": {map[string]any{"tolerateFailures": true, "steps": []any{step(account, 0, nil)}},
			defaultError("failed to prepare bundle: gas estimation not supported when tolerateFailures is set: failed transactions leave state uncertain for subsequent transactions")},
		"estimation of one of": {map[string]any{"steps": []any{map[string]any{"oneOf": true, "steps": []any{step(account, 0, nil)}}}},
			defaultError("failed to prepare bundle: gas estimation not supported when oneOf is set: only one branch executes, making state uncertain for sibling branches")},
		"a failing estimation": {map[string]any{"steps": []any{step(account, 0, map[string]any{"to": reverter})}},
			defaultError("failed to prepare bundle: gas estimation failed: failed to estimate gas for transaction 0: execution reverted")},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "sonic_prepareBundle", test.proposal))
			require.Zero(t, h.Core.OpenStates())
		})
	}
}

// A proposal whose transactions all name their gas needs no estimation, so
// its flags may let transactions fail. A transaction with a tip and no fee
// cap gets the suggestion as its fee cap.
func TestPrepareBundle_TakesFlagsWithoutEstimation(t *testing.T) {
	h, account := bundleHarness(t)
	var prepared preparedForm
	h.Call(t, &prepared, "sonic_prepareBundle", map[string]any{"oneOf": true, "tolerateFailures": true, "steps": []any{
		step(account, 0, map[string]any{"gas": "0x5208", "maxPriorityFeePerGas": "0x1", "tolerateFailed": true, "tolerateInvalid": true}),
	}})
	require.Nil(t, prepared.Transactions[0].GasPrice)
	require.Equal(t, h.Environment.SuggestedGasPrice(h.Environment.Head()), prepared.Transactions[0].MaxFeePerGas.ToInt())
	require.JSONEq(t, `{"blockRange":{"first":"0x4","length":"0x400"},"tolerateFailures":true,"oneOf":true,"steps":[`+
		`{"tolerateFailed":true,"tolerateInvalid":true,"from":"`+strings.ToLower(account.Address().Hex())+`","hash":"`+
		planStepHash(t, prepared)+`"}]}`, string(prepared.ExecutionPlan))
}

func planStepHash(t *testing.T, prepared preparedForm) string {
	t.Helper()
	var plan planForm
	require.NoError(t, json.Unmarshal(prepared.ExecutionPlan, &plan))
	return plan.Steps[0].Hash.Hex()
}

func TestSubmitBundle_RefusesWhatItCannotSubmit(t *testing.T) {
	h, account := bundleHarness(t)
	plan := func(first, length string, steps ...any) map[string]any {
		return map[string]any{"blockRange": map[string]any{"first": first, "length": length}, "steps": steps}
	}
	leaf := map[string]any{"from": account.Address(), "hash": common.Hash{0x01}}
	cases := map[string]struct {
		bundle map[string]any
		err    string
	}{
		"no transactions": {map[string]any{"executionPlan": plan("0x4", "0x1", leaf)}, "signedTransactions must not be empty"},
		"too long a range": {map[string]any{"signedTransactions": []string{"0x01"}, "executionPlan": plan("0x4", "0x401", leaf)},
			"invalid block range: invalid block range: length 1025 is too large; must be at most 1024 blocks"},
		"an empty group": {map[string]any{"signedTransactions": []string{"0x01"}, "executionPlan": plan("0x4", "0x1", map[string]any{"steps": []any{}})},
			"invalid execution plan: failed to convert execution plan: invalid execution plan level: invalid execution plan level: must have either executionStep or group"},
		"a step without transaction": {map[string]any{"signedTransactions": []string{"0x01", "0x02"}, "executionPlan": plan("0x4", "0x1", leaf)},
			"executionPlan steps count (1) must match signedTransactions count (2)"},
		"an undecodable transaction": {map[string]any{"signedTransactions": []string{"0x01"}, "executionPlan": plan("0x4", "0x1", leaf)},
			"failed to decode bundled transaction 0: typed transaction too short"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, defaultError(test.err), h.Fail(t, "sonic_submitBundle", test.bundle))
		})
	}

	h.Core.SetMode("SendTransaction", tests.Fail)
	signed, err := account.Sign(&types.AccessListTx{ChainID: big.NewInt(tests.ChainID), To: &writer, Gas: 50000, GasPrice: big.NewInt(1e9)}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, defaultError("failed to submit bundle transaction: "+tests.ErrInjected.Error()),
		h.Fail(t, "sonic_submitBundle", map[string]any{"signedTransactions": []hexutil.Bytes{signed}, "executionPlan": plan("0x4", "0x1", leaf)}))
}

func TestGetBundleInfo_AnswersOnceTheBlockIsServed(t *testing.T) {
	h, _ := bundleHarness(t)
	served, stored, unknown := common.Hash{0x01}, common.Hash{0x02}, common.Hash{0x03}
	h.Core.SetBundleInfo(served, core.BundleInfo{BlockNumber: 3, Offset: 2, Count: 4})
	h.Core.SetBundleInfo(stored, core.BundleInfo{BlockNumber: 4, Offset: 0, Count: 1})
	require.Equal(t, `{"block":"0x3","position":"0x2","count":"0x4"}`, h.Result(t, "sonic_getBundleInfo", served))
	require.Equal(t, "null", h.Result(t, "sonic_getBundleInfo", stored))
	require.Equal(t, "null", h.Result(t, "sonic_getBundleInfo", unknown))

	for _, method := range []string{"BundleInfo", "Header"} {
		h.Core.SetMode(method, tests.Fail)
		require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "sonic_getBundleInfo", served).Message)
		h.Core.SetMode(method, tests.Normal)
	}
}

func defaultError(message string) *dispatch.Error {
	return &dispatch.Error{Code: dispatch.CodeDefault, Message: message}
}
