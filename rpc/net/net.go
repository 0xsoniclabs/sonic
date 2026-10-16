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

// Package net implements the net_ namespace. Only net_version is a fact of the
// chain; the peer facts (net_listening, net_peerCount) belong to the p2p layer
// and are not served by the module.
package net

import (
	"context"
	"strconv"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Register registers net_version.
func Register(d *dispatch.Dispatcher, c core.ChainReader) {
	dispatch.Add0(d, "net_version", func(context.Context) (string, error) {
		return strconv.FormatUint(c.ChainInfo().NetworkID, 10), nil
	})
}
