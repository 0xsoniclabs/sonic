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

var errMethodNotEnabled = errors.New("this method is not enabled")

// registerProfilingShadows answers go-ethereum's profiling and logging
// methods of the debug namespace with an error, as the reference does unless
// it is built with the enable_debug tag; the node's own handlers stay
// unreachable.
func registerProfilingShadows(d *dispatch.Dispatcher) {
	type (
		level   struct{ Level int }
		pattern struct{ Pattern string }
		file    struct{ File string }
		fileFor struct {
			File string
			Nsec uint
		}
		rate   struct{ Rate int }
		filter struct{ Filter *string }
		gc     struct{ V int }
		limit  struct{ Limit int64 }
	)
	shadow := func(name string, m dispatch.Method) { d.Add("debug_"+name, m) }
	shadow("verbosity", disabled[level]())
	shadow("vmodule", disabled[pattern]())
	shadow("memStats", disabled[dispatch.None]())
	shadow("gcStats", disabled[dispatch.None]())
	shadow("cpuProfile", disabled[fileFor]())
	shadow("startCPUProfile", disabled[file]())
	shadow("stopCPUProfile", disabled[dispatch.None]())
	shadow("goTrace", disabled[fileFor]())
	shadow("startGoTrace", disabled[file]())
	shadow("stopGoTrace", disabled[dispatch.None]())
	shadow("blockProfile", disabled[fileFor]())
	shadow("setBlockProfileRate", disabled[rate]())
	shadow("writeBlockProfile", disabled[file]())
	shadow("mutexProfile", disabled[fileFor]())
	shadow("setMutexProfileFraction", disabled[rate]())
	shadow("writeMutexProfile", disabled[file]())
	shadow("writeMemProfile", disabled[file]())
	shadow("stacks", disabled[filter]())
	shadow("freeOSMemory", disabled[dispatch.None]())
	shadow("setGCPercent", disabled[gc]())
	shadow("setMemoryLimit", disabled[limit]())
}

// disabled is a method that decodes its parameters and answers that it is not
// enabled.
func disabled[P any]() dispatch.Method {
	return dispatch.Func(func(context.Context, P) (any, error) {
		return nil, errMethodNotEnabled
	})
}
