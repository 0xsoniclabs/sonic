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

// Package debug serves the debug namespace: the traces of transactions,
// blocks and calls, and blocks as stored.
package debug

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/tracers"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type handlers struct {
	environment *exec.Environment
}

// Register adds the methods of the namespace.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
	handler := &handlers{environment: environment}
	config := dispatch.Optional[tracers.TraceConfig]("config")
	number := dispatch.Required[uint64]("number")
	dispatcher.Add("debug_getBlockRlp", handler.getBlockRlp, number)
	dispatcher.Add("debug_printBlock", handler.printBlock, number)
	dispatcher.Add("debug_traceTransaction", handler.traceTransaction, dispatch.Required[common.Hash]("hash"), config)
	dispatcher.Add("debug_traceBlockByNumber", handler.traceBlockByNumber, dispatch.Required[chain.BlockNumber]("number"), config)
	dispatcher.Add("debug_traceBlockByHash", handler.traceBlockByHash, dispatch.Required[common.Hash]("hash"), config)
	dispatcher.Add("debug_traceCall", handler.traceCall,
		dispatch.Required[exec.CallArguments]("transaction"),
		dispatch.Required[chain.BlockReference]("block"),
		dispatch.Optional[exec.TraceCallConfig]("config"))
	registerProfiling(dispatcher)
}
