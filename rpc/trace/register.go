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

// Package trace serves the trace namespace: the action traces of
// transactions, blocks and calls, OpenEthereum's tracing API.
package trace

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type handlers struct {
	dispatcher  *dispatch.Dispatcher
	environment *exec.Environment
}

// Register adds the methods of the namespace.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
	handler := &handlers{dispatcher: dispatcher, environment: environment}
	hash := dispatch.Required[common.Hash]("hash")
	dispatcher.Add("trace_transaction", handler.transaction, hash)
	dispatcher.Add("trace_get", handler.get, hash, dispatch.Required[[]hexutil.Uint]("indexes"))
	dispatcher.Add("trace_block", handler.block, dispatch.Required[chain.BlockReference]("block"))
	dispatcher.Add("trace_filter", handler.filter, dispatch.Required[filterArguments]("filter"))
}
