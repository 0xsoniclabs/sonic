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

package web3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestWeb3(t *testing.T) {
	fc := tests.NewFakeCore(t)
	h := tests.NewHarness(t, fc, func(d *dispatch.Dispatcher, _ *dispatch.Chain) { Register(d, "sonic/test") })
	require.JSONEq(t, `"sonic/test"`, string(h.Result("web3_clientVersion")))
	// keccak256 of the empty input and of "hello"
	require.JSONEq(t, `"0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"`, string(h.Result("web3_sha3", "0x")))
	require.JSONEq(t, `"0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8"`, string(h.Result("web3_sha3", "0x68656c6c6f")))
	require.Equal(t, dispatch.CodeInvalidParams, h.Fails("web3_sha3").Code)
	require.Equal(t, dispatch.CodeInvalidParams, h.Fails("web3_sha3", "hello").Code)
}
