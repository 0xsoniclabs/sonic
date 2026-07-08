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

package networks

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/guard"
)

// ValidatorNode is the subset of *p2p.Node the validator network needs. Keeping
// it an interface decouples this package from the concrete node for testing;
// *p2p.Node satisfies it.
type ValidatorNode interface {
	ID() p2p.PeerID
	Addresses() []ma.Multiaddr
	Connect(ctx context.Context, info peer.AddrInfo) error
	ClosePeer(target p2p.PeerID, reason string) error
	DisconnectAndBan(target p2p.PeerID, reason string, banDuration time.Duration)
	OpenStream(ctx context.Context, target p2p.PeerID, id protocol.ID) (p2p.Stream, error)
	Publish(ctx context.Context, topic string, message []byte) error
	RegisterGossipTopic(topic p2p.GossipTopic)
	RegisterStreamProtocol(streamProtocol p2p.StreamProtocol)
	Logger() logger.Logger
}

// ValidatorNetworkConfig tunes the validator network: address advertisement and
// the tolerance for failed authentication handshakes before a peer is banned.
type ValidatorNetworkConfig struct {
	// Directory tunes address advertisement/discovery.
	Directory ValidatorDirectoryConfig
	// HandshakeFailures bounds how many handshake failures a peer may cause
	// before it is banned (a short burst is tolerated for epoch-boundary skew).
	HandshakeFailures guard.FailureLimitConfig
	// HandshakeBanDuration is how long a sustained handshake-flooder is banned.
	HandshakeBanDuration time.Duration
}

func (c ValidatorNetworkConfig) withDefaults() ValidatorNetworkConfig {
	if c.HandshakeFailures.FailureBurst <= 0 {
		c.HandshakeFailures.FailureBurst = 3
	}
	if c.HandshakeFailures.FailuresPerSecond <= 0 {
		c.HandshakeFailures.FailuresPerSecond = 0.2
	}
	if c.HandshakeBanDuration <= 0 {
		c.HandshakeBanDuration = 60 * time.Second
	}
	return c
}

// ValidatorNetwork composes the validator address directory, the full mesh, and
// the authentication handshake into a single unit. Consensus supplies only the
// membership (identity); discovery, dialing, and authentication are handled
// internally. Construct it before starting the node (it registers a gossip topic
// and a stream protocol), then call Start after the node has started.
type ValidatorNetwork struct {
	node       ValidatorNode
	membership Membership
	directory  *ValidatorDirectory
	mesh       *ValidatorMesh
	handshake  *HandshakeProtocol
}

// NewValidatorNetwork wires the directory, mesh, and handshake over node and
// registers the directory (gossip) and handshake (stream) protocols on it. The
// node must not have been started yet.
func NewValidatorNetwork(
	node ValidatorNode,
	membership Membership,
	signer Signer,
	verifier Verifier,
	validatorID uint32,
	config ValidatorNetworkConfig,
) *ValidatorNetwork {
	config = config.withDefaults()
	log := node.Logger()

	directory := NewValidatorDirectory(
		membership, signer, verifier, node, node, log, config.Directory, uint64(time.Now().UnixNano()),
	)

	// A peer that fails the handshake is disconnected; a peer whose failures
	// become sustained is also banned for a cooldown. A successful handshake
	// clears the peer's failure history.
	failures := guard.NewFailureLimiter(config.HandshakeFailures)
	onAuthenticated := func(peerID peer.ID, _ uint32) {
		failures.Forget(peerID.String())
	}
	onFailure := func(peerID peer.ID, _ error) {
		banFor := time.Duration(0)
		if failures.Record(peerID.String()) {
			banFor = config.HandshakeBanDuration
		}
		node.DisconnectAndBan(peerID, "handshake-failure", banFor)
	}

	handshake := NewHandshakeProtocol(node.ID(), signer, verifier, membership, validatorID, log, onAuthenticated, onFailure)
	mesh := NewValidatorMesh(node, directory, func(ctx context.Context, peerID peer.ID) {
		if err := handshake.Authenticate(ctx, node, peerID); err != nil {
			log.Debug("validator authentication failed", "peer", peerID, "err", err)
		}
	})

	node.RegisterGossipTopic(directory)
	node.RegisterStreamProtocol(handshake)

	return &ValidatorNetwork{
		node:       node,
		membership: membership,
		directory:  directory,
		mesh:       mesh,
		handshake:  handshake,
	}
}

// Start begins advertising this node's address and maintaining the mesh. Call it
// after the node has started.
func (n *ValidatorNetwork) Start(ctx context.Context) {
	n.directory.Start(ctx)
	n.mesh.Track(ctx, n.membership)
}

// Stop tears the mesh and directory down.
func (n *ValidatorNetwork) Stop() {
	n.mesh.Stop()
	n.directory.Stop()
}
