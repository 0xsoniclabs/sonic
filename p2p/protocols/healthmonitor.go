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

// Peer-health monitoring protocol.
//
// This is an optional diagnostic protocol a node operator can activate to
// measure the quality of this node's P2P connections: the round-trip time (RTT)
// to each directly-connected peer, plus that peer's self-reported status (role,
// client version, synced block height). It is built on the open/closed registry
// exactly like the network scan, and inherits the framed stream's per-peer rate
// limiting and per-message size caps.
//
// Wire exchange (protocol /sonic/ping/1, one exchange per stream):
//
//	prober                              responder (PingProtocol)
//	  | -- Ping{nonce} ---------------->  |
//	  | <- Pong{nonce, role,              |
//	  |         client_version, height}   |
//
// The prober measures RTT locally as the wall-clock time from writing the Ping
// to reading the matching Pong, so no clock is trusted across peers. The nonce
// guards against a mismatched/replayed response on the stream.
//
// Activation and usage:
//
//	// On every node that should answer probes (report its own status):
//	node.RegisterStreamProtocol(protocols.NewPingProtocol(statusSource))
//
//	// On the diagnosing node, drive periodic probing of connected peers:
//	monitor := protocols.NewHealthMonitor(node, protocols.HealthMonitorConfig{}, registry)
//	monitor.Start(ctx)
//	defer monitor.Stop()
//
//	// Read results: aggregate RTT distribution and probe outcomes are exported
//	// as the Prometheus metrics below; per-peer detail (for pinpointing a slow
//	// or lagging connection) is available in memory:
//	for _, peer := range monitor.Snapshot() {
//		// peer.Peer, peer.Sample.Average, peer.Sample.Jitter,
//		// peer.Sample.LossRate(), peer.Sample.BlockHeight, ...
//	}
//
// Metrics (bounded cardinality — never labelled by peer ID):
//   - sonic_p2p_ping_rtt_seconds     histogram of probe RTTs across peers
//   - sonic_p2p_ping_probes_total{result} counter of probe outcomes
//
// Per-peer status is deliberately kept out of the metrics (unbounded
// cardinality) and exposed only through Snapshot.

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// PingProtocolID is the libp2p protocol the peer-health probe runs on.
const PingProtocolID = protocol.ID("/sonic/ping/1")

// Per-message-type size caps. Both messages are small; the pong is bounded
// generously to accommodate the client-version string.
const (
	maxPingSize = 64
	maxPongSize = 1 << 10
)

// nonceSize is the length of the opaque nonce that matches a pong to its ping.
const nonceSize = 16

// PingProtocol is the responder side of the peer-health protocol: it answers a
// Ping with a Pong that echoes the nonce and reports this node's current status.
// It is registered on the node like any other StreamProtocol.
type PingProtocol struct {
	status networks.NodeStatusSource
}

// NewPingProtocol creates the responder, sourcing this node's reported status
// (role, client version, synced height) from status.
func NewPingProtocol(status networks.NodeStatusSource) *PingProtocol {
	return &PingProtocol{status: status}
}

// ProtocolID implements p2p.StreamProtocol.
func (p *PingProtocol) ProtocolID() protocol.ID { return PingProtocolID }

// Handle serves a single probe: it reads one Ping and writes one Pong echoing
// the nonce alongside this node's status. It is one-shot (one exchange per
// stream) to keep the amplification factor at one.
func (p *PingProtocol) Handle(stream p2p.Stream) {
	defer func() { _ = stream.Close() }()

	var ping pb.Ping
	if err := stream.ReadMessage(&ping, maxPingSize); err != nil {
		_ = stream.Reset()
		return
	}
	status := p.status.Status()
	pong := &pb.Pong{
		Nonce:         ping.Nonce,
		Role:          roleToProto(status.Role),
		ClientVersion: status.ClientVersion,
		BlockHeight:   status.BlockHeight,
	}
	if err := stream.WriteMessage(pong, maxPongSize); err != nil {
		return
	}
}

// MonitorHost is the subset of the P2P node the health monitor needs: the set of
// currently-connected peers to probe, a way to open a stream to one, and the
// shared logger. *p2p.Node satisfies it.
type MonitorHost interface {
	ConnectedPeers() []p2p.PeerID
	OpenStream(ctx context.Context, target p2p.PeerID, id protocol.ID) (p2p.Stream, error)
	Logger() logger.Logger
}

// HealthMonitorConfig tunes the active prober. Zero values are replaced with
// sensible defaults.
type HealthMonitorConfig struct {
	// Interval is the delay between probe rounds.
	Interval time.Duration
	// Timeout bounds a single peer probe.
	Timeout time.Duration
	// MaxConcurrentProbes caps how many peers are probed at once in a round.
	MaxConcurrentProbes int
	// SmoothingFactor is the EWMA weight (alpha, 0..1) applied to a new sample.
	SmoothingFactor float64
}

