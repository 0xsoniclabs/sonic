// Copyright 2025 Sonic Operations Ltd
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

package topology

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// ConnectionAdvisor is a utility that provides suggestions on which peers to
// connect to and which peers to disconnect from based on the local node's
// neighborhood. The advisor is used to maintain a healthy peer set and to
// optimize the local node's connectivity as well as the overall network topology.
type ConnectionAdvisor interface {
	// GetNewPeerSuggestion returns a new peer that should be connected to.
	GetNewPeerSuggestion() *enode.Node

	// GetRedundantPeerSuggestion returns the ID of a peer that should be
	// disconnected in favor of another peer if needed.
	GetRedundantPeerSuggestion() *enode.ID

	// UpdatePeers updates information about the peers connected to a given peer.
	// This information is used to obtain an overview on the local node's
	// neighborhood from which decisions changes in the peer set can be made.
	// The provided peer is assumed to be a peer of the local node.
	UpdatePeers(peer enode.ID, peers []*enode.Node)
}

func NewConnectionAdvisor(localId enode.ID) ConnectionAdvisor {
	return newConnectionAdvisor[enode.ID, *enode.Node](
		localId, 60*time.Second, func(n *enode.Node) enode.ID { return n.ID() },
	)
}

func newConnectionAdvisor[I comparable, R any](
	localId I,
	maxPeerInfoAge time.Duration,
	getId func(R) I,
) *connectionAdvisor[I, R] {
	return &connectionAdvisor[I, R]{
		neighborhood:   make(map[I]neighborhoodEntry[R]),
		localId:        localId,
		getId:          getId,
		maxPeerInfoAge: maxPeerInfoAge,
	}
}

// connectionAdvisor is a ConnectionAdvisor implementation that suggests new peers
// to connect to and redundant peers to disconnect from based on a simple heuristic.
// The implementation is generic to simplify testing. The type parameter I is the
// type of the peer ID and R is the type of the peer reference required to establish
// a connection.
type connectionAdvisor[I comparable, R any] struct {
	mu sync.Mutex

	// Keep track of the neighbors of each peer.
	neighborhood map[I]neighborhoodEntry[R]

	// The ID of the local node.
	localId I

	// maxPeerInfoAge is the maximum age of peer information that is considered
	// when making suggestions on adding or removing peers. Older information
	// is discarded.
	maxPeerInfoAge time.Duration

	// getId returns the ID of a stored peer reference
	getId func(R) I
}

type neighborhoodEntry[T any] struct {
	peers []T
	time  time.Time
}

func (c *connectionAdvisor[I, T]) GetNewPeerSuggestion() T {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Search for a peer of a peer that is not already connected to the local node.
	now := time.Now()
	for peer, entry := range c.neighborhood {
		if now.Sub(entry.time) > c.maxPeerInfoAge {
			delete(c.neighborhood, peer)
			continue
		}
		for _, peer := range entry.peers {
			peerId := c.getId(peer)
			if peerId == c.localId {
				continue
			}
			if _, found := c.neighborhood[peerId]; !found {
				return peer
			}
		}
	}

	var zero T
	return zero
}

func (c *connectionAdvisor[I, T]) GetRedundantPeerSuggestion() *I {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Count the number of indirect connections to all peers and
	// recommend the one that has the most indirect connections.
	count := map[I]int{}
	now := time.Now()
	for peer, entry := range c.neighborhood {
		if now.Sub(entry.time) > c.maxPeerInfoAge {
			delete(c.neighborhood, peer)
			continue
		}
		for _, peer := range entry.peers {
			peerId := c.getId(peer)
			if _, found := c.neighborhood[peerId]; found {
				count[peerId]++
			}
		}
	}
	delete(count, c.localId)

	var maxCount int
	var maxPeer I
	for peer, c := range count {
		if c > maxCount {
			maxCount = c
			maxPeer = peer
		}
	}

	if maxCount == 0 {
		return nil
	}
	return &maxPeer
}

func (c *connectionAdvisor[I, T]) UpdatePeers(peer I, peers []T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.neighborhood[peer] = neighborhoodEntry[T]{
		peers: peers,
		time:  time.Now(),
	}
}
