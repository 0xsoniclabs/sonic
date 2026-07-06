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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics is the Prometheus instrumentation for the P2P layer. Label
// cardinality is deliberately bounded: metrics are labelled by
// protocol/topic/direction/result/reason and never by peer ID.
type Metrics struct {
	streamBytes     *prometheus.CounterVec
	messages        *prometheus.CounterVec
	rateDropped     *prometheus.CounterVec
	peerDisconnects *prometheus.CounterVec
	connections     *prometheus.GaugeVec
	gossip          *prometheus.CounterVec
	scanDuration    prometheus.Histogram
}

// NewMetrics registers and returns the P2P metric collectors on the given
// registerer. A nil registerer falls back to the default one, so the future
// node wiring can hand in its own registry (see HANDOFF.md).
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	factory := promauto.With(registerer)
	return &Metrics{
		streamBytes: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_stream_bytes_total",
			Help: "Total bytes transferred over protocol streams.",
		}, []string{"direction", "protocol"}),
		messages: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_messages_total",
			Help: "Total framed protocol messages processed.",
		}, []string{"direction", "protocol", "result"}),
		rateDropped: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_ratelimit_dropped_total",
			Help: "Messages dropped because a per-peer rate limit was exceeded.",
		}, []string{"protocol", "reason"}),
		peerDisconnects: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_peer_disconnects_total",
			Help: "Peers disconnected by the node, by reason.",
		}, []string{"reason"}),
		connections: factory.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sonic_p2p_connections",
			Help: "Current number of connections.",
		}, []string{"direction"}),
		gossip: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "sonic_p2p_gossip_messages_total",
			Help: "Total gossipsub messages by topic and validation result.",
		}, []string{"topic", "result"}),
		scanDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "sonic_p2p_scan_duration_seconds",
			Help:    "Duration of a completed network scan.",
			Buckets: prometheus.DefBuckets,
		}),
	}
}
