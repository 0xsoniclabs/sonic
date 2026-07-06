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
// set, reconciling as the set changes over time. It does not itself perform the
// authentication handshake; that is the HandshakeProtocol, registered
// separately on the node.
type ValidatorMesh struct {
	host          MeshHost
	logger        logger.Logger
	mutex         sync.Mutex
	connected     map[peer.ID]Validator
	cancelUpdates func()
}

// NewValidatorMesh creates a validator mesh over the given host.
func NewValidatorMesh(host MeshHost) *ValidatorMesh {
	return &ValidatorMesh{
		host:      host,
		logger:    host.Logger(),
		connected: make(map[peer.ID]Validator),
	}
}

// Track performs an initial reconciliation against the set and subscribes to
// its updates so the mesh follows the validator set over time. Call Stop to
// unsubscribe.
func (m *ValidatorMesh) Track(ctx context.Context, set ValidatorSet) {
	m.Reconcile(ctx, set.Current())
	m.cancelUpdates = set.OnUpdate(func(validators []Validator) {
		m.Reconcile(ctx, validators)
	})
}

// Stop unsubscribes from validator-set updates.
func (m *ValidatorMesh) Stop() {
	if m.cancelUpdates != nil {
		m.cancelUpdates()
		m.cancelUpdates = nil
	}
}

// Reconcile brings the mesh in line with the desired validator set: it dials
// validators that are not yet connected and disconnects validators that have
// been removed from the set. Failed dials are retried on the next reconcile.
func (m *ValidatorMesh) Reconcile(ctx context.Context, validators []Validator) {
	self := m.host.ID()
	desired := make(map[peer.ID]Validator, len(validators))
	for _, validator := range validators {
		if validator.Peer.ID == self {
			continue
		}
		desired[validator.Peer.ID] = validator
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	added, removed := 0, 0
	next := make(map[peer.ID]Validator, len(desired))
	for id, validator := range m.connected {
		if _, ok := desired[id]; ok {
			next[id] = validator
		} else {
			_ = m.host.ClosePeer(id, "removed-from-set")
			removed++
		}
	}
	for id, validator := range desired {
		if _, ok := next[id]; ok {
			continue
		}
		if err := m.host.Connect(ctx, validator.Peer); err != nil {
			m.logger.Debug("validator dial failed", "peer", id, "err", err)
			continue
		}
		next[id] = validator
		added++
	}
	m.connected = next
	m.logger.Info("validator mesh reconciled",
		"validators", len(desired), "added", added, "removed", removed)
}

// HandshakeProtocol authenticates validators on the mesh. On an inbound stream
// it verifies the peer's binding proof against the current validator set; it
// also drives the outbound proof via Authenticate.
type HandshakeProtocol struct {
	self            peer.ID
	signer          Signer
	verifier        Verifier
	set             ValidatorSet
	validatorID     uint32
	logger          logger.Logger
	onAuthenticated func(peer.ID, uint32)
}

// NewHandshakeProtocol creates the handshake protocol. self is the local peer
// identity the outbound proof binds to. onAuthenticated, if set, is called with
// the peer ID and validator ID once a peer is authenticated.
func NewHandshakeProtocol(
	self peer.ID,
	signer Signer,
	verifier Verifier,
	set ValidatorSet,
	validatorID uint32,
	log logger.Logger,
	onAuthenticated func(peer.ID, uint32),
) *HandshakeProtocol {
	return &HandshakeProtocol{
		self:            self,
		signer:          signer,
		verifier:        verifier,
		set:             set,
		validatorID:     validatorID,
		logger:          log,
		onAuthenticated: onAuthenticated,
	}
}

// ProtocolID implements p2p.StreamProtocol.
func (h *HandshakeProtocol) ProtocolID() protocol.ID {
	return HandshakeProtocolID
}

// Handle verifies an inbound validator binding proof.
func (h *HandshakeProtocol) Handle(stream p2p.Stream) {
	var proof pb.ValidatorHandshake
	if err := stream.ReadMessage(&proof, maxHandshakeSize); err != nil {
		h.logger.Debug("handshake read failed", "peer", stream.Peer(), "err", err)
		_ = stream.Reset()
		return
	}
	if err := VerifyBindingProof(h.verifier, &proof, stream.Peer(), h.set.Epoch(), h.isValidator); err != nil {
		h.logger.Info("validator handshake rejected", "peer", stream.Peer(), "err", err)
		_ = stream.Reset()
		return
	}
	if h.onAuthenticated != nil {
		h.onAuthenticated(stream.Peer(), proof.ValidatorId)
	}
	h.logger.Debug("validator authenticated", "peer", stream.Peer(), "validator", proof.ValidatorId)
	_ = stream.Close()
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
	proof, err := CreateBindingProof(h.signer, h.self, h.validatorID, h.set.Epoch(), nonce)
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
	for _, validator := range h.set.Current() {
		if string(validator.PublicKey) == string(publicKey) {
			return true
		}
	}
	return false
}

// StreamOpener opens outbound streams; *p2p.Node satisfies it.
type StreamOpener interface {
	OpenStream(ctx context.Context, target p2p.PeerID, id protocol.ID) (p2p.Stream, error)
}
