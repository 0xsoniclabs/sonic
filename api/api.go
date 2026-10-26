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

// Package api lists the services of the node's own RPC handler: the legacy
// namespaces and the test namespace, served on IPC only. Everything else is
// served by the RPC module.
package api

import (
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/api/legacy"
	"github.com/0xsoniclabs/sonic/api/testapi"
)

// backend is what the services of the node's handler read of the node.
type backend interface {
	legacy.Backend
	testapi.Backend
}

// GetAPIs returns the services of the node's handler.
func GetAPIs(b backend) []rpc.API {
	return append(legacy.GetAPIs(b), rpc.API{
		Namespace: "test",
		Service:   testapi.NewTestApi(b),
	})
}
