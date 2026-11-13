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

package debug

import (
	"context"
	"errors"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// errNotEnabled answers the profiling methods of a build without the
// enable_debug tag, which leaves profiling out of reach of clients.
var errNotEnabled = errors.New("this method is not enabled")

// The parameters of the profiling methods, as go-ethereum's profiling
// handler takes them, so that a call is decoded as it would be there.
type (
	levelParams struct {
		Level int
	}
	patternParams struct {
		Pattern string
	}
	fileParams struct {
		File string
	}
	fileDurationParams struct {
		File        string
		NanoSeconds uint
	}
	rateParams struct {
		Rate int
	}
	filterParams struct {
		Filter *string
	}
	limitParams struct {
		Limit int64
	}
)

// registerProfiling answers each method of go-ethereum's profiling handler,
// which the debug namespace would otherwise reach, with errNotEnabled.
func registerProfiling(dispatcher *dispatch.Dispatcher) {
	dispatch.Add(dispatcher, "debug_verbosity", notEnabled[levelParams])
	dispatch.Add(dispatcher, "debug_vmodule", notEnabled[patternParams])
	dispatch.Add(dispatcher, "debug_memStats", notEnabled[struct{}])
	dispatch.Add(dispatcher, "debug_gcStats", notEnabled[struct{}])
	dispatch.Add(dispatcher, "debug_cpuProfile", notEnabled[fileDurationParams])
	dispatch.Add(dispatcher, "debug_startCPUProfile", notEnabled[fileParams])
	dispatch.Add(dispatcher, "debug_stopCPUProfile", notEnabled[struct{}])
	dispatch.Add(dispatcher, "debug_goTrace", notEnabled[fileDurationParams])
	dispatch.Add(dispatcher, "debug_startGoTrace", notEnabled[fileParams])
	dispatch.Add(dispatcher, "debug_stopGoTrace", notEnabled[struct{}])
	dispatch.Add(dispatcher, "debug_blockProfile", notEnabled[fileDurationParams])
	dispatch.Add(dispatcher, "debug_setBlockProfileRate", notEnabled[rateParams])
	dispatch.Add(dispatcher, "debug_writeBlockProfile", notEnabled[fileParams])
	dispatch.Add(dispatcher, "debug_mutexProfile", notEnabled[fileDurationParams])
	dispatch.Add(dispatcher, "debug_setMutexProfileFraction", notEnabled[rateParams])
	dispatch.Add(dispatcher, "debug_writeMutexProfile", notEnabled[fileParams])
	dispatch.Add(dispatcher, "debug_writeMemProfile", notEnabled[fileParams])
	dispatch.Add(dispatcher, "debug_stacks", notEnabled[filterParams])
	dispatch.Add(dispatcher, "debug_freeOSMemory", notEnabled[struct{}])
	dispatch.Add(dispatcher, "debug_setGCPercent", notEnabled[rateParams])
	dispatch.Add(dispatcher, "debug_setMemoryLimit", notEnabled[limitParams])
}

func notEnabled[Params any](context.Context, Params) (any, error) {
	return nil, errNotEnabled
}
