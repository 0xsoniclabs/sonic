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

// Register adds the web3 methods; clientVersion answers the version string
// of the node.
func Register(d *dispatch.Dispatcher, clientVersion string) {
	d.Add("web3_clientVersion", dispatch.Func(func(context.Context, dispatch.None) (any, error) {
		return clientVersion, nil
	}))
	d.Add("web3_sha3", dispatch.Func(sha3))
}

type sha3Args struct {
	Input hexutil.Bytes `rpc:"input"`
}

// sha3 answers the Keccak-256 hash of the input.
func sha3(_ context.Context, p sha3Args) (any, error) {
	return hexutil.Bytes(crypto.Keccak256(p.Input)), nil
}
