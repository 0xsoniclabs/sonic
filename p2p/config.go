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

package p2p

import "github.com/0xsoniclabs/sonic/p2p/guard"

// Config holds the tunables for the P2P layer. It is a plain, TOML-friendly
// struct; wiring it into the node-wide configuration is a follow-up task (see
// HANDOFF.md).
type Config struct {
	// ListenAddresses are the multiaddrs the host binds to. QUIC is preferred;
	// a TCP address may be added for environments where QUIC is blocked.
	ListenAddresses []string

	// BootstrapPeers are multiaddrs (including the peer ID) of seed nodes used
	// to join the network and to seed gossipsub-based discovery.
	BootstrapPeers []string

	// HostKeyPath, when non-empty, is the file the libp2p host key is persisted
	// to and loaded from, yielding a stable peer ID across restarts (intended
	// for archives). When empty, an ephemeral in-memory key is generated on
	// every start.
	HostKeyPath string

	// RateLimit bounds the traffic a single peer may cause.
	RateLimit guard.RateLimitConfig

	// Resources bounds aggregate resource usage (connections, streams, memory).
	Resources guard.ResourceLimitConfig

	// ConnectionManager bounds the number of maintained connections.
	ConnectionManager ConnectionManagerConfig
}

// ConnectionManagerConfig configures the libp2p connection manager watermarks.
type ConnectionManagerConfig struct {
	// LowWater is the number of connections below which trimming stops.
	LowWater int
	// HighWater is the number of connections above which trimming starts.
	HighWater int
}

// DefaultConfig returns a Config with conservative, production-oriented
// defaults suitable for a permissionless network.
func DefaultConfig() Config {
	return Config{
		ListenAddresses: []string{
			"/ip4/0.0.0.0/udp/4002/quic-v1",
			"/ip4/0.0.0.0/tcp/4002",
		},
		BootstrapPeers: nil,
		HostKeyPath:    "",
		RateLimit: guard.RateLimitConfig{
			BytesPerSecond:    5 << 20, // 5 MiB/s
			ByteBurst:         10 << 20,
			MessagesPerSecond: 1000,
			MessageBurst:      2000,
		},
		Resources: guard.ResourceLimitConfig{
			MaxConnections:        400,
			MaxConnectionsPerPeer: 4,
			MaxStreamsPerPeer:     64,
		},
		ConnectionManager: ConnectionManagerConfig{
			LowWater:  200,
			HighWater: 400,
		},
	}
}
