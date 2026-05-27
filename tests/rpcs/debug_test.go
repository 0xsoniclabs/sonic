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
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/stretchr/testify/require"
)

func TestDebugMethods_notEnabled(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	const wantErr = "this method is not enabled"
	var result interface{}

	cases := []struct {
		method string
		args   []any
	}{
		{"debug_verbosity", []any{1}},
		{"debug_vmodule", []any{"p2p=5"}},
		{"debug_memStats", []any{}},
		{"debug_gcStats", []any{}},
		{"debug_cpuProfile", []any{"cpu.out", 1}},
		{"debug_startCPUProfile", []any{"cpu.out"}},
		{"debug_stopCPUProfile", []any{}},
		{"debug_goTrace", []any{"trace.out", 1}},
		{"debug_startGoTrace", []any{"trace.out"}},
		{"debug_stopGoTrace", []any{}},
		{"debug_blockProfile", []any{"block.out", 1}},
		{"debug_setBlockProfileRate", []any{1}},
		{"debug_writeBlockProfile", []any{"block.out"}},
		{"debug_mutexProfile", []any{"mutex.out", 1}},
		{"debug_setMutexProfileFraction", []any{1}},
		{"debug_writeMutexProfile", []any{"mutex.out"}},
		{"debug_writeMemProfile", []any{"mem.out"}},
		{"debug_stacks", []any{nil}},
		{"debug_freeOSMemory", []any{}},
		{"debug_setGCPercent", []any{100}},
		{"debug_setMemoryLimit", []any{int64(1 << 30)}},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			err := client.Client().Call(&result, tc.method, tc.args...)
			require.EqualError(t, err, wantErr)
		})
	}
}
