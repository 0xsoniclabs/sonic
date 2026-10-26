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

// Package sonic serves the sonic namespace: the gas of transactions that
// depend on each other, and bundles.
package sonic

import (
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
	dispatcher.Add("sonic_estimateGasForTransactions", handler.estimateGasForTransactions,
		dispatch.Required[[]exec.CallArguments]("transactions"),
		dispatch.Optional[chain.BlockReference]("block"),
		dispatch.Optional[exec.StateOverride]("stateOverrides"),
		dispatch.Optional[exec.BlockOverrides]("blockOverrides"))
}
