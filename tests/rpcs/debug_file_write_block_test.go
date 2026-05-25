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

package rpcs

import (
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/stretchr/testify/require"
)

// TestDebugFileWriteMethods_AreBlockedOverHTTP verifies that the file-writing
// debug RPC methods are disabled when called over HTTP. These methods allow
// arbitrary file creation and must not be reachable from the network.
func TestDebugFileWriteMethods_AreBlockedOverHTTP(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	methods := []struct {
		name   string
		params []any
	}{
		{"debug_startCPUProfile", []any{"/tmp/sonic-test-cpu.prof"}},
		{"debug_stopCPUProfile", []any{}},
		{"debug_cpuProfile", []any{"/tmp/sonic-test-cpu.prof", 1}},
		{"debug_startGoTrace", []any{"/tmp/sonic-test-trace.out"}},
		{"debug_stopGoTrace", []any{}},
		{"debug_goTrace", []any{"/tmp/sonic-test-trace.out", 1}},
		{"debug_blockProfile", []any{"/tmp/sonic-test-block.prof", 1}},
		{"debug_writeBlockProfile", []any{"/tmp/sonic-test-block.prof"}},
		{"debug_mutexProfile", []any{"/tmp/sonic-test-mutex.prof", 1}},
		{"debug_writeMutexProfile", []any{"/tmp/sonic-test-mutex.prof"}},
		{"debug_writeMemProfile", []any{"/tmp/sonic-test-mem.prof"}},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			var result any
			err := client.Client().Call(&result, m.name, m.params...)
			require.Error(t, err, "method %s must be blocked over HTTP", m.name)
			require.True(t,
				strings.Contains(err.Error(), "disabled"),
				"expected 'disabled' in error for %s, got: %v", m.name, err,
			)
		})
	}
}
