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

package web3_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/web3"
)

func TestWeb3_ClientVersionAndSha3(t *testing.T) {
	h := tests.NewHarness(t, tests.NewFakeCore(), func(h *tests.Harness) { web3.Register(h.D, "sonic/v9") })
	require.Equal(t, `"sonic/v9"`, string(h.MustCall(nil, "web3_clientVersion")))
	var hash hexutil.Bytes
	h.MustCall(&hash, "web3_sha3", "0x0102")
	require.Equal(t, hexutil.Bytes(crypto.Keccak256([]byte{1, 2})), hash)
	require.Equal(t, -32602, h.Fails("web3_sha3", "zz").Code)
	require.Equal(t, -32602, h.Fails("web3_sha3").Code)
}
