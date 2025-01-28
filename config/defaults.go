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
		DiscoveryV4: false,  // disable discovery v4 by default
		DiscoveryV5: true,  // enable discovery v5 by default
		ListenAddr:  fmt.Sprintf(":%d", flags.ListenPortFlag.Value),
		MaxPeers:    50,
		NAT:         nat.Any(),
	},
}
