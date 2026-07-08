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
	"crypto/rand"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// HandshakeProtocolID is the libp2p protocol on which validators exchange
// binding proofs.
const HandshakeProtocolID = protocol.ID("/sonic/validator-handshake/1")

// maxHandshakeSize bounds a ValidatorHandshake message.
const maxHandshakeSize = 4 << 10

// MeshHost is the subset of the P2P node the validator mesh needs. *p2p.Node
// satisfies it; tests use a mock.
type MeshHost interface {
	// ID returns the local peer identity.
	ID() p2p.PeerID
	// Connect dials a peer.
	Connect(ctx context.Context, info peer.AddrInfo) error
	// ClosePeer closes connections to a peer, recording a reason.
	ClosePeer(target p2p.PeerID, reason string) error
	// Logger returns the shared logger.
	Logger() logger.Logger
}

// ValidatorMesh maintains a full mesh of connections to the current validator
// set. Members are supplied by consensus (identity only); their network
// addresses are resolved internally via the AddressResolver, so the mesh dials a
// validator as soon as its address is discovered and re-dials as the set or the
// known addresses change. Authentication is performed by the HandshakeProtocol.
type ValidatorMesh struct {
	host      MeshHost
	resolver  AddressResolver
	logger    logger.Logger
	afterDial func(ctx context.Context, peerID peer.ID)

	mutex          sync.Mutex
	connected      map[peer.ID]peer.AddrInfo
	cancelMembers  func()
	cancelDiscover func()
}

// NewValidatorMesh creates a validator mesh over the given host, resolving
// validator addresses via resolver. afterDial, if non-nil, is invoked (in its
// own goroutine) after a validator is successfully dialed, so the caller can
// drive the outbound authentication handshake.
func NewValidatorMesh(host MeshHost, resolver AddressResolver, afterDial func(context.Context, peer.ID)) *ValidatorMesh {
	return &ValidatorMesh{
		host:      host,
		resolver:  resolver,
		logger:    host.Logger(),
		afterDial: afterDial,
		connected: make(map[peer.ID]peer.AddrInfo),
	}
}

// Track reconciles against the current membership now and re-reconciles whenever
// the membership changes or a new address is discovered. Call Stop to
// unsubscribe.
func (m *ValidatorMesh) Track(ctx context.Context, membership Membership) {
	m.Reconcile(ctx, membership.Members())
	m.cancelMembers = membership.OnChange(func() {
		m.Reconcile(ctx, membership.Members())
	})
	m.cancelDiscover = m.resolver.OnDiscovery(func() {
		m.Reconcile(ctx, membership.Members())
	})
}

// Stop unsubscribes from membership and discovery updates.
func (m *ValidatorMesh) Stop() {
	if m.cancelMembers != nil {
		m.cancelMembers()
		m.cancelMembers = nil
	}
	if m.cancelDiscover != nil {
		m.cancelDiscover()
		m.cancelDiscover = nil
	}
}

