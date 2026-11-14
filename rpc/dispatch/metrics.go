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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// unknownMethod is the method label of calls to methods no one serves, so
// that clients cannot grow the label set.
const unknownMethod = "other"

// metrics are the dispatcher's Prometheus metrics. Their labels are bounded:
// method names are those registered, and nothing is labelled by address,
// block, client or error text.
type metrics struct {
	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
	responseBytes *prometheus.HistogramVec
	inflight      prometheus.Gauge
	panics        *prometheus.CounterVec
	loopRestarts  *prometheus.CounterVec

	subscriptions      *prometheus.GaugeVec
	subscriptionEvents *prometheus.CounterVec
}

func newMetrics(registerer prometheus.Registerer) *metrics {
	return &metrics{
		requests: Register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_requests_total",
			Help: "Calls served, by method and result (success or error).",
		}, []string{"method", "result"})),
		duration: Register(registerer, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rpc_duration_seconds",
			Help:    "Time to serve a call, by method.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 4, 10),
		}, []string{"method"})),
		responseBytes: Register(registerer, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rpc_response_bytes",
			Help:    "Size of the encoded results, by method.",
			Buckets: prometheus.ExponentialBuckets(64, 8, 8),
		}, []string{"method"})),
		inflight: Register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_inflight",
			Help: "Calls being served.",
		})),
		panics: Register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_panics_total",
			Help: "Panics recovered while serving a call, by method.",
		}, []string{"method"})),
		loopRestarts: Register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_loop_restarts_total",
			Help: "Restarts of a long-lived loop after a panic, by loop.",
		}, []string{"loop"})),
		subscriptions: Register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_subscriptions",
			Help: "Live subscriptions, by kind.",
		}, []string{"kind"})),
		subscriptionEvents: Register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_subscription_events_total",
			Help: "Notifications of subscriptions, by kind and outcome (sent or failed).",
		}, []string{"kind", "outcome"})),
	}
}

// Register registers a collector and answers it, or the one registered
// before under the same name: several nodes in one process, as integration
// tests run them, share their metrics.
func Register[Collector prometheus.Collector](registerer prometheus.Registerer, collector Collector) Collector {
	err := registerer.Register(collector)
	if registered, exists := errors.AsType[prometheus.AlreadyRegisteredError](err); exists {
		return registered.ExistingCollector.(Collector)
	}
	if err != nil {
		panic(err)
	}
	return collector
}

func (m *metrics) observe(method string, response Response, duration time.Duration) {
	result := "success"
	if response.Error != nil {
		result = "error"
	}
	m.requests.WithLabelValues(method, result).Inc()
	m.duration.WithLabelValues(method).Observe(duration.Seconds())
	m.responseBytes.WithLabelValues(method).Observe(float64(len(response.Result)))
}

func (m *metrics) countUnknown() {
	m.requests.WithLabelValues(unknownMethod, "error").Inc()
}
