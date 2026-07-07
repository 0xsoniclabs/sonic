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

// Member describes a consensus validator as far as callers of this package need
// to: its consensus identity only. Network location is discovered internally by
// the validator directory and never supplied by callers.
type Member struct {
	// ID is the consensus identifier of the validator.
	ID uint32
	// PublicKey is the validator's consensus public key bytes (secp256k1), used
	// to authenticate its advertisements and binding proof.
	PublicKey []byte
}

// Membership is the consensus-provided set of current validators and the sole
// external input to the validator network: callers supply who the validators
// are; addresses, discovery, and authentication are handled inside the package.
// The set changes over time (e.g. at epoch boundaries), so updates are exposed.
type Membership interface {
	// Members returns the current set of validators.
	Members() []Member
	// Epoch returns the epoch the current set belongs to.
	Epoch() uint64
	// OnChange registers a callback invoked whenever the set changes. The
	// returned function cancels the subscription.
	OnChange(callback func()) (cancel func())
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
