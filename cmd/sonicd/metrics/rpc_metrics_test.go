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

package metrics

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// TestMetricsServer_ServesTheMetricsOfTheRPCModule is MON1: the module counts
// on the default registry, which the node's metrics endpoint serves.
func TestMetricsServer_ServesTheMetricsOfTheRPCModule(t *testing.T) {
	module := rpc.New(tests.NewFakeCore(), rpc.DefaultConfig(), prometheus.DefaultRegisterer)
	require.NoError(t, module.Start())
	defer func() { _ = module.Stop() }()
	require.NoError(t, module.Call(context.Background(), nil, "eth_chainId"))

	server := httptest.NewServer(newMetricsServer("").Handler)
	defer server.Close()
	resp, err := server.Client().Get(server.URL + "/debug/metrics/prometheus/native")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	for _, metric := range []string{"rpc_requests_total", "rpc_duration_seconds", "rpc_head_served", "rpc_inflight"} {
		require.Contains(t, string(body), metric)
	}
	require.Contains(t, string(body), `rpc_requests_total{method="eth_chainId",result="ok"} 1`)
}
