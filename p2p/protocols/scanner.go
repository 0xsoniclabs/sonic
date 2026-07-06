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

package protocols

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// ScanHost is the subset of the P2P node the scanner needs to reach peers.
// *p2p.Node satisfies it; tests use a loopback pair or a mock.
type ScanHost interface {
	// Connect dials a peer so a stream can be opened to it.
	Connect(ctx context.Context, info peer.AddrInfo) error
	// OpenStream opens a framed stream to a peer on a protocol.
	OpenStream(ctx context.Context, target p2p.PeerID, id protocol.ID) (p2p.Stream, error)
}

// NodeReport is a single node's response to a scan query.
type NodeReport struct {
	Peer          peer.ID
	Role          networks.Role
	ClientVersion string
	BlockHeight   uint64
	Peers         []peer.AddrInfo
}

// NetworkReport aggregates the results of a network scan.
type NetworkReport struct {
	NodeCount       int
	ClientVersions  map[string]int
	HeightHistogram map[uint64]int
	RoleCounts      map[networks.Role]int
}

// Scanner crawls the network breadth-first, querying each reachable node for
// its status and peers, and aggregates a NetworkReport. It bounds the crawl by
// a maximum number of nodes to visit.
type Scanner struct {
	host     ScanHost
	maxNodes int
}

// NewScanner creates a Scanner. maxNodes caps how many nodes are visited; a
// value <= 0 means unbounded.
func NewScanner(host ScanHost, maxNodes int) *Scanner {
	return &Scanner{host: host, maxNodes: maxNodes}
}

// Scan crawls outward from the seed peers and returns the aggregated report.
func (s *Scanner) Scan(ctx context.Context, seeds []peer.AddrInfo) NetworkReport {
	report := NetworkReport{
		ClientVersions:  make(map[string]int),
		HeightHistogram: make(map[uint64]int),
		RoleCounts:      make(map[networks.Role]int),
	}
	visited := make(map[peer.ID]struct{})
	queue := append([]peer.AddrInfo(nil), seeds...)

	for len(queue) > 0 {
		if s.maxNodes > 0 && len(visited) >= s.maxNodes {
			break
		}
		info := queue[0]
		queue = queue[1:]
		if _, seen := visited[info.ID]; seen {
			continue
		}
		visited[info.ID] = struct{}{}

		node, err := s.query(ctx, info)
		if err != nil {
			continue
		}
		report.NodeCount++
		report.ClientVersions[node.ClientVersion]++
		report.HeightHistogram[node.BlockHeight]++
		report.RoleCounts[node.Role]++

		for _, discovered := range node.Peers {
			if _, seen := visited[discovered.ID]; !seen {
				queue = append(queue, discovered)
			}
		}
	}
	return report
}

// query runs the scan exchange against a single node.
func (s *Scanner) query(ctx context.Context, info peer.AddrInfo) (NodeReport, error) {
	if err := s.host.Connect(ctx, info); err != nil {
		return NodeReport{}, err
	}
	stream, err := s.host.OpenStream(ctx, info.ID, ScanProtocolID)
	if err != nil {
		return NodeReport{}, err
	}
	defer func() { _ = stream.Close() }()

	if err := stream.WriteMessage(&pb.ScanStatusRequest{}, maxScanRequestSize); err != nil {
		return NodeReport{}, err
	}
	var status pb.ScanStatusResponse
	if err := stream.ReadMessage(&status, maxScanStatusSize); err != nil {
		return NodeReport{}, err
	}
	if err := stream.WriteMessage(&pb.ScanPeersRequest{}, maxScanRequestSize); err != nil {
		return NodeReport{}, err
	}
	var peers pb.ScanPeersResponse
	if err := stream.ReadMessage(&peers, maxScanPeersSize); err != nil {
		return NodeReport{}, err
	}

	return NodeReport{
		Peer:          info.ID,
		Role:          roleFromProto(status.Role),
		ClientVersion: status.ClientVersion,
		BlockHeight:   status.BlockHeight,
		Peers:         parsePeerAddresses(peers.PeerAddresses),
	}, nil
}

// parsePeerAddresses parses multiaddr strings into de-duplicated AddrInfos.
func parsePeerAddresses(addresses []string) []peer.AddrInfo {
	byPeer := make(map[peer.ID]*peer.AddrInfo)
	for _, address := range addresses {
		info, err := peer.AddrInfoFromString(address)
		if err != nil {
			continue
		}
		if existing, ok := byPeer[info.ID]; ok {
			existing.Addrs = append(existing.Addrs, info.Addrs...)
			continue
		}
		byPeer[info.ID] = info
	}
	result := make([]peer.AddrInfo, 0, len(byPeer))
	for _, info := range byPeer {
		result = append(result, *info)
	}
	return result
}
