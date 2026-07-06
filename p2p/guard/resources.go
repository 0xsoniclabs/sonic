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

package guard

import (
	"github.com/libp2p/go-libp2p/core/network"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// ResourceLimitConfig configures the aggregate resource caps enforced by the
// libp2p resource manager.
type ResourceLimitConfig struct {
	// MaxConnections is the maximum number of simultaneous connections.
	MaxConnections int
	// MaxConnectionsPerPeer is the maximum number of connections to one peer.
	MaxConnectionsPerPeer int
	// MaxStreamsPerPeer is the maximum number of concurrent streams to one peer.
	MaxStreamsPerPeer int
}

// NewResourceManager builds a libp2p resource manager whose system and per-peer
// limits are derived from config, layered over libp2p's auto-scaled defaults so
// that any limit left at zero keeps a sensible default.
func NewResourceManager(config ResourceLimitConfig) (network.ResourceManager, error) {
	defaults := rcmgr.DefaultLimits
	concrete := defaults.AutoScale()

	partial := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			Conns:        rcmgr.LimitVal(config.MaxConnections),
			ConnsInbound: rcmgr.LimitVal(config.MaxConnections),
		},
		PeerDefault: rcmgr.ResourceLimits{
			Conns:          rcmgr.LimitVal(config.MaxConnectionsPerPeer),
			ConnsInbound:   rcmgr.LimitVal(config.MaxConnectionsPerPeer),
			Streams:        rcmgr.LimitVal(config.MaxStreamsPerPeer),
			StreamsInbound: rcmgr.LimitVal(config.MaxStreamsPerPeer),
		},
	}
	return rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(partial.Build(concrete)))
}
