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
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

var (
	funded   = common.Address{0x0f}
	contract = common.Address{0xc0}
	reverter = common.Address{0xc1}
	spinner  = common.Address{0xc2}
	hasher   = common.Address{0xc3}
	// slot is the storage slot "0x01" names.
	slot = common.HexToHash("0x01")
)

// newStateHarness serves a Carmen core of three blocks, the head the second,
// with a funded account and contracts.
func newStateHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		funded:   {Balance: big.NewInt(1e18), Nonce: 7},
		contract: {Code: tests.ReturnsFortyTwo, Storage: map[common.Hash]common.Hash{slot: {0x02}}},
		reverter: {Code: tests.RevertsWithData},
		spinner:  {Code: tests.Spins},
		hasher:   {Code: tests.ReturnsBlockHash},
	})
	fake.AddBlock()
	fake.SetHead(1)
	fake.AddBlock() // stored, not served
	fake.SetPoolNonce(funded, 9)
	return newHarness(t, fake)
}

func TestStateGetters_ReadTheArchiveAtEachSelector(t *testing.T) {
	h := newStateHarness(t)
	served, err := h.Core.Block(1)
	require.NoError(t, err)
	unserved, err := h.Core.Block(2)
	require.NoError(t, err)
	selectors := map[string]struct {
		block any // nil for none
		err   string
	}{
		"none":             {nil, ""},
		"latest":           {"latest", ""},
		"safe":             {"safe", ""},
		"earliest":         {"earliest", ""},
		"number":           {"0x1", ""},
		"hash":             {served.Hash, ""},
		"hash object":      {map[string]any{"blockHash": served.Hash}, ""},
		"above the head":   {"0x2", "header not found"},
		"hash above head":  {unserved.Hash, "header not found"},
		"unknown hash":     {common.Hash{0x01}, "header not found"},
		"missing selector": {map[string]any{}, "unknown header selector"},
	}
	methods := map[string]struct {
		params []any
		want   string
	}{
		"eth_getBalance":          {[]any{funded}, `"0xde0b6b3a7640000"`},
		"eth_getCode":             {[]any{contract}, `"` + hexutil.Encode(tests.ReturnsFortyTwo) + `"`},
		"eth_getStorageAt":        {[]any{contract, "0x01"}, `"0x0200000000000000000000000000000000000000000000000000000000000000"`},
		"eth_getTransactionCount": {[]any{funded}, `"0x7"`},
	}
	for method, call := range methods {
		for name, selector := range selectors {
			t.Run(method+"/"+name, func(t *testing.T) {
				params := call.params
				if selector.block != nil {
					params = append(append([]any{}, params...), selector.block)
				}
				if selector.err != "" {
					require.Equal(t, selector.err, h.Fail(t, method, params...).Message)
				} else {
					require.Equal(t, call.want, h.Result(t, method, params...))
				}
				require.Zero(t, h.Core.OpenStates(), "every state is released")
			})
		}
	}
}

func TestGetTransactionCount_AnswersThePoolsNonceForPending(t *testing.T) {
	h := newStateHarness(t)
	states := h.Core.Calls("State")
	require.Equal(t, `"0x9"`, h.Result(t, "eth_getTransactionCount", funded, "pending"))
	require.Equal(t, states, h.Core.Calls("State"), "the pending nonce reads no state")
}

func TestStateGetters_PassOnAFailingState(t *testing.T) {
	h := newStateHarness(t)
	h.Core.SetMode("State", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getBalance", funded).Message)
	h.Core.SetMode("State", tests.NotFound)
	require.Equal(t, "header not found", h.Fail(t, "eth_getBalance", funded).Message)
}

func TestGetAccount_AnswersTheFieldsOfTheProof(t *testing.T) {
	h := newStateHarness(t)
	var account struct {
		Balance     *hexutil.Big   `json:"balance"`
		Nonce       hexutil.Uint64 `json:"nonce"`
		CodeHash    common.Hash    `json:"codeHash"`
		StorageRoot common.Hash    `json:"storageRoot"`
	}
	h.Call(t, &account, "eth_getAccount", funded, "latest")
	require.Equal(t, int64(1e18), account.Balance.ToInt().Int64())
	require.Equal(t, hexutil.Uint64(7), account.Nonce)
	require.Zero(t, h.Core.OpenStates())
}

func TestGetProof_ProvesTheAccountAndItsSlots(t *testing.T) {
	h := newStateHarness(t)
	var proof struct {
		Address      common.Address `json:"address"`
		AccountProof []string       `json:"accountProof"`
		StorageProof []struct {
			Key   string       `json:"key"`
			Value *hexutil.Big `json:"value"`
		} `json:"storageProof"`
	}
	h.Call(t, &proof, "eth_getProof", contract, []string{"0x01", "1"}, "latest")
	require.Equal(t, contract, proof.Address)
	require.NotEmpty(t, proof.AccountProof)
	require.Len(t, proof.StorageProof, 2)
	require.Equal(t, slot.Hex(), proof.StorageProof[0].Key)
	require.Equal(t, new(big.Int).SetBytes(common.Hash{0x02}.Bytes()), proof.StorageProof[0].Value.ToInt())
	require.Zero(t, h.Core.OpenStates())
}

func TestGetProof_RefusesBadKeysBeforeReadingState(t *testing.T) {
	h := newStateHarness(t)
	tooMany := make([]string, 1025)
	for i := range tooMany {
		tooMany[i] = "0x01"
	}
	cases := map[string]struct {
		keys []string
		err  string
	}{
		"at the limit":   {tooMany[:1024], ""},
		"over the limit": {tooMany, "too many storage keys requested (max 1024, got 1025)"},
		"too long":       {[]string{"0x" + strings.Repeat("11", 33)}, `storage key too long (want at most 32 bytes): "0x` + strings.Repeat("11", 33) + `"`},
		"not hex":        {[]string{"0xzz"}, `invalid hex in storage key: "0xzz"`},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			states := h.Core.Calls("State")
			if test.err == "" {
				h.Result(t, "eth_getProof", contract, test.keys, "latest")
				return
			}
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: test.err}, h.Fail(t, "eth_getProof", contract, test.keys, "latest"))
			require.Equal(t, states, h.Core.Calls("State"))
		})
	}
}
