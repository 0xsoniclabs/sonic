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

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/web3"
)

func newWeb3(t *testing.T) *tests.Harness {
	return tests.NewHarness(t, tests.NewFakeCore(t), func(dispatcher *dispatch.Dispatcher, _ *exec.Environment) {
		web3.Register(dispatcher, "Sonic/v2.3.0/linux-amd64/go1.26")
	})
}

func TestClientVersion_IsTheNameOfTheNode(t *testing.T) {
	require.JSONEq(t, `"Sonic/v2.3.0/linux-amd64/go1.26"`, string(newWeb3(t).Result("web3_clientVersion")))
}

func TestSha3_HashesTheInput(t *testing.T) {
	harness := newWeb3(t)
	require.JSONEq(t, `"0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"`, string(harness.Result("web3_sha3", "0x")))
	require.Equal(t, "parameter 0 (input): json: cannot unmarshal hex string without 0x prefix into Go value of type hexutil.Bytes", harness.Failure("web3_sha3", "00").Message)
}
