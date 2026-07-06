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

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/protobuf/proto"
)

// PeerID identifies a peer on the network. It is the libp2p peer identifier,
// re-exported so callers need not import libp2p directly.
type PeerID = peer.ID

// Stream is a bidirectional, protobuf-framed, rate-limited connection to a
// single peer, handed to a StreamProtocol. Reads are subject to the per-peer
// traffic limits; each read and write supplies its own size cap, so limits can
// differ per message type.
type Stream interface {
	// Peer returns the identity of the remote peer.
	Peer() PeerID
	// ReadMessage reads one framed message into m, rejecting frames larger than
	// maxSize before their body is read. It returns ErrRateLimited if the peer
	// has exceeded its traffic budget.
	ReadMessage(m proto.Message, maxSize int) error
	// WriteMessage writes m as a single framed message, rejecting it if it
	// exceeds maxSize.
	WriteMessage(m proto.Message, maxSize int) error
	// Close closes the stream for writing and signals a clean end of messages.
	Close() error
	// Reset abruptly discards the stream, signalling an error to the peer.
	Reset() error
}

// StreamProtocol is a request/response or streaming protocol bound to a libp2p
// protocol ID. Registering one is the primary open/closed extension point: new
// protocols are added by implementing this interface, never by editing the core.
type StreamProtocol interface {
	// ProtocolID is the libp2p protocol identifier the protocol listens on.
	ProtocolID() protocol.ID
	// Handle serves a single inbound stream. It owns the stream's lifecycle and
	// must Close or Reset it before returning.
	Handle(stream Stream)
}

// ValidationResult is the outcome of validating a received gossip message.
type ValidationResult int

const (
	// ValidationAccept forwards the message and delivers it to the application.
	ValidationAccept ValidationResult = iota
	// ValidationReject drops the message and penalises the sender's score.
	ValidationReject
	// ValidationIgnore drops the message without penalising the sender.
	ValidationIgnore
)

// GossipTopic is a publish/subscribe protocol over gossipsub. Like
// StreamProtocol, it is registered on the node and is a pure extension point.
type GossipTopic interface {
	// Topic is the gossipsub topic name.
	Topic() string
	// Validate is invoked before a message is forwarded; a non-accept result
	// stops propagation and is the anti-spam gate for the topic.
	Validate(from PeerID, message []byte) ValidationResult
	// Deliver handles a message that passed validation.
	Deliver(from PeerID, message []byte)
}
