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
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// --- PingProtocol ---

// TestPingProtocol_Handle_EchoesNonceAndReportsStatus checks the responder
// echoes the ping nonce and reports its configured role, client version, and
// block height.
func TestPingProtocol_Handle_EchoesNonceAndReportsStatus(t *testing.T) {
	status := networks.NodeStatus{Role: networks.RoleArchive, ClientVersion: "sonic/v9.9.9", BlockHeight: 1234}
	responder := startPingNode(t, status)
	prober := startStartedNode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, prober.Connect(ctx, addrInfoOf(responder)), "prober failed to connect")

	stream, err := prober.OpenStream(ctx, responder.ID(), PingProtocolID)
	require.NoError(t, err, "failed to open ping stream")
	defer func() { _ = stream.Close() }()

	nonce := []byte("0123456789abcdef")
	require.NoError(t, stream.WriteMessage(&pb.Ping{Nonce: nonce}, maxPingSize), "failed to write ping")
	var pong pb.Pong
	require.NoError(t, stream.ReadMessage(&pong, maxPongSize), "failed to read pong")

	require.Equal(t, nonce, pong.Nonce, "pong did not echo the nonce")
	require.Equal(t, pb.NodeRole_NODE_ROLE_ARCHIVE, pong.Role, "unexpected reported role")
	require.Equal(t, "sonic/v9.9.9", pong.ClientVersion, "unexpected reported client version")
	require.Equal(t, uint64(1234), pong.BlockHeight, "unexpected reported block height")
}

// --- HealthMonitor ---

// TestHealthMonitor_Record_UpdatesLatencyStatistics feeds a known sequence of
// probe outcomes and asserts the EWMA, jitter, loss rate, and reported status
// update deterministically.
func TestHealthMonitor_Record_UpdatesLatencyStatistics(t *testing.T) {
	monitor := NewHealthMonitor(fakeMonitorHost{}, HealthMonitorConfig{SmoothingFactor: 0.5}, prometheus.NewRegistry())
	peerID := p2p.PeerID("peer-1")

	monitor.record(peerID, 100*time.Millisecond, networks.NodeStatus{Role: networks.RoleValidator, ClientVersion: "v1", BlockHeight: 42}, nil)
	monitor.record(peerID, 200*time.Millisecond, networks.NodeStatus{Role: networks.RoleValidator, ClientVersion: "v1", BlockHeight: 43}, nil)
	monitor.record(peerID, 0, networks.NodeStatus{}, errors.New("probe failed"))

	sample := sampleFor(t, monitor, peerID)
	require.Equal(t, 200*time.Millisecond, sample.Latest, "latest should be the last successful RTT")
	require.Equal(t, 150*time.Millisecond, sample.Average, "EWMA: 0.5*200 + 0.5*100")
	require.Equal(t, 50*time.Millisecond, sample.Jitter, "jitter EWMA: 0.5*|200-100| + 0.5*0")
	require.Equal(t, uint64(3), sample.Probes, "all three probes counted")
	require.Equal(t, uint64(1), sample.Failures, "the failing probe counted")
	require.InDelta(t, 1.0/3.0, sample.LossRate(), 1e-9, "loss rate = failures/probes")
	require.Equal(t, uint64(43), sample.BlockHeight, "block height from the last success is retained")
	require.Equal(t, networks.RoleValidator, sample.Role, "reported role is retained")
}

// TestHealthMonitor_ProbeAll_PrunesDisconnectedPeers verifies a round drops the
// samples of peers that are no longer connected.
func TestHealthMonitor_ProbeAll_PrunesDisconnectedPeers(t *testing.T) {
	monitor := NewHealthMonitor(fakeMonitorHost{}, HealthMonitorConfig{}, prometheus.NewRegistry())
	stale := p2p.PeerID("stale")
	live := p2p.PeerID("live")
	monitor.record(stale, 10*time.Millisecond, networks.NodeStatus{}, nil)
	monitor.record(live, 10*time.Millisecond, networks.NodeStatus{}, nil)

	monitor.prune([]p2p.PeerID{live})

	peers := peerSet(monitor.Snapshot())
	require.NotContains(t, peers, stale, "a disconnected peer must be pruned")
	require.Contains(t, peers, live, "a connected peer must be retained")
}

