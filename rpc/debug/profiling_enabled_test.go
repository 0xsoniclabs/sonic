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

//go:build enable_debug

package debug_test

import (
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/stretchr/testify/require"
)

func TestProfiling_IsLeftToTheNode(t *testing.T) {
	h := tests.NewHarness(t, tests.NewFakeCore(), func(h *tests.Harness) { debug.Register(h.Dispatcher) })
	require.False(t, h.Dispatcher.Serves("debug_verbosity"))
}
