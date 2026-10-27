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

// errNotEnabled is the answer of go-ethereum's profiling methods in a build
// without enable_debug, as the reference shadowed them.
var errNotEnabled = errors.New("this method is not enabled")

// registerProfiling adds the profiling methods of go-ethereum's debug
// handler, with the parameters of the reference's shadows, answering that
// they are not enabled: none of them reaches the node's handler.
func registerProfiling(dispatcher *dispatch.Dispatcher) {
	file := dispatch.Required[string]("file")
	seconds := dispatch.Required[uint]("nsec")
	rate := dispatch.Required[int]("rate")
	for name, parameters := range map[string][]dispatch.Parameter{
		"debug_verbosity":               {dispatch.Required[int]("level")},
		"debug_vmodule":                 {dispatch.Required[string]("pattern")},
		"debug_memStats":                nil,
		"debug_gcStats":                 nil,
		"debug_cpuProfile":              {file, seconds},
		"debug_startCPUProfile":         {file},
		"debug_stopCPUProfile":          nil,
		"debug_goTrace":                 {file, seconds},
		"debug_startGoTrace":            {file},
		"debug_stopGoTrace":             nil,
		"debug_blockProfile":            {file, seconds},
		"debug_setBlockProfileRate":     {rate},
		"debug_writeBlockProfile":       {file},
		"debug_mutexProfile":            {file, seconds},
		"debug_setMutexProfileFraction": {rate},
		"debug_writeMutexProfile":       {file},
		"debug_writeMemProfile":         {file},
		"debug_stacks":                  {dispatch.Optional[string]("filter")},
		"debug_freeOSMemory":            nil,
		"debug_setGCPercent":            {dispatch.Required[int]("percent")},
		"debug_setMemoryLimit":          {dispatch.Required[int64]("limit")},
	} {
		dispatcher.Add(name, notEnabled, parameters...)
	}
}

func notEnabled(context.Context, dispatch.Arguments) (any, error) {
	return nil, errNotEnabled
}