// Reconcile brings the mesh in line with the desired members whose addresses are
// known: it dials validators not yet connected and disconnects validators no
// longer in the set. Members without a discovered address are skipped and picked
// up on a later reconcile (e.g. when their address arrives). Failed dials are
// retried on the next reconcile.
func (m *ValidatorMesh) Reconcile(ctx context.Context, members []Member) {
	self := m.host.ID()
	desired := make(map[peer.ID]peer.AddrInfo, len(members))
	for _, member := range members {
		info, ok := m.resolver.Resolve(member.PublicKey)
		if !ok || info.ID == self {
			continue
		}
		desired[info.ID] = info
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	added, removed := 0, 0
	next := make(map[peer.ID]peer.AddrInfo, len(desired))
	for id, info := range m.connected {
		if _, ok := desired[id]; ok {
			next[id] = info
		} else {
			_ = m.host.ClosePeer(id, "removed-from-set")
			removed++
		}
	}
	for id, info := range desired {
		if _, ok := next[id]; ok {
			continue
		}
		if err := m.host.Connect(ctx, info); err != nil {
			m.logger.Debug("validator dial failed", "peer", id, "err", err)
			continue
		}
		next[id] = info
		added++
		if m.afterDial != nil {
			go m.afterDial(ctx, id)
		}
	}
	m.connected = next
	m.logger.Info("validator mesh reconciled",
		"reachable", len(desired), "added", added, "removed", removed)
}

// HandshakeProtocol authenticates validators on the mesh. On an inbound stream
// it verifies the peer's binding proof against the current membership; it also
// drives the outbound proof via Authenticate.
type HandshakeProtocol struct {
	self            peer.ID
	signer          Signer
	verifier        Verifier
	membership      Membership
	validatorID     uint32
	logger          logger.Logger
	onAuthenticated func(peer.ID, uint32)
	onFailure       func(peer.ID, error)
}

// NewHandshakeProtocol creates the handshake protocol. self is the local peer
// identity the outbound proof binds to. onAuthenticated, if set, is called with
// the peer ID and validator ID once a peer is authenticated; onFailure, if set,
// is called with the peer ID and error when an inbound handshake fails, so the
// caller can disconnect (and, on floods, ban) the peer.
func NewHandshakeProtocol(
	self peer.ID,
	signer Signer,
	verifier Verifier,
	membership Membership,
	validatorID uint32,
	log logger.Logger,
	onAuthenticated func(peer.ID, uint32),
	onFailure func(peer.ID, error),
) *HandshakeProtocol {
	return &HandshakeProtocol{
		self:            self,
		signer:          signer,
		verifier:        verifier,
		membership:      membership,
		validatorID:     validatorID,
		logger:          log,
		onAuthenticated: onAuthenticated,
		onFailure:       onFailure,
	}
}

// ProtocolID implements p2p.StreamProtocol.
func (h *HandshakeProtocol) ProtocolID() protocol.ID {
	return HandshakeProtocolID
}

// Handle verifies an inbound validator binding proof. On any failure it resets
// the stream and reports it via onFailure, so the caller can disconnect the peer
// (and ban it if failures become sustained).
func (h *HandshakeProtocol) Handle(stream p2p.Stream) {
	var proof pb.ValidatorHandshake
	if err := stream.ReadMessage(&proof, maxHandshakeSize); err != nil {
		h.logger.Debug("handshake read failed", "peer", stream.Peer(), "err", err)
		h.fail(stream, err)
		return
	}
	if err := VerifyBindingProof(h.verifier, &proof, stream.Peer(), h.membership.Epoch(), h.isValidator); err != nil {
		h.logger.Info("validator handshake rejected", "peer", stream.Peer(), "err", err)
		h.fail(stream, err)
		return
	}
	if h.onAuthenticated != nil {
		h.onAuthenticated(stream.Peer(), proof.ValidatorId)
	}
	h.logger.Debug("validator authenticated", "peer", stream.Peer(), "validator", proof.ValidatorId)
	_ = stream.Close()
}

func (h *HandshakeProtocol) fail(stream p2p.Stream, err error) {
	_ = stream.Reset()
	if h.onFailure != nil {
		h.onFailure(stream.Peer(), err)
	}
}

// Authenticate opens a handshake stream to target and sends this node's binding
// proof for the current epoch.
func (h *HandshakeProtocol) Authenticate(ctx context.Context, opener StreamOpener, target p2p.PeerID) error {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	// The proof binds to our own peer identity, which the receiver checks
	// against the connection's remote peer.
	proof, err := CreateBindingProof(h.signer, h.self, h.validatorID, h.membership.Epoch(), nonce)
	if err != nil {
		return err
	}
	stream, err := opener.OpenStream(ctx, target, HandshakeProtocolID)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()
	return stream.WriteMessage(proof, maxHandshakeSize)
}

func (h *HandshakeProtocol) isValidator(publicKey []byte) bool {
	for _, member := range h.membership.Members() {
		if string(member.PublicKey) == string(publicKey) {
			return true
		}
	}
	return false
}

// StreamOpener opens outbound streams; *p2p.Node satisfies it.
type StreamOpener interface {
	OpenStream(ctx context.Context, target p2p.PeerID, id protocol.ID) (p2p.Stream, error)
}
