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

// Package protocols contains concrete protocols built on the P2P layer. It
// currently provides the network-scan protocol, the worked example of the
// open/closed registry: it is a self-contained p2p.StreamProtocol whose only
// couplings to the rest of the node are the injected NodeStatusSource and
// PeerSource interfaces.
package protocols

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// ScanProtocolID is the libp2p protocol the network scan runs on.
const ScanProtocolID = protocol.ID("/sonic/scan/1")

// Per-message-type size caps. Requests are tiny; the peers response can be
// larger. Distinct caps demonstrate per-message-type limiting.
const (
	maxScanRequestSize = 256
	maxScanStatusSize  = 1 << 10
	maxScanPeersSize   = 1 << 20
)

// ScanProtocol answers network-scan queries about this node: its role, client
// version, synced block height, and the peers it knows. It is registered on the
// node like any other StreamProtocol.
type ScanProtocol struct {
	status networks.NodeStatusSource
	peers  networks.PeerSource
}

// NewScanProtocol creates the serving side of the network-scan protocol.
func NewScanProtocol(status networks.NodeStatusSource, peers networks.PeerSource) *ScanProtocol {
	return &ScanProtocol{status: status, peers: peers}
}

// ProtocolID implements p2p.StreamProtocol.
func (s *ScanProtocol) ProtocolID() protocol.ID { return ScanProtocolID }

// Handle serves one scan exchange: a status request/response followed by a
// peers request/response.
func (s *ScanProtocol) Handle(stream p2p.Stream) {
	defer func() { _ = stream.Close() }()

	var statusRequest pb.ScanStatusRequest
	if err := stream.ReadMessage(&statusRequest, maxScanRequestSize); err != nil {
		_ = stream.Reset()
		return
	}
	status := s.status.Status()
	response := &pb.ScanStatusResponse{
		Role:          roleToProto(status.Role),
		ClientVersion: status.ClientVersion,
		BlockHeight:   status.BlockHeight,
	}
	if err := stream.WriteMessage(response, maxScanStatusSize); err != nil {
		return
	}

	var peersRequest pb.ScanPeersRequest
	if err := stream.ReadMessage(&peersRequest, maxScanRequestSize); err != nil {
		_ = stream.Reset()
		return
	}
	if err := stream.WriteMessage(&pb.ScanPeersResponse{PeerAddresses: s.peerAddresses()}, maxScanPeersSize); err != nil {
		return
	}
}

func (s *ScanProtocol) peerAddresses() []string {
	var addresses []string
	for _, info := range s.peers.Peers() {
		p2pAddrs, err := peer.AddrInfoToP2pAddrs(&info)
		if err != nil {
			continue
		}
		for _, multiaddr := range p2pAddrs {
			addresses = append(addresses, multiaddr.String())
		}
	}
	return addresses
}

func roleToProto(role networks.Role) pb.NodeRole {
	switch role {
	case networks.RoleValidator:
		return pb.NodeRole_NODE_ROLE_VALIDATOR
	case networks.RoleArchive:
		return pb.NodeRole_NODE_ROLE_ARCHIVE
	case networks.RoleObserver:
		return pb.NodeRole_NODE_ROLE_OBSERVER
	default:
		return pb.NodeRole_NODE_ROLE_UNSPECIFIED
	}
}

func roleFromProto(role pb.NodeRole) networks.Role {
	switch role {
	case pb.NodeRole_NODE_ROLE_VALIDATOR:
		return networks.RoleValidator
	case pb.NodeRole_NODE_ROLE_ARCHIVE:
		return networks.RoleArchive
	case pb.NodeRole_NODE_ROLE_OBSERVER:
		return networks.RoleObserver
	default:
		return networks.RoleUnspecified
	}
}
