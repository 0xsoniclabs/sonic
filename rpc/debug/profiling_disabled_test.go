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

//go:build !enable_debug

package debug_test

import (
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/stretchr/testify/require"
)

func TestProfiling_IsNotEnabled(t *testing.T) {
	h := tests.NewHarness(t, tests.NewFakeCore(), func(h *tests.Harness) { debug.Register(h.Dispatcher) })
	cases := map[string][]any{
		"debug_verbosity":               {3},
		"debug_vmodule":                 {"eth/*=5"},
		"debug_memStats":                nil,
		"debug_gcStats":                 nil,
		"debug_cpuProfile":              {"file", 1},
		"debug_startCPUProfile":         {"file"},
		"debug_stopCPUProfile":          nil,
		"debug_goTrace":                 {"file", 1},
		"debug_startGoTrace":            {"file"},
		"debug_stopGoTrace":             nil,
		"debug_blockProfile":            {"file", 1},
		"debug_setBlockProfileRate":     {1},
		"debug_writeBlockProfile":       {"file"},
		"debug_mutexProfile":            {"file", 1},
		"debug_setMutexProfileFraction": {1},
		"debug_writeMutexProfile":       {"file"},
		"debug_writeMemProfile":         {"file"},
		"debug_stacks":                  nil,
		"debug_freeOSMemory":            nil,
		"debug_setGCPercent":            {100},
		"debug_setMemoryLimit":          {1 << 30},
	}
	for method, params := range cases {
		t.Run(method, func(t *testing.T) {
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "this method is not enabled"}, h.Fail(t, method, params...))
		})
	}
	require.Equal(t, dispatch.CodeInvalidParams, h.Fail(t, "debug_verbosity", "loud").Code, "parameters are decoded first")
}