// TestHealthMonitor_ProbeAll_RecordsRttAndReportedStatus runs a real probe round
// against a peer that serves the protocol and asserts RTT and reported status
// land in the snapshot and the success metric increments.
func TestHealthMonitor_ProbeAll_RecordsRttAndReportedStatus(t *testing.T) {
	status := networks.NodeStatus{Role: networks.RoleArchive, ClientVersion: "sonic/v9.9.9", BlockHeight: 1234}
	responder := startPingNode(t, status)
	prober := startStartedNode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	require.NoError(t, prober.Connect(ctx, addrInfoOf(responder)), "prober failed to connect")

	registry := prometheus.NewRegistry()
	monitor := NewHealthMonitor(prober, HealthMonitorConfig{Timeout: 5 * time.Second}, registry)
	monitor.ProbeAll(ctx)

	sample := sampleFor(t, monitor, responder.ID())
	require.Greater(t, sample.Latest, time.Duration(0), "expected a positive RTT")
	require.Greater(t, sample.Average, time.Duration(0), "expected a positive average RTT")
	require.Equal(t, uint64(1), sample.Probes, "expected one probe")
	require.Equal(t, uint64(0), sample.Failures, "expected no failures")
	require.Equal(t, networks.RoleArchive, sample.Role, "unexpected reported role")
	require.Equal(t, "sonic/v9.9.9", sample.ClientVersion, "unexpected reported client version")
	require.Equal(t, uint64(1234), sample.BlockHeight, "unexpected reported block height")

	require.GreaterOrEqual(t, counterValue(t, registry, "sonic_p2p_ping_probes_total", "result", "success"), float64(1), "expected a success probe metric")
}

// TestHealthMonitor_ProbeAll_CountsFailureForUnservedPeer probes a connected
// peer that does not serve the protocol and asserts the probe is recorded as a
// failure.
func TestHealthMonitor_ProbeAll_CountsFailureForUnservedPeer(t *testing.T) {
	responder := startStartedNode(t) // does not register PingProtocol
	prober := startStartedNode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	require.NoError(t, prober.Connect(ctx, addrInfoOf(responder)), "prober failed to connect")

	registry := prometheus.NewRegistry()
	monitor := NewHealthMonitor(prober, HealthMonitorConfig{Timeout: 3 * time.Second}, registry)
	monitor.ProbeAll(ctx)

	sample := sampleFor(t, monitor, responder.ID())
	require.Equal(t, uint64(1), sample.Probes, "expected one probe")
	require.Equal(t, uint64(1), sample.Failures, "expected the probe to fail")
	require.Equal(t, 1.0, sample.LossRate(), "expected full loss")

	require.GreaterOrEqual(t, counterValue(t, registry, "sonic_p2p_ping_probes_total", "result", "failure"), float64(1), "expected a failure probe metric")
}

// --- helpers ---

func startPingNode(t *testing.T, status networks.NodeStatus) *p2p.Node {
	t.Helper()
	node := startPlainNode(t)
	node.RegisterStreamProtocol(NewPingProtocol(fakeStatusSource{status: status}))
	require.NoError(t, node.Start(), "failed to start ping node")
	t.Cleanup(func() { _ = node.Stop() })
	return node
}

func startStartedNode(t *testing.T) *p2p.Node {
	t.Helper()
	node := startPlainNode(t)
	require.NoError(t, node.Start(), "failed to start node")
	t.Cleanup(func() { _ = node.Stop() })
	return node
}

func sampleFor(t *testing.T, monitor *HealthMonitor, peerID p2p.PeerID) HealthSample {
	t.Helper()
	for _, entry := range monitor.Snapshot() {
		if entry.Peer == peerID {
			return entry.Sample
		}
	}
	t.Fatalf("no health sample for peer %s", peerID)
	return HealthSample{}
}

func peerSet(snapshot []PeerHealth) map[p2p.PeerID]struct{} {
	peers := make(map[p2p.PeerID]struct{}, len(snapshot))
	for _, entry := range snapshot {
		peers[entry.Peer] = struct{}{}
	}
	return peers
}

// fakeMonitorHost is a MonitorHost for unit tests that exercise the monitor's
// bookkeeping without a real network.
type fakeMonitorHost struct {
	peers []p2p.PeerID
}

func (f fakeMonitorHost) ConnectedPeers() []p2p.PeerID { return f.peers }

func (f fakeMonitorHost) OpenStream(context.Context, p2p.PeerID, protocol.ID) (p2p.Stream, error) {
	return nil, errors.New("fakeMonitorHost: no streams")
}

func (f fakeMonitorHost) Logger() logger.Logger { return log.Root() }
