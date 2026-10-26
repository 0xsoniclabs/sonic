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
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestState_Getters(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)

	var balance hexutil.Big
	h.MustCall(&balance, "eth_getBalance", r.Writer, "latest")
	require.Equal(t, int64(1), balance.ToInt().Int64())
	h.MustCall(&balance, "eth_getBalance", r.Writer)
	require.Equal(t, int64(1), balance.ToInt().Int64(), "latest by default")

	var code hexutil.Bytes
	h.MustCall(&code, "eth_getCode", r.Writer, "0x1")
	require.Equal(t, hexutil.Bytes(tests.StorageWritingCode), code)
	require.Equal(t, `"0x"`, string(h.MustCall(nil, "eth_getCode", r.Recipient, "latest")))

	var slot hexutil.Bytes
	h.MustCall(&slot, "eth_getStorageAt", r.Writer, "0x0", "latest")
	require.Equal(t, make(hexutil.Bytes, 32), slot)

	var nonce hexutil.Uint64
	h.MustCall(&nonce, "eth_getTransactionCount", r.Wallet.Address, "latest")
	require.Zero(t, nonce)
	r.Core.SetNonce(r.Wallet.Address, 5)
	h.MustCall(&nonce, "eth_getTransactionCount", r.Wallet.Address, "pending")
	require.Equal(t, hexutil.Uint64(5), nonce, "pending is the pool's nonce")
}

// TestState_AMissingBlockIsHeaderNotFound is the wording of every state
// getter of the reference (F3).
func TestState_AMissingBlockIsHeaderNotFound(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	for _, method := range []string{"eth_getBalance", "eth_getCode", "eth_getTransactionCount", "eth_getAccount"} {
		require.Equal(t, "header not found", h.Fails(method, r.Writer, "0x9").Message, method)
		require.Equal(t, "header not found", h.Fails(method, r.Writer, common.Hash{1}).Message, method)
	}
	require.Equal(t, "header not found", h.Fails("eth_getStorageAt", r.Writer, "0x0", "0x9").Message)
	require.Equal(t, "header not found", h.Fails("eth_getProof", r.Writer, []string{}, "0x9").Message)
}

// TestState_ReadsNoHeader is H2: a state getter opens the state of its block
// and reads no header or block.
func TestState_ReadsNoHeader(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	headers, blocks := r.Core.Calls("Header"), r.Core.Calls("Block")
	h.MustCall(nil, "eth_getBalance", r.Writer, "0x0")
	h.MustCall(nil, "eth_getCode", r.Writer, "0x1")
	h.MustCall(nil, "eth_getStorageAt", r.Writer, "0x0", "latest")
	h.MustCall(nil, "eth_getTransactionCount", r.Writer, "latest")
	require.Equal(t, headers, r.Core.Calls("Header"))
	require.Equal(t, blocks, r.Core.Calls("Block"))
}

func TestState_EveryOpenedStateIsReleasedOnce(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	h.MustCall(nil, "eth_getBalance", r.Writer, "latest")
	h.MustCall(nil, "eth_getProof", r.Writer, []string{"0x0"}, "latest")
	h.MustCall(nil, "eth_call", map[string]any{"to": r.Returner}, "latest")
	opened, released := r.Core.TrackStates()
	require.Positive(t, opened)
	require.Equal(t, opened, released)
}

func TestState_GetAccountAndGetProofAgreeWithTheState(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	var account eth.GetAccountResult
	h.MustCall(&account, "eth_getAccount", r.Writer, "latest")
	require.Equal(t, uint64(1), (*uint256.Int)(account.Balance).Uint64())
	require.Equal(t, crypto.Keccak256Hash(tests.StorageWritingCode), account.CodeHash)

	var proof eth.AccountResult
	h.MustCall(&proof, "eth_getProof", r.Writer, []string{"0x0", "0x1"}, "latest")
	require.Equal(t, r.Writer, proof.Address)
	require.Equal(t, account.CodeHash, proof.CodeHash)
	require.Equal(t, account.StorageRoot, proof.StorageHash)
	require.NotEmpty(t, proof.AccountProof)
	require.Len(t, proof.StorageProof, 2)
	require.Equal(t, common.BigToHash(common.Big1).Hex(), proof.StorageProof[1].Key)
	require.Zero(t, proof.StorageProof[0].Value.ToInt().Sign())
}

// TestState_GetProofChecksItsKeysBeforeAnyState is V10.
func TestState_GetProofChecksItsKeysBeforeAnyState(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	states := r.Core.Calls("State")
	keys := make([]string, 1025)
	for i := range keys {
		keys[i] = "0x0"
	}
	e := h.Fails("eth_getProof", r.Writer, keys, "latest")
	require.Equal(t, -32602, e.Code)
	require.Equal(t, "too many storage keys requested (max 1024, got 1025)", e.Message)

	e = h.Fails("eth_getProof", r.Writer, []string{"0x" + strings.Repeat("11", 33)}, "latest")
	require.Equal(t, -32602, e.Code)
	require.Contains(t, e.Message, "storage key too long")
	e = h.Fails("eth_getProof", r.Writer, []string{"0xzz"}, "latest")
	require.Equal(t, -32602, e.Code)
	require.Equal(t, `invalid hex in storage key: "0xzz"`, e.Message)
	require.Equal(t, states, r.Core.Calls("State"))

	var proof eth.AccountResult
	h.MustCall(&proof, "eth_getProof", r.Writer, []string{"1"}, "latest")
	require.Equal(t, common.BigToHash(common.Big1).Hex(), proof.StorageProof[0].Key, "a short key is padded")
}
