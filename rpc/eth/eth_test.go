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

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// newEth is a harness serving the eth namespace over a core.
func newEth(t *testing.T, c core.Core, mutate ...tests.Mutate) *tests.Harness {
	t.Helper()
	return tests.NewHarness(t, c, func(h *tests.Harness) { eth.Register(h.D, h.Env, c, c, "eth") }, mutate...)
}

// chainOf is a fake chain of n empty blocks, published.
func chainOf(n int) *tests.FakeCore {
	fc := tests.NewFakeCore()
	for range n {
		fc.AddBlock(nil)
	}
	fc.Publish(uint64(n))
	return fc
}