func (c HealthMonitorConfig) withDefaults() HealthMonitorConfig {
	if c.Interval <= 0 {
		c.Interval = 30 * time.Second
	}
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
	if c.MaxConcurrentProbes <= 0 {
		c.MaxConcurrentProbes = 16
	}
	if c.SmoothingFactor <= 0 || c.SmoothingFactor > 1 {
		c.SmoothingFactor = 0.2
	}
	return c
}

// HealthSample is the accumulated health of a single peer connection.
type HealthSample struct {
	// Latest is the most recent successful RTT.
	Latest time.Duration
	// Average is the exponentially-weighted moving average of the RTT.
	Average time.Duration
	// Jitter is the EWMA of the absolute change between successive RTTs.
	Jitter time.Duration
	// Probes is the total number of probes attempted against the peer.
	Probes uint64
	// Failures is how many of those probes failed (timeout, refused, mismatch).
	Failures uint64
	// LastProbe is when the peer was last probed.
	LastProbe time.Time
	// Role is the peer's last self-reported role.
	Role networks.Role
	// ClientVersion is the peer's last self-reported client version.
	ClientVersion string
	// BlockHeight is the peer's last self-reported synced height.
	BlockHeight uint64
}

// LossRate returns the fraction of probes that failed, in [0,1].
func (s HealthSample) LossRate() float64 {
	if s.Probes == 0 {
		return 0
	}
	return float64(s.Failures) / float64(s.Probes)
}

// PeerHealth pairs a peer with its accumulated health sample.
type PeerHealth struct {
	Peer   p2p.PeerID
	Sample HealthSample
}

// HealthMonitor periodically probes this node's connected peers, measuring RTT
// and collecting their reported status, and exposes the result as an aggregate
// metric and a per-peer snapshot. See the file header for the protocol and usage.
type HealthMonitor struct {
	host    MonitorHost
	config  HealthMonitorConfig
	logger  logger.Logger
	metrics *healthMetrics

	// now is injectable so tests can drive RTT and EWMA math deterministically.
	now func() time.Time

	mutex   sync.Mutex
	samples map[p2p.PeerID]*HealthSample

	wait          sync.WaitGroup
	cancelContext context.CancelFunc
}

// NewHealthMonitor creates a health monitor over host. Its metrics are
// registered on registerer (a nil registerer uses the default one).
func NewHealthMonitor(host MonitorHost, config HealthMonitorConfig, registerer prometheus.Registerer) *HealthMonitor {
	return &HealthMonitor{
		host:    host,
		config:  config.withDefaults(),
		logger:  host.Logger(),
		metrics: newHealthMetrics(registerer),
		now:     time.Now,
		samples: make(map[p2p.PeerID]*HealthSample),
	}
}

// Start begins probing connected peers every Interval until Stop is called or
// ctx is cancelled.
func (m *HealthMonitor) Start(ctx context.Context) {
	runContext, cancel := context.WithCancel(ctx)
	m.cancelContext = cancel
	m.wait.Add(1)
	go m.loop(runContext)
}

// Stop ends probing and waits for the in-flight round to finish.
func (m *HealthMonitor) Stop() {
	if m.cancelContext != nil {
		m.cancelContext()
		m.cancelContext = nil
	}
	m.wait.Wait()
}

// ProbeAll runs a single probe round against all currently-connected peers,
// pruning samples for peers that have since disconnected. Start calls it every
// Interval; it is exported so a round can also be driven on demand.
func (m *HealthMonitor) ProbeAll(ctx context.Context) {
	peers := m.host.ConnectedPeers()
	m.prune(peers)

	semaphore := make(chan struct{}, m.config.MaxConcurrentProbes)
	var round sync.WaitGroup
	for _, peerID := range peers {
		round.Add(1)
		semaphore <- struct{}{}
		go func(peerID p2p.PeerID) {
			defer round.Done()
			defer func() { <-semaphore }()
			m.probe(ctx, peerID)
		}(peerID)
	}
	round.Wait()
	m.logRound()
}

// Snapshot returns the current per-peer health, for logs or a diagnostic API.
func (m *HealthMonitor) Snapshot() []PeerHealth {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	result := make([]PeerHealth, 0, len(m.samples))
	for peerID, sample := range m.samples {
		result = append(result, PeerHealth{Peer: peerID, Sample: *sample})
	}
	return result
}

func (m *HealthMonitor) loop(ctx context.Context) {
	defer m.wait.Done()
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.probeAllWithContext(ctx)
		}
	}
}

// probeAllWithContext exists so the loop honours cancellation between rounds.
func (m *HealthMonitor) probeAllWithContext(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	m.ProbeAll(ctx)
}

