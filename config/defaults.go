// Copyright 2025 Sonic Operations Ltd
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

package config

import (
	"fmt"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	DefaultHTTPPort = 18545 // Default TCP port for the HTTP RPC server
	DefaultWSPort   = 18546 // Default TCP port for the websocket RPC server
)

// NodeDefaultConfig contains reasonable default settings.
var NodeDefaultConfig = node.Config{
	HTTPPort:            DefaultHTTPPort,
	HTTPModules:         []string{},
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSPort:              DefaultWSPort,
	WSModules:           []string{},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		NoDiscovery: false, // enable discovery by default
		DiscoveryV4: false, // disable discovery v4 by default
		DiscoveryV5: true,  // enable discovery v5 by default
		ListenAddr:  fmt.Sprintf(":%d", flags.ListenPortFlag.Value),
		MaxPeers:    50,
		NAT:         nat.Any(),
	},
}
