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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// otherLabel is the method label of every request whose method is not
// registered. Labels are bounded by the method table, never by client input.
const otherLabel = "other"

// Result labels of rpc_requests_total.
const (
	ResultOK            = "ok"
	ResultError         = "error"
	ResultInvalidParams = "invalid_params"
	ResultNotFound      = "not_found"
	ResultPanic         = "panic"
	ResultTimeout       = "timeout"
	ResultCancelled     = "cancelled"
	ResultRejected      = "rejected"
)

// Metrics are the Prometheus collectors of the module.
type Metrics struct {
	Requests           *prometheus.CounterVec   // rpc_requests_total{method,result}
	Duration           *prometheus.HistogramVec // rpc_duration_seconds{method}
	ResponseBytes      *prometheus.HistogramVec // rpc_response_bytes{method}
	Connections        *prometheus.GaugeVec     // rpc_connections{transport}
	Inflight           prometheus.Gauge         // rpc_inflight
	Subscriptions      *prometheus.GaugeVec     // rpc_subscriptions{kind}
	SubscriptionEvents *prometheus.CounterVec   // rpc_subscription_events_total{kind,outcome}
	HeadServed         prometheus.Gauge         // rpc_head_served
	HeadLag            prometheus.Gauge         // rpc_head_lag
	Panics             *prometheus.CounterVec   // rpc_panics_total{method}
	LoopRestarts       *prometheus.CounterVec   // rpc_loop_restarts_total{loop}
}

// NewMetrics creates the collectors and registers them. Collectors already
// registered on the registerer are reused, so several modules in one process
// (as in integration tests) share them.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_requests_total", Help: "RPC requests by method and result.",
		}, []string{"method", "result"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "rpc_duration_seconds", Help: "RPC request duration by method.",
			Buckets: []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		}, []string{"method"}),
		ResponseBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "rpc_response_bytes", Help: "Size of the encoded RPC results by method.",
			Buckets: prometheus.ExponentialBuckets(64, 4, 12),
		}, []string{"method"}),
		Connections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_connections", Help: "Open RPC connections by transport.",
		}, []string{"transport"}),
		Inflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_inflight", Help: "RPC requests being served.",
		}),
		Subscriptions: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_subscriptions", Help: "Active subscriptions by kind.",
		}, []string{"kind"}),
		SubscriptionEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_subscription_events_total", Help: "Subscription events by kind and outcome.",
		}, []string{"kind", "outcome"}),
		HeadServed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_served", Help: "Number of the block served as latest.",
		}),
		HeadLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_lag", Help: "Committed block number minus the served head.",
		}),
		Panics: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_panics_total", Help: "Recovered panics by method.",
		}, []string{"method"}),
		LoopRestarts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_loop_restarts_total", Help: "Restarts of the module's loops after a panic.",
		}, []string{"loop"}),
	}
	if reg == nil {
		return m
	}
	m.Requests = register(reg, m.Requests)
	m.Duration = register(reg, m.Duration)
	m.ResponseBytes = register(reg, m.ResponseBytes)
	m.Connections = register(reg, m.Connections)
	m.Inflight = register(reg, m.Inflight)
	m.Subscriptions = register(reg, m.Subscriptions)
	m.SubscriptionEvents = register(reg, m.SubscriptionEvents)
	m.HeadServed = register(reg, m.HeadServed)
	m.HeadLag = register(reg, m.HeadLag)
	m.Panics = register(reg, m.Panics)
	m.LoopRestarts = register(reg, m.LoopRestarts)
	return m
}

func register[C prometheus.Collector](reg prometheus.Registerer, c C) C {
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

var (
	defaultMetricsOnce sync.Once
	defaultMetrics     *Metrics
)

// DefaultMetrics returns the collectors registered on the default Prometheus
// registry, which the node's metrics endpoint serves.
func DefaultMetrics() *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetrics = NewMetrics(prometheus.DefaultRegisterer)
	})
	return defaultMetrics
}
