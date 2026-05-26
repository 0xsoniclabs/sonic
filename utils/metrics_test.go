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

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheusHistogram_RegistersAndUpdate(t *testing.T) {
	// NewPrometheusHistogram registers with the default Prometheus
	// registerer. To avoid side-effects we swap in a fresh registry.
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	t.Cleanup(func() { prometheus.DefaultRegisterer = origRegisterer })

	histogram := NewPrometheusHistogram(prometheus.HistogramOpts{
		Name:    "test_new_histogram",
		Help:    "test histogram",
		Buckets: []float64{1, 5, 10},
	})
	require.NotNil(t, histogram)

	histogram.Observe(0.5) // bucket <=1
	histogram.Observe(3.0) // bucket <=5
	histogram.Observe(7.0) // bucket <=10
	histogram.Observe(20)  // +Inf bucket

	gathered, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, gathered, 1)
	require.Equal(t, "test_new_histogram", gathered[0].GetName())

	hist := gathered[0].GetMetric()[0].GetHistogram()
	require.Equal(t, uint64(4), hist.GetSampleCount())
	require.InDelta(t, 30.5, hist.GetSampleSum(), 1e-9)
}
