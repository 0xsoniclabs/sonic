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

package transport

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

type fixedHealth struct{ served, committed uint64 }

func (health fixedHealth) Health() (uint64, uint64) { return health.served, health.committed }

// newTestServer starts a server on a free port of the loopback interface.
func newTestServer(t *testing.T, prefix string) (*Server, *dispatch.Dispatcher, *prometheus.Registry) {
	t.Helper()
	registry := prometheus.NewRegistry()
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(registry))
	registerModuleService(dispatcher, 0)
	server := NewServer("127.0.0.1:0", Timeouts{Read: time.Second, ReadHeader: time.Second, Write: 5 * time.Second, Idle: time.Second}, dispatcher)
	server.EnableHTTP(NewFront(dispatcher, nil), HTTPConfig{PathPrefix: prefix, VirtualHosts: []string{"localhost"}}, fixedHealth{served: 5, committed: 7})
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		server.Stop()
		dispatcher.Stop()
	})
	return server, dispatcher, registry
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	response, err := noReuse.RoundTrip(mustRequest(t, http.MethodGet, url, ""))
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response.StatusCode, string(body)
}

func mustRequest(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestServer_ServesCallsOnItsAddress(t *testing.T) {
	server, _, _ := newTestServer(t, "")
	response, err := noReuse.RoundTrip(mustRequest(t, http.MethodPost, "http://"+server.Address(), bodies["call"]))
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","id":1,"result":[5,"five"]}`+"\n", string(body))
}

func TestServer_ReportsTheHeadOnHealth(t *testing.T) {
	server, _, _ := newTestServer(t, "")
	status, body := get(t, "http://"+server.Address()+"/health")
	require.Equal(t, http.StatusOK, status)
	require.JSONEq(t, `{"head":5,"committed":7,"lag":2}`, body)
}

func TestServer_ServesOnlyUnderItsPrefix(t *testing.T) {
	server, _, _ := newTestServer(t, "/rpc")
	status, _ := get(t, "http://"+server.Address()+"/other")
	require.Equal(t, http.StatusNotFound, status)
	response, err := noReuse.RoundTrip(mustRequest(t, http.MethodPost, "http://"+server.Address()+"/rpc", bodies["call"]))
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	response, err = noReuse.RoundTrip(mustRequest(t, http.MethodPost, "http://"+server.Address()+"/", bodies["call"]))
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestServer_CountsItsConnections(t *testing.T) {
	server, _, registry := newTestServer(t, "")
	connection, err := net.Dial("tcp", server.Address())
	require.NoError(t, err)
	eventually(t, func() bool { return scrapeGauge(t, registry, "rpc_connections") == 1 }, "the connection to be counted")
	require.NoError(t, connection.Close())
	eventually(t, func() bool { return scrapeGauge(t, registry, "rpc_connections") == 0 }, "the closed connection to be uncounted")
}

func TestServer_StopClosesTheListener(t *testing.T) {
	server, dispatcher, _ := newTestServer(t, "")
	server.Stop()
	_, err := server.listener.Accept()
	require.ErrorIs(t, err, net.ErrClosed)
	server.Stop()
	stopped := make(chan struct{})
	go func() {
		dispatcher.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("the server goroutine did not end")
	}
}

func TestServer_StartFailsOnATakenAddress(t *testing.T) {
	first, dispatcher, _ := newTestServer(t, "")
	second := NewServer(first.Address(), Timeouts{}, dispatcher)
	require.Error(t, second.Start())
	require.Equal(t, first.Address(), second.Address(), "an unstarted server reports its configured address")
}

func TestServer_StartFailsOnceTheDispatcherStops(t *testing.T) {
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(prometheus.NewRegistry()))
	dispatcher.Stop()
	server := NewServer("127.0.0.1:0", Timeouts{}, dispatcher)
	require.ErrorIs(t, server.Start(), dispatch.ErrStopping)
}

func scrapeGauge(t *testing.T, registry *prometheus.Registry, name string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	var total float64
	for _, family := range families {
		if family.GetName() == name {
			for _, metric := range family.GetMetric() {
				total += metric.GetGauge().GetValue()
			}
		}
	}
	return total
}

func eventually(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestCheckPath_MatchesGoEthereum(t *testing.T) {
	request := func(path string) *http.Request {
		built, _ := http.NewRequest(http.MethodGet, "http://localhost"+path, nil)
		return built
	}
	require.True(t, checkPath(request("/"), ""))
	require.False(t, checkPath(request("/x"), ""))
	require.True(t, checkPath(request("/rpc/x"), "/rpc"))
	require.False(t, checkPath(request("/rp"), "/rpc"))
}
