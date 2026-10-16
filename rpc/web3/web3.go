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

// Package web3 implements the web3_ namespace.
package web3

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Register registers web3_clientVersion and web3_sha3. The version string is
// the node's name and version, supplied by the composition root.
func Register(d *dispatch.Dispatcher, clientVersion string) {
	dispatch.Add0(d, "web3_clientVersion", func(context.Context) (string, error) {
		return clientVersion, nil
	})
	dispatch.Add1(d, "web3_sha3", func(_ context.Context, input hexutil.Bytes) (hexutil.Bytes, error) {
		return crypto.Keccak256(input), nil
	}, "input")
}
