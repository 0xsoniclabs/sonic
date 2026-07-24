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

package debug

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewPProfServer_HasTimeouts(t *testing.T) {
	server := newPProfServer("127.0.0.1:0")
	require.NotZero(t, server.ReadTimeout, "ReadTimeout must be set")
	require.NotZero(t, server.IdleTimeout, "IdleTimeout must be set")
}

func TestNewPProfServer_HasAddress(t *testing.T) {
	addr := "127.0.0.1:6060"
	server := newPProfServer(addr)
	require.Equal(t, addr, server.Addr)
}

func TestStartPProf_ServesPProfEndpoints(t *testing.T) {
	addr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	StartPProf(addr, false)
	waitForServer(t, addr)

	client := &http.Client{Timeout: 5 * time.Second}

	paths := []string{
		"/debug/pprof/",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			resp, err := client.Get("http://" + addr + path)
			require.NoError(t, err)
			defer func() { require.NoError(t, resp.Body.Close()) }()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)
		})
	}
}

func TestStartPProf_WithMetrics_ServesMetricsEndpoint(t *testing.T) {
	addr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	StartPProf(addr, true)
	waitForServer(t, addr)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/debug/metrics/prometheus/native")
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		require.NoError(t, conn.Close())
		return true
	}, 2*time.Second, 10*time.Millisecond)
}
