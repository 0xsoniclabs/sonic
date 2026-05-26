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

package utils

import "github.com/prometheus/client_golang/prometheus"

//go:generate mockgen -source=metrics.go -destination=metrics_mock.go -package=utils

// MetricsHistogram is an interface that wraps the methods of a
// prometheus Histogram to facilitate testing with mocks.
type MetricsHistogram interface {
	Observe(float64)
}

// PrometheusHistogram wraps a prometheus.Histogram to implement the
// MetricsHistogram interface. Unlike go-metrics histograms, which
// are exported as Prometheus summaries (quantiles), this produces a native
// Prometheus bucketed histogram suitable for Grafana heatmaps.
type PrometheusHistogram struct {
	histogram prometheus.Histogram
}

// NewPrometheusHistogram creates a new PrometheusHistogram with the given
// options, registered with the default Prometheus registerer.
func NewPrometheusHistogram(opts prometheus.HistogramOpts) *PrometheusHistogram {
	h := prometheus.NewHistogram(opts)
	prometheus.MustRegister(h)
	return &PrometheusHistogram{histogram: h}
}

func (h *PrometheusHistogram) Observe(v float64) {
	h.histogram.Observe(v)
}

// MetricsCounter is an interface that wraps the methods of a
// prometheus Counter to facilitate testing with mocks.
type MetricsCounter interface {
	Inc(int64)
}
