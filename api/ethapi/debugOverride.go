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

package ethapi

import (
	"errors"
	"runtime"
	"runtime/debug"
)

var errMethodNotEnabled = errors.New("this method is not enabled")

// The following methods shadow the identically-named endpoints from
// go-ethereum's internal/debug.Handler, Registering them here under
// the same debug namespace ensures callers receive an error.

func (api *PublicDebugAPI) Verbosity(level int) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) Vmodule(pattern string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) MemStats() (*runtime.MemStats, error) {
	return nil, errMethodNotEnabled
}

func (api *PublicDebugAPI) GcStats() (*debug.GCStats, error) {
	return nil, errMethodNotEnabled
}

func (api *PublicDebugAPI) CpuProfile(file string, nsec uint) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) StartCPUProfile(file string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) StopCPUProfile() error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) GoTrace(file string, nsec uint) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) StartGoTrace(file string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) StopGoTrace() error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) BlockProfile(file string, nsec uint) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) SetBlockProfileRate(rate int) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) WriteBlockProfile(file string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) MutexProfile(file string, nsec uint) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) SetMutexProfileFraction(rate int) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) WriteMutexProfile(file string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) WriteMemProfile(file string) error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) Stacks(filter *string) (string, error) {
	return "", errMethodNotEnabled
}

func (api *PublicDebugAPI) FreeOSMemory() error {
	return errMethodNotEnabled
}

func (api *PublicDebugAPI) SetGCPercent(v int) (int, error) {
	return 0, errMethodNotEnabled
}

func (api *PublicDebugAPI) SetMemoryLimit(limit int64) (int64, error) {
	return 0, errMethodNotEnabled
}
