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

package main

import (
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
)

// clientVersion is the version string every demo node reports in its status.
const clientVersion = "sonic-p2p-demo/v0.1.0"

// blockTime is how fast the faked chain advances: one block per second.
const blockTime = time.Second

// maxHeightDrift bounds the per-node block-height offset. A small drift makes the
// nodes report slightly different heights so the health report has something to
// show, while still looking like a single shared chain.
const maxHeightDrift = 4

// genesisEpoch is the shared reference time from which the faked chain height is
// derived. It is a fixed literal so every node computes the same height from the
// same wall-clock instant.
var genesisEpoch = time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

// demoStatus is the faked NodeStatusSource for a demo node. The block height is a
// shared ~1 block/sec clock plus a small deterministic per-node drift.
type demoStatus struct {
	role         networks.Role
	heightOffset uint64
	now          func() time.Time
}

// newDemoStatus builds the status source for a node in the given role, deriving
// its height drift deterministically from seed (the validator ID or peer ID).
func newDemoStatus(role networks.Role, seed string) demoStatus {
	return demoStatus{
		role:         role,
		heightOffset: heightDrift(seed),
		now:          time.Now,
	}
}

// Status implements networks.NodeStatusSource.
func (s demoStatus) Status() networks.NodeStatus {
	return networks.NodeStatus{
		Role:          s.role,
		ClientVersion: clientVersion,
		BlockHeight:   s.blockHeight(),
	}
}

func (s demoStatus) blockHeight() uint64 {
	elapsed := s.now().Sub(genesisEpoch)
	if elapsed < 0 {
		elapsed = 0
	}
	return uint64(elapsed/blockTime) + s.heightOffset
}

// heightDrift maps a seed to a small, stable per-node height offset.
func heightDrift(seed string) uint64 {
	return uint64(crypto.Keccak256([]byte(seed))[0]) % maxHeightDrift
}

// nodePeerSource implements networks.PeerSource over a running node, so the scan
// protocol can report the peers this node currently knows.
type nodePeerSource struct {
	node *p2p.Node
}

// Peers implements networks.PeerSource.
func (s nodePeerSource) Peers() []peer.AddrInfo {
	connected := s.node.ConnectedPeers()
	store := s.node.Host().Peerstore()
	infos := make([]peer.AddrInfo, 0, len(connected))
	for _, id := range connected {
		infos = append(infos, store.PeerInfo(id))
	}
	return infos
}
