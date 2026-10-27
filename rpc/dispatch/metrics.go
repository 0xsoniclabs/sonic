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

// The results a request is counted under in rpc_requests_total.
const (
	ResultSuccess  = "success"
	ResultError    = "error"
	ResultInvalid  = "invalid"
	ResultPanic    = "panic"
	ResultRejected = "rejected"
)

// OtherMethod is the label of every method name no Register names, so that a
// client cannot add label values.
const OtherMethod = "other"

// The outcomes an event of a subscription or polled filter is counted under
// in rpc_subscription_events_total.
const (
	// OutcomeDelivered is an event written to the client, or answered by a
	// poll.
	OutcomeDelivered = "delivered"
	// OutcomeFailed is a notification whose write to the client failed.
	OutcomeFailed = "failed"
	// OutcomeLost counts a subscription or filter that ended because its
	// feed lost events, once each: how many it lost is not known.
	OutcomeLost = "lost"
)

// Metrics are the module's Prometheus metrics. Labels are bounded by
// construction: methods by the registered names, transports, subscription
// kinds and loops by the module's own constants; no label names an address,
// a block, a client or an error text.
type Metrics struct {
	Requests           *prometheus.CounterVec
	Duration           *prometheus.HistogramVec
	ResponseBytes      *prometheus.HistogramVec
	Panics             *prometheus.CounterVec
	Inflight           prometheus.Gauge
	Connections        *prometheus.GaugeVec
	Subscriptions      *prometheus.GaugeVec
	SubscriptionEvents *prometheus.CounterVec
	LoopRestarts       *prometheus.CounterVec
	HeadServed         prometheus.Gauge
	HeadLag            prometheus.Gauge
}

// NewMetrics creates the metrics and registers them. A metric another module
// of the process registered before is shared with it.
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	return &Metrics{
		Requests: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_requests_total",
			Help: "RPC requests by method and result.",
		}, []string{"method", "result"})),
		Duration: register(registerer, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rpc_duration_seconds",
			Help:    "Time to answer an RPC request, by method.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 4, 10),
		}, []string{"method"})),
		ResponseBytes: register(registerer, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rpc_response_bytes",
			Help:    "Size of the encoded result of an RPC request, by method.",
			Buckets: prometheus.ExponentialBuckets(64, 4, 10),
		}, []string{"method"})),
		Panics: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_panics_total",
			Help: "Panics recovered while serving RPC requests, by method.",
		}, []string{"method"})),
		Inflight: register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_inflight",
			Help: "RPC requests being served.",
		})),
		Connections: register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_connections",
			Help: "Open RPC connections, by transport.",
		}, []string{"transport"})),
		Subscriptions: register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "rpc_subscriptions",
			Help: "Live subscriptions and polled filters, by kind.",
		}, []string{"kind"})),
		SubscriptionEvents: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_subscription_events_total",
			Help: "Events handed to subscriptions and filters, by kind and outcome: delivered, failed, and lost (a subscription or filter that ended because its feed lost events).",
		}, []string{"kind", "outcome"})),
		LoopRestarts: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rpc_loop_restarts_total",
			Help: "Restarts of the module's long-lived loops after a panic or failure, by loop.",
		}, []string{"loop"})),
		HeadServed: register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_served",
			Help: "Number of the block the module serves as latest.",
		})),
		HeadLag: register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_lag",
			Help: "Committed head minus the served head, in blocks.",
		})),
	}
}

func register[Collector prometheus.Collector](registerer prometheus.Registerer, collector Collector) Collector {
	err := registerer.Register(collector)
	var registered prometheus.AlreadyRegisteredError
	if errors.As(err, &registered) {
		return registered.ExistingCollector.(Collector)
	}
	if err != nil {
		panic(err)
	}
	return collector
}
