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

package legacy

import "github.com/ethereum/go-ethereum/rpc"

// GetAPIs returns the services of the legacy namespaces, which the node's
// handler serves on IPC only.
func GetAPIs(b Backend) []rpc.API {
	nonceLock := new(AddrLocker)
	return []rpc.API{
		{Namespace: "eth", Service: NewPublicBlockChainAPI(b)},
		{Namespace: "eth", Service: NewPublicTransactionPoolAPI(b, nonceLock)},
		{Namespace: "eth", Service: NewPublicAccountAPI(b.AccountManager())},
		{Namespace: "personal", Service: NewPrivateAccountAPI(b, nonceLock)},
		{Namespace: "debug", Service: NewPrivateDebugAPI(b)},
		{Namespace: "dag", Service: NewPublicDAGChainAPI(b)},
		{Namespace: "abft", Service: NewPublicAbftAPI(b)},
	}
}
