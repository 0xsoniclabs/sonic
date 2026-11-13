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

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/web3"
	"github.com/stretchr/testify/require"
)

func TestWeb3_AnswersTheClientAndHashes(t *testing.T) {
	h := tests.NewHarness(t, tests.NewFakeCore(), func(h *tests.Harness) {
		web3.Register(h.Dispatcher, "Sonic/v1.0.0")
	})
	require.Equal(t, `"Sonic/v1.0.0"`, h.Result(t, "web3_clientVersion"))
	require.Equal(t, `"0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"`, h.Result(t, "web3_sha3", "0x"))
	require.Equal(t, `"0x5fe7f977e71dba2ea1a68e21057beebb9be2ac30c6410aa38d4f3fbe41dcffd2"`, h.Result(t, "web3_sha3", "0x01"))
	require.Equal(t, dispatch.CodeInvalidParams, h.Fail(t, "web3_sha3", "01").Code)
}
