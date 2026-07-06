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

// Package networks implements the logical networks that run on top of the P2P
// node: the authenticated validator full-mesh, a reusable gossip-topic wrapper,
// and archive discovery. All node-specific data is injected through the
// interfaces defined here, so the networks are exercised in isolation with
// mocks; wiring the interfaces to real node state is a follow-up task (see
// HANDOFF.md).
package networks

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

//go:generate mockgen -source=sources.go -destination=sources_mock.go -package=networks

// Role identifies the role a node plays on the network.
type Role int

const (
	RoleUnspecified Role = iota
	RoleValidator
	RoleArchive
	RoleObserver
)

// Validator describes a consensus validator as far as the P2P layer needs it:
// its consensus identity and where to reach it on the network.
type Validator struct {
	// ID is the consensus identifier of the validator.
	ID uint32
	// PublicKey is the validator's consensus public key bytes (secp256k1),
	// used to authenticate its binding proof.
	PublicKey []byte
	// Peer is the network location (peer ID and addresses) of the validator.
	Peer peer.AddrInfo
}

// ValidatorSet is the source of the current validator set. Because the set
// changes over time (e.g. at epoch boundaries), it also exposes updates so the
// validator mesh can reconcile its connections.
type ValidatorSet interface {
	// Current returns the current set of validators.
	Current() []Validator
	// Epoch returns the epoch the current set belongs to.
	Epoch() uint64
	// OnUpdate registers a callback invoked with the new set whenever it
	// changes. The returned function cancels the subscription.
	OnUpdate(callback func([]Validator)) (cancel func())
}

// NodeStatus is the self-reported status of a node, returned to a network scan.
type NodeStatus struct {
	Role          Role
	ClientVersion string
	BlockHeight   uint64
}

// NodeStatusSource provides this node's own status for the scan protocol.
type NodeStatusSource interface {
	// Status returns the node's current role, client version, and synced height.
	Status() NodeStatus
}

// PeerSource provides the peers this node knows about, so a scan can crawl the
// network graph.
type PeerSource interface {
	// Peers returns the currently known peers with their addresses.
	Peers() []peer.AddrInfo
}
