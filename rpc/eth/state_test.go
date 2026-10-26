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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	holder   = common.HexToAddress("0xa0")
	contract = common.HexToAddress("0xc1")
	// counter increments slot 0 and returns its new value.
	counter = []byte{0x60, 0x00, 0x54, 0x60, 0x01, 0x01, 0x80, 0x60, 0x00, 0x55, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
)

// stateChain is a core with real state, three blocks archived and the
// harness of the namespace over it.
type stateChain struct {
	core    *tests.CarmenCore
	harness *tests.Harness
	blocks  []common.Hash
}

func newStateChain(t *testing.T, accounts map[common.Address]tests.Account, mutations ...tests.Mutate) stateChain {
	t.Helper()
	all := map[common.Address]tests.Account{
		holder:   {Balance: big.NewInt(1234), Nonce: 5},
		contract: {Code: counter, Storage: map[common.Hash]common.Hash{{}: common.BigToHash(big.NewInt(7))}},
	}
	for address, account := range accounts {
		all[address] = account
	}
	carmenCore := tests.NewCarmenCore(t, all)
	hashes := []common.Hash{}
	for range 2 {
		hashes = append(hashes, carmenCore.AppendBlock(tests.BlockSpec{}).Hash)
	}
	carmenCore.PublishHead(2)
	harness := newEth(t, carmenCore.FakeCore, mutations...)
	harness.WaitForHead(2)
	return stateChain{core: carmenCore, harness: harness, blocks: hashes}
}

