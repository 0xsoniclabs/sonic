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
	"sync"

	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Gater is a libp2p connmgr.ConnectionGater that maintains a ban list of peers.
// Banned peers are rejected as early as possible in the connection lifecycle,
// both for outbound dials and inbound connections. It is safe for concurrent
// use.
type Gater struct {
	mutex  sync.RWMutex
	banned map[peer.ID]struct{}
}

// NewGater creates an empty Gater that permits all peers until they are banned.
func NewGater() *Gater {
	return &Gater{banned: make(map[peer.ID]struct{})}
}

// Ban adds a peer to the ban list. Existing connections are not closed by the
// gater itself; the caller closes them.
func (g *Gater) Ban(p peer.ID) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.banned[p] = struct{}{}
}

// Unban removes a peer from the ban list.
func (g *Gater) Unban(p peer.ID) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	delete(g.banned, p)
}

// IsBanned reports whether the peer is currently banned.
func (g *Gater) IsBanned(p peer.ID) bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	_, ok := g.banned[p]
	return ok
}

// InterceptPeerDial rejects outbound dials to banned peers.
func (g *Gater) InterceptPeerDial(p peer.ID) bool {
	return !g.IsBanned(p)
}

// InterceptAddrDial rejects outbound dials to banned peers.
func (g *Gater) InterceptAddrDial(p peer.ID, _ ma.Multiaddr) bool {
	return !g.IsBanned(p)
}

// InterceptAccept permits inbound connection attempts; the peer identity is not
// yet known at this stage, so gating happens in InterceptSecured.
func (g *Gater) InterceptAccept(_ network.ConnMultiaddrs) bool {
	return true
}

// InterceptSecured rejects secured connections with banned peers, in either
// direction, once the remote identity is known.
func (g *Gater) InterceptSecured(_ network.Direction, p peer.ID, _ network.ConnMultiaddrs) bool {
	return !g.IsBanned(p)
}

// InterceptUpgraded permits fully upgraded connections; banning is enforced in
// the earlier stages.
func (g *Gater) InterceptUpgraded(_ network.Conn) (bool, control.DisconnectReason) {
	return true, 0
}
