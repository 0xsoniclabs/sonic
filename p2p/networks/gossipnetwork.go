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

import "github.com/0xsoniclabs/sonic/p2p"

// GossipNetwork is a reusable p2p.GossipTopic assembled from a validate and a
// deliver callback. It is the substrate for concrete pub/sub protocols such as
// finalized-block dissemination and the archive directory, turning a pair of
// functions into a registrable topic without boilerplate.
type GossipNetwork struct {
	topic    string
	validate func(from p2p.PeerID, message []byte) p2p.ValidationResult
	deliver  func(from p2p.PeerID, message []byte)
}

// NewGossipNetwork builds a gossip topic. validate is the anti-spam gate run
// before propagation; deliver handles messages that pass validation.
func NewGossipNetwork(
	topic string,
	validate func(from p2p.PeerID, message []byte) p2p.ValidationResult,
	deliver func(from p2p.PeerID, message []byte),
) *GossipNetwork {
	return &GossipNetwork{topic: topic, validate: validate, deliver: deliver}
}

// Topic implements p2p.GossipTopic.
func (g *GossipNetwork) Topic() string { return g.topic }

// Validate implements p2p.GossipTopic.
func (g *GossipNetwork) Validate(from p2p.PeerID, message []byte) p2p.ValidationResult {
	if g.validate == nil {
		return p2p.ValidationAccept
	}
	return g.validate(from, message)
}

// Deliver implements p2p.GossipTopic.
func (g *GossipNetwork) Deliver(from p2p.PeerID, message []byte) {
	if g.deliver != nil {
		g.deliver(from, message)
	}
}
