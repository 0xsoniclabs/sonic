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

package gossip

import "errors"

// errDebugMethodDisabled is returned by all methods of debugFileWriteBlocker.
var errDebugMethodDisabled = errors.New("method disabled from security reasons")

// debugFileWriteBlocker shadows the file-writing methods of go-ethereum's
// internal/debug.HandlerT that are automatically registered under the "debug"
// JSON-RPC namespace by node/api.go.
//
// go-ethereum's rpc.Server merges all services registered under the same
// namespace into a single callback map, with later registrations silently
// overwriting earlier ones for the same method name. By appending this struct
// last in Service.APIs() we ensure the dangerous methods are replaced with
// stubs that always return an error.
//
// Affected RPC methods and their risk:
//   - debug_startCPUProfile / debug_stopCPUProfile / debug_cpuProfile
//   - debug_startGoTrace / debug_stopGoTrace / debug_goTrace
//   - debug_blockProfile / debug_writeBlockProfile
//   - debug_mutexProfile / debug_writeMutexProfile
//   - debug_writeMemProfile
//
// All of the above call os.Create(expandHome(file)) with a caller-controlled
// path, allowing arbitrary file creation anywhere the process can write.
type debugFileWriteBlocker struct{}

func (*debugFileWriteBlocker) StartCPUProfile(_ string) error      { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) StopCPUProfile() error               { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) CpuProfile(_ string, _ uint) error   { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) StartGoTrace(_ string) error         { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) StopGoTrace() error                  { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) GoTrace(_ string, _ uint) error      { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) BlockProfile(_ string, _ uint) error { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) WriteBlockProfile(_ string) error    { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) MutexProfile(_ string, _ uint) error { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) WriteMutexProfile(_ string) error    { return errDebugMethodDisabled }
func (*debugFileWriteBlocker) WriteMemProfile(_ string) error      { return errDebugMethodDisabled }
