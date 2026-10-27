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

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestProfiling_IsNotEnabledWithoutTheBuildTag(t *testing.T) {
	harness := tests.NewHarness(t, tests.NewFakeCore(t), func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		debug.Register(dispatcher, environment)
	})
	for _, call := range [][]any{{"debug_memStats"}, {"debug_writeMemProfile", "/tmp/heap"}, {"debug_stacks"}, {"debug_setGCPercent", 50}} {
		failure := harness.Failure(call[0].(string), call[1:]...)
		require.Equal(t, dispatch.CodeDefault, failure.Code, call[0])
		require.Equal(t, "this method is not enabled", failure.Message, call[0])
	}
	failure := harness.Failure("debug_cpuProfile", "/tmp/cpu")
	require.Equal(t, dispatch.CodeInvalidParameters, failure.Code, "the parameters are checked first, as the reference's server checked them")
}
