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

// Package web3 serves the web3 namespace.
package web3

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Register adds the methods of the namespace; clientVersion is the name the
// node gives itself on the p2p network.
func Register(dispatcher *dispatch.Dispatcher, clientVersion string) {
	dispatcher.Add("web3_clientVersion", func(context.Context, dispatch.Arguments) (any, error) {
		return clientVersion, nil
	})
	dispatcher.Add("web3_sha3", func(_ context.Context, arguments dispatch.Arguments) (any, error) {
		return hexutil.Bytes(crypto.Keccak256(dispatch.Argument[hexutil.Bytes](arguments, 0))), nil
	}, dispatch.Required[hexutil.Bytes]("input"))
}
