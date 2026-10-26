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

package dispatch

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// Results a request is counted under in rpc_requests_total.
const (
	resultOK            = "ok"
	resultError         = "error"
	resultInvalidParams = "invalid_params"
	resultNotFound      = "not_found"
	resultPanic         = "panic"
	resultRejected      = "rejected"
)

// otherMethod is the label of every method the dispatcher does not serve, so
// that a client cannot grow the label set by sending names.
const otherMethod = "other"

// Outcomes of a subscription event in rpc_subscription_events_total.
const (
	OutcomeSent    = "sent"
	OutcomeDropped = "dropped"
)

// Metrics are the counters of the RPC module. Every label value comes from a
// fixed set: method names the dispatcher registered, transports, subscription
// kinds and loops the module names. None is a client, an address, a block or
// an error text.
type Metrics struct {
	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
	responseBytes *prometheus.HistogramVec
	connections   *prometheus.GaugeVec
	inflight      prometheus.Gauge
	subscriptions *prometheus.GaugeVec
	subEvents     *prometheus.CounterVec
	panics        *prometheus.CounterVec
	loopRestarts  *prometheus.CounterVec
	registerer    prometheus.Registerer
}

// NewMetrics registers the module's metrics with reg; the node passes the
// default registry, tests one of their own. Metrics another module of the
// process registered already are shared, as when tests run several nodes in
// one process.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		requests: Register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_requests_total", Help: "RPC requests by method and result.",
		}, []string{"method", "result"})),
		duration: Register(reg, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "rpc_duration_seconds", Help: "Time to serve an RPC request, by method.",
			Buckets: []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		}, []string{"method"})),
		responseBytes: Register(reg, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "rpc_response_bytes", Help: "Size of the encoded result of an RPC request, by method.",
			Buckets: prometheus.ExponentialBuckets(64, 4, 12),
		}, []string{"method"})),
		connections: Register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_connections", Help: "Open RPC connections by transport.",
		}, []string{"transport"})),
		inflight: Register(reg, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_inflight", Help: "RPC requests being served.",
		})),
		subscriptions: Register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_subscriptions", Help: "Active subscriptions and polled filters by kind.",
		}, []string{"kind"})),
		subEvents: Register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_subscription_events_total", Help: "Subscription events by kind and outcome.",
		}, []string{"kind", "outcome"})),
		panics: Register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_panics_total", Help: "Recovered panics by method or loop.",
		}, []string{"method"})),
		loopRestarts: Register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_loop_restarts_total", Help: "Restarts of long-lived RPC loops after a panic.",
		}, []string{"loop"})),
		registerer: reg,
	}
}

// Register registers a collector, or returns the one registered under its
// name already.
func Register[C prometheus.Collector](reg prometheus.Registerer, c C) C {
	if err := reg.Register(c); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			if existing, ok := already.ExistingCollector.(C); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}

// Registerer is the registry the metrics live in, for the packages that add
// metrics of their own.
func (m *Metrics) Registerer() prometheus.Registerer {
	return m.registerer
}

// Connections counts the open connections of one transport.
func (m *Metrics) Connections(transport string) prometheus.Gauge {
	return m.connections.WithLabelValues(transport)
}

// Subscriptions counts the active consumers of one kind.
func (m *Metrics) Subscriptions(kind string) prometheus.Gauge {
	return m.subscriptions.WithLabelValues(kind)
}

// SubscriptionEvents counts the events of one kind by outcome.
func (m *Metrics) SubscriptionEvents(kind, outcome string) prometheus.Counter {
	return m.subEvents.WithLabelValues(kind, outcome)
}