func TestStateGetters_ReadTheStateOfTheBlock(t *testing.T) {
	chain := newStateChain(t, nil)
	harness := chain.harness
	for _, block := range []any{nil, "latest", "0x1", map[string]any{"blockHash": chain.blocks[1]}, "pending", "earliest"} {
		parameters := []any{holder}
		if block != nil {
			parameters = append(parameters, block)
		}
		require.JSONEq(t, `"0x4d2"`, string(harness.Result("eth_getBalance", parameters...)), "block %v", block)
	}
	require.JSONEq(t, `"0x"`, string(harness.Result("eth_getCode", holder)))
	require.JSONEq(t, `"0x6000546001018060005560005260206000f3"`, string(harness.Result("eth_getCode", contract, "latest")))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000000000000000000000000007"`, string(harness.Result("eth_getStorageAt", contract, "0x0")))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000000000000000000000000007"`, string(harness.Result("eth_getStorageAt", contract, "0", "latest")), "the key is converted leniently")
	require.JSONEq(t, `"0x5"`, string(harness.Result("eth_getTransactionCount", holder)))
	require.JSONEq(t, `"0x5"`, string(harness.Result("ftm_getTransactionCount", holder, "latest")))
}

// TestGetTransactionCount_AnswersThePoolForPending holds I5.
func TestGetTransactionCount_AnswersThePoolForPending(t *testing.T) {
	chain := newStateChain(t, nil)
	chain.core.SetPoolNonce(holder, 9)
	before := chain.core.States()
	require.JSONEq(t, `"0x9"`, string(chain.harness.Result("eth_getTransactionCount", holder, "pending")))
	require.Equal(t, before, chain.core.States(), "no state is opened")
}

// TestStateGetters_WordAMissingBlockAsTheReference holds V1 and F3.
func TestStateGetters_WordAMissingBlockAsTheReference(t *testing.T) {
	chain := newStateChain(t, nil)
	cases := map[string]any{
		"a number above the head": "0x3",
		"an unknown hash":         common.Hash{0xee},
		"an unknown hash object":  map[string]any{"blockHash": crypto.Keccak256Hash([]byte("none"))},
	}
	for _, method := range []string{"eth_getBalance", "eth_getCode", "eth_getTransactionCount", "eth_getAccount"} {
		for name, block := range cases {
			require.Equal(t, "header not found", chain.harness.Failure(method, holder, block).Message, "%s with %s", method, name)
		}
		require.Equal(t, "unknown header selector", chain.harness.Failure(method, holder, map[string]any{}).Message, method)
	}
	require.Equal(t, "header not found", chain.harness.Failure("eth_getStorageAt", contract, "0x0", "0x3").Message)
	require.Equal(t, "header not found", chain.harness.Failure("eth_getProof", contract, []string{}, "0x3").Message)
}

// TestStateGetters_ReadNoHeader holds H2: a state getter needs the state,
// not the header of its block.
func TestStateGetters_ReadNoHeader(t *testing.T) {
	chain := newStateChain(t, nil)
	before := chain.core.Calls("Header")
	chain.harness.Result("eth_getBalance", holder, "0x1")
	chain.harness.Result("eth_getCode", holder, "0x1")
	chain.harness.Result("eth_getStorageAt", contract, "0x0", "0x1")
	chain.harness.Result("eth_getTransactionCount", holder, "0x1")
	require.Equal(t, before, chain.core.Calls("Header"))
}

// TestStateGetters_ReleaseEveryState holds I7.
func TestStateGetters_ReleaseEveryState(t *testing.T) {
	chain := newStateChain(t, nil)
	before := chain.core.States()
	chain.harness.Result("eth_getBalance", holder)
	chain.harness.Result("eth_getAccount", holder)
	chain.harness.Result("eth_getProof", contract, []string{"0x0"})
	after := chain.core.States()
	require.Equal(t, before.Opened+3, after.Opened)
	require.Equal(t, after.Opened, after.Released)
}

func TestGetAccount_ReadsTheProvenFields(t *testing.T) {
	chain := newStateChain(t, nil)
	answer := decodeMap(t, chain.harness.Result("eth_getAccount", contract, "latest"))
	require.Equal(t, crypto.Keccak256Hash(counter).Hex(), answer["codeHash"])
	require.NotEqual(t, types.EmptyRootHash.Hex(), answer["storageRoot"], "the contract has storage")
	require.Equal(t, "0x0", answer["balance"])
	require.Equal(t, "0x0", answer["nonce"])

	answer = decodeMap(t, chain.harness.Result("eth_getAccount", holder))
	require.Equal(t, types.EmptyCodeHash.Hex(), answer["codeHash"])
	require.Equal(t, "0x4d2", answer["balance"])
	require.Equal(t, "0x5", answer["nonce"])
}

func TestGetProof_ProvesTheAccountAndItsSlots(t *testing.T) {
	chain := newStateChain(t, nil)
	answer := decodeMap(t, chain.harness.Result("eth_getProof", contract, []string{"0x0", "0x1"}))
	require.Equal(t, strings.ToLower(contract.Hex()), strings.ToLower(answer["address"].(string)))
	require.NotEmpty(t, answer["accountProof"])
	require.Equal(t, crypto.Keccak256Hash(counter).Hex(), answer["codeHash"])
	slots := answer["storageProof"].([]any)
	require.Len(t, slots, 2)
	first := slots[0].(map[string]any)
	require.Equal(t, common.Hash{}.Hex(), first["key"])
	require.Equal(t, "0x7", first["value"])
	require.NotEmpty(t, first["proof"])
	require.Equal(t, "0x0", slots[1].(map[string]any)["value"])
}

// TestGetProof_ChecksTheKeysBeforeAnyState holds V10.
func TestGetProof_ChecksTheKeysBeforeAnyState(t *testing.T) {
	chain := newStateChain(t, nil)
	before := chain.core.States()
	tooMany := make([]string, 1025)
	for index := range tooMany {
		tooMany[index] = "0x1"
	}
	cases := map[string]struct {
		keys    []string
		message string
	}{
		"too many keys": {tooMany, "too many storage keys requested (max 1024, got 1025)"},
		"invalid hex":   {[]string{"0xzz"}, `invalid hex in storage key: "0xzz"`},
		"too long":      {[]string{"0x" + strings.Repeat("00", 33)}, `storage key too long (want at most 32 bytes): "0x` + strings.Repeat("00", 33) + `"`},
	}
	for name, test := range cases {
		failure := chain.harness.Failure("eth_getProof", contract, test.keys)
		require.Equal(t, dispatch.CodeInvalidParameters, failure.Code, name)
		require.Equal(t, test.message, failure.Message, name)
	}
	require.Equal(t, before, chain.core.States(), "no state is opened")

	answer := decodeMap(t, chain.harness.Result("eth_getProof", contract, []string{"1", "0X01"}))
	slots := answer["storageProof"].([]any)
	require.Equal(t, common.BigToHash(big.NewInt(1)).Hex(), slots[0].(map[string]any)["key"], "an odd key is padded")
	require.Equal(t, common.BigToHash(big.NewInt(1)).Hex(), slots[1].(map[string]any)["key"])
}
