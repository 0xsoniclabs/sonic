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

package api

import (
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/api/legacy"
	"github.com/0xsoniclabs/sonic/api/testapi"
)

// backend is what the legacy namespaces need of the node.
type backend interface {
	legacy.Backend
	testapi.Backend
}

// GetAPIs returns the legacy namespaces the node's own handler serves (X3):
// the RPC module serves everything else.
func GetAPIs(apiBackend backend, defaults legacy.Defaults) []rpc.API {
	nonceLock := new(legacy.AddrLocker)
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   legacy.NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "dag",
			Version:   "1.0",
			Service:   legacy.NewPublicDAGChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   legacy.NewPublicTransactionPoolAPI(apiBackend, nonceLock, defaults),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   legacy.NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   legacy.NewPublicAccountAPI(apiBackend.AccountManager()),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   legacy.NewPrivateAccountAPI(apiBackend, nonceLock, defaults),
			Public:    false,
		}, {
			Namespace: "abft",
			Version:   "1.0",
			Service:   legacy.NewPublicAbftAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "test",
			Version:   "1.0",
			Service:   testapi.NewTestApi(apiBackend),
			Public:    true,
		},
	}
}
