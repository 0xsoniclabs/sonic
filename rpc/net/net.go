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
// Package net serves net_version; the p2p facts net_listening and
// net_peerCount stay with the node.
package net

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// Register adds net_version, the network id in decimal.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	d.Add("net_version", dispatch.Func(func(context.Context, dispatch.None) (any, error) {
		return fmt.Sprintf("%d", env.ChainInfo().NetworkID), nil
	}))
}
