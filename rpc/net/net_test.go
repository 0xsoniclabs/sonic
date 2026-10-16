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

package net

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestVersionIsTheNetworkID(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithChainID(146))
	h := tests.NewHarness(t, fc, tests.Default(), func(d *dispatch.Dispatcher, _ *dispatch.Chain) {
		Register(d, fc)
	})
	require.JSONEq(t, `"146"`, string(h.Result("net_version")))
}