func (m *HealthMonitor) probe(ctx context.Context, peerID p2p.PeerID) {
	probeContext, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	status, rtt, err := m.exchange(probeContext, peerID)
	m.record(peerID, rtt, status, err)
	if err != nil {
		m.metrics.probes.WithLabelValues("failure").Inc()
		return
	}
	m.metrics.rtt.Observe(rtt.Seconds())
	m.metrics.probes.WithLabelValues("success").Inc()
}

// exchange runs one ping/pong against a peer and returns its reported status and
// the measured RTT.
func (m *HealthMonitor) exchange(ctx context.Context, peerID p2p.PeerID) (networks.NodeStatus, time.Duration, error) {
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return networks.NodeStatus{}, 0, err
	}

	start := m.now()
	stream, err := m.host.OpenStream(ctx, peerID, PingProtocolID)
	if err != nil {
		return networks.NodeStatus{}, 0, err
	}
	defer func() { _ = stream.Close() }()

	if err := stream.WriteMessage(&pb.Ping{Nonce: nonce}, maxPingSize); err != nil {
		_ = stream.Reset()
		return networks.NodeStatus{}, 0, err
	}
	var pong pb.Pong
	if err := stream.ReadMessage(&pong, maxPongSize); err != nil {
		_ = stream.Reset()
		return networks.NodeStatus{}, 0, err
	}
	rtt := m.now().Sub(start)
	if !bytes.Equal(pong.Nonce, nonce) {
		return networks.NodeStatus{}, 0, errNonceMismatch
	}
	status := networks.NodeStatus{
		Role:          roleFromProto(pong.Role),
		ClientVersion: pong.ClientVersion,
		BlockHeight:   pong.BlockHeight,
	}
	return status, rtt, nil
}

// record folds one probe outcome into the peer's sample.
func (m *HealthMonitor) record(peerID p2p.PeerID, rtt time.Duration, status networks.NodeStatus, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sample := m.samples[peerID]
	if sample == nil {
		sample = &HealthSample{}
		m.samples[peerID] = sample
	}
	sample.Probes++
	sample.LastProbe = m.now()
	if err != nil {
		sample.Failures++
		return
	}

	alpha := m.config.SmoothingFactor
	if sample.Average == 0 {
		sample.Average = rtt // first successful sample seeds the average
	} else {
		sample.Jitter = ewma(sample.Jitter, absDuration(rtt-sample.Latest), alpha)
		sample.Average = ewma(sample.Average, rtt, alpha)
	}
	sample.Latest = rtt
	sample.Role = status.Role
	sample.ClientVersion = status.ClientVersion
	sample.BlockHeight = status.BlockHeight
}

// prune drops samples for peers that are no longer connected.
func (m *HealthMonitor) prune(connected []p2p.PeerID) {
	live := make(map[p2p.PeerID]struct{}, len(connected))
	for _, peerID := range connected {
		live[peerID] = struct{}{}
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for peerID := range m.samples {
		if _, ok := live[peerID]; !ok {
			delete(m.samples, peerID)
		}
	}
}

// logRound emits a one-line summary of the round: the slowest peer and how many
// peers show any probe loss. Full detail is available via Snapshot.
func (m *HealthMonitor) logRound() {
	snapshot := m.Snapshot()
	if len(snapshot) == 0 {
		return
	}
	worst := snapshot[0]
	lossy := 0
	for _, entry := range snapshot {
		if entry.Sample.Latest > worst.Sample.Latest {
			worst = entry
		}
		if entry.Sample.LossRate() > 0 {
			lossy++
		}
	}
	m.logger.Debug("peer-health round complete",
		"peers", len(snapshot),
		"slowest_peer", worst.Peer,
		"slowest_rtt", worst.Sample.Latest,
		"peers_with_loss", lossy,
	)
}

// healthMetrics holds the monitor's own Prometheus collectors, kept out of the
// core metrics so the core stays closed to modification.
type healthMetrics struct {
	rtt    prometheus.Histogram
	probes *prometheus.CounterVec
}

func newHealthMetrics(registerer prometheus.Registerer) *healthMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	factory := promauto.With(registerer)
	return &healthMetrics{
		rtt: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "sonic_p2p_ping_rtt_seconds",
			Help:    "Round-trip time of peer-health probes.",
			Buckets: prometheus.DefBuckets,
		}),
		probes: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_ping_probes_total",
			Help: "Peer-health probes by result.",
		}, []string{"result"}),
	}
}

// errNonceMismatch is returned when a pong does not echo the ping's nonce.
var errNonceMismatch = errors.New("p2p: ping nonce mismatch")

// ewma returns the exponentially-weighted moving average of current and sample.
func ewma(current, sample time.Duration, alpha float64) time.Duration {
	return time.Duration(alpha*float64(sample) + (1-alpha)*float64(current))
}

// absDuration returns the absolute value of a duration.
func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
