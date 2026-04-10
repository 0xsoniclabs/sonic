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
	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/api/sccapi"
	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// Backend is an aggreate of all the APIs' backends.
// This is a compromise solution because of the monolithic nature of the
// original api backend object. This interface allows to develop and
// maintain the APIs independently of the backend implementation.
type backend interface {
	ethapi.Backend
	sonicapi.BundleApiBackend
	sccapi.SccApiBackend
}

func GetAPIs(apiBackend backend) []rpc.API {
	nonceLock := new(ethapi.AddrLocker)
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicEthereumAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "dag",
			Version:   "1.0",
			Service:   ethapi.NewPublicDAGChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicTransactionPoolAPI(apiBackend, nonceLock),
			Public:    true,
		}, {
			Namespace: "txpool",
			Version:   "1.0",
			Service:   ethapi.NewPublicTxPoolAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   ethapi.NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicAccountAPI(apiBackend.AccountManager()),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   ethapi.NewPrivateAccountAPI(apiBackend, nonceLock),
			Public:    false,
		}, {
			Namespace: "abft",
			Version:   "1.0",
			Service:   ethapi.NewPublicAbftAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "sonic",
			Version:   "1.0",
			Service:   sonicapi.NewPublicBundleAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "scc",
			Version:   "1.0",
			Service:   sccapi.NewPublicSccApi(apiBackend),
			Public:    true,
		},
	}
}
