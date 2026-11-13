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
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPConfig is the configuration of the HTTP endpoint, as the node holds it.
type HTTPConfig struct {
	// Endpoint is the host and port to listen on.
	Endpoint string
	// PathPrefix is the path the requests are served on, the root if empty.
	PathPrefix         string
	CorsAllowedOrigins []string
	VirtualHosts       []string
	Timeouts           Timeouts
	Batch              BatchLimits
}

// Timeouts are the HTTP server's timeouts. All zero sets none; otherwise a
// timeout below a second is raised to its default, as go-ethereum does.
type Timeouts struct {
	Read, ReadHeader, Write, Idle time.Duration
}

// defaultTimeouts are go-ethereum's defaults of the HTTP timeouts.
var defaultTimeouts = Timeouts{Read: 30 * time.Second, ReadHeader: 30 * time.Second, Write: 30 * time.Second, Idle: 120 * time.Second}

// shutdownTimeout bounds how long stopping waits for HTTP requests in flight.
const shutdownTimeout = 5 * time.Second

// HTTPServer serves JSON-RPC over HTTP on one endpoint.
type HTTPServer struct {
	server   *http.Server
	listener net.Listener
}

// ValidatePrefix checks a path prefix of an endpoint.
func ValidatePrefix(what, prefix string) error {
	if prefix == "" {
		return nil
	}
	if prefix[0] != '/' {
		return fmt.Errorf(`%s RPC path prefix %q does not contain leading "/"`, what, prefix)
	}
	if strings.ContainsAny(prefix, "?#") {
		return fmt.Errorf("%s RPC path prefix %q contains URL meta-characters", what, prefix)
	}
	return nil
}

// StartHTTP listens on the endpoint and serves the front's calls in a
// goroutine of the dispatcher.
func StartHTTP(dispatcher *dispatch.Dispatcher, config HTTPConfig, front *Front, registerer prometheus.Registerer) (*HTTPServer, error) {
	if err := ValidatePrefix("HTTP", config.PathPrefix); err != nil {
		return nil, err
	}
	handler := &httpHandler{front: front, limits: config.Batch}
	server := &http.Server{Handler: &pathHandler{prefix: config.PathPrefix, next: newHTTPStack(handler, config.CorsAllowedOrigins, config.VirtualHosts)}}
	server.Protocols = new(http.Protocols)
	server.Protocols.SetHTTP1(true)
	server.Protocols.SetUnencryptedHTTP2(true)
	server.ConnState = countConnections(connectionsGauge(registerer).WithLabelValues("http"))
	if timeouts := config.Timeouts; timeouts != (Timeouts{}) {
		timeouts = sanitize(timeouts)
		server.ReadTimeout = timeouts.Read
		server.ReadHeaderTimeout = timeouts.ReadHeader
		server.WriteTimeout = timeouts.Write
		server.IdleTimeout = timeouts.Idle
		handler.timeout = timeouts.Write - timeoutMargin
	}
	listener, err := net.Listen("tcp", config.Endpoint)
	if err != nil {
		return nil, err
	}
	if !dispatcher.Go("HTTP server", func(context.Context) { _ = server.Serve(listener) }) {
		_ = listener.Close()
		return nil, errors.New("the RPC module is stopping")
	}
	log.Info("HTTP server started", "endpoint", listener.Addr(), "prefix", config.PathPrefix,
		"cors", strings.Join(config.CorsAllowedOrigins, ","), "vhosts", strings.Join(config.VirtualHosts, ","))
	return &HTTPServer{server: server, listener: listener}, nil
}

// Address is where the server listens.
func (s *HTTPServer) Address() net.Addr {
	return s.listener.Addr()
}

// Stop closes the listener and waits for the requests in flight, for at most
// the shutdown timeout.
func (s *HTTPServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(ctx); errors.Is(err, context.DeadlineExceeded) {
		log.Warn("HTTP server graceful shutdown timed out")
		_ = s.server.Close()
	}
	log.Info("HTTP server stopped", "endpoint", s.listener.Addr())
}

// countConnections counts the open connections of an HTTP server.
func countConnections(open prometheus.Gauge) func(net.Conn, http.ConnState) {
	return func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			open.Inc()
		case http.StateClosed, http.StateHijacked:
			open.Dec()
		}
	}
}

// sanitize raises each timeout below a second to its default.
func sanitize(timeouts Timeouts) Timeouts {
	raise := func(name string, value *time.Duration, fallback time.Duration) {
		if *value < time.Second {
			log.Warn("Sanitizing invalid HTTP "+name+" timeout", "provided", *value, "updated", fallback)
			*value = fallback
		}
	}
	raise("read", &timeouts.Read, defaultTimeouts.Read)
	raise("read header", &timeouts.ReadHeader, defaultTimeouts.ReadHeader)
	raise("write", &timeouts.Write, defaultTimeouts.Write)
	raise("idle", &timeouts.Idle, defaultTimeouts.Idle)
	return timeouts
}

// pathHandler serves the requests on the path prefix, at the root without
// one, and answers others 404.
type pathHandler struct {
	prefix string
	next   http.Handler
}

func (h *pathHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if !matchesPrefix(request, h.prefix) {
		response.WriteHeader(http.StatusNotFound)
		return
	}
	h.next.ServeHTTP(response, request)
}

func matchesPrefix(request *http.Request, prefix string) bool {
	if prefix == "" {
		return request.URL.Path == "/"
	}
	return strings.HasPrefix(request.URL.Path, prefix)
}
