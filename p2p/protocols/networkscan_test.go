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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
)

// TestScanner_CrawlsNetwork_AggregatesReport starts three real P2P nodes, has
// the middle node advertise the third as a peer, and verifies a scan launched
// from an observer discovers both and aggregates their status correctly.
func TestScanner_CrawlsNetwork_AggregatesReport(t *testing.T) {
	archive := startScanNode(t, networks.NodeStatus{
		Role: networks.RoleArchive, ClientVersion: "sonic/v2.2.0", BlockHeight: 500,
	}, nil)

	validator := startScanNode(t, networks.NodeStatus{
		Role: networks.RoleValidator, ClientVersion: "sonic/v2.2.0", BlockHeight: 500,
	}, []peer.AddrInfo{addrInfoOf(archive)})

	observer := startPlainNode(t)

	scanner := NewScanner(observer, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	report := scanner.Scan(ctx, []peer.AddrInfo{addrInfoOf(validator)})

	require.Equal(t, 2, report.NodeCount, "expected to reach 2 nodes")
	require.Equal(t, 1, report.RoleCounts[networks.RoleValidator], "unexpected role breakdown: %+v", report.RoleCounts)
	require.Equal(t, 1, report.RoleCounts[networks.RoleArchive], "unexpected role breakdown: %+v", report.RoleCounts)
	require.Equal(t, 2, report.ClientVersions["sonic/v2.2.0"], "unexpected client-version breakdown: %+v", report.ClientVersions)
	require.Equal(t, 2, report.HeightHistogram[500], "unexpected height histogram: %+v", report.HeightHistogram)
}

// --- helpers ---

func startScanNode(t *testing.T, status networks.NodeStatus, peers []peer.AddrInfo) *p2p.Node {
	t.Helper()
	node := startPlainNode(t)
	node.RegisterStreamProtocol(NewScanProtocol(
		fakeStatusSource{status: status},
		fakePeerSource{peers: peers},
	))
	require.NoError(t, node.Start(), "failed to start scan node")
	t.Cleanup(func() { _ = node.Stop() })
	return node
}

func startPlainNode(t *testing.T) *p2p.Node {
	t.Helper()
	config := p2p.DefaultConfig()
	config.ListenAddresses = []string{
		"/ip4/127.0.0.1/udp/0/quic-v1",
		"/ip4/127.0.0.1/tcp/0",
	}
	node, err := p2p.New(config, log.Root(), prometheus.NewRegistry())
	require.NoError(t, err, "failed to create node")
	// Plain nodes that are only dialed still need to be started to serve; nodes
	// with protocols are started by the caller. Start here is idempotent.
	return node
}

func addrInfoOf(node *p2p.Node) peer.AddrInfo {
	return peer.AddrInfo{ID: node.ID(), Addrs: node.Host().Addrs()}
}

type fakeStatusSource struct{ status networks.NodeStatus }

func (f fakeStatusSource) Status() networks.NodeStatus { return f.status }

type fakePeerSource struct{ peers []peer.AddrInfo }

func (f fakePeerSource) Peers() []peer.AddrInfo { return f.peers }
