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
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Timeouts are the timeouts of an HTTP server as the node's configuration
// holds them, already checked by the node; zero values set no timeout.
type Timeouts struct {
	Read       time.Duration
	ReadHeader time.Duration
	Write      time.Duration
	Idle       time.Duration
}

// Health reports the head the module serves and the committed head.
type Health interface {
	Health() (served, committed uint64)
}

// Server listens on one address and serves the HTTP endpoint, the WebSocket
// endpoint, or both on one port, as go-ethereum's node shares a server when
// the ports match.
type Server struct {
	address    string
	timeouts   Timeouts
	dispatcher *dispatch.Dispatcher

	httpHandler      http.Handler
	httpPrefix       string
	health           Health
	websocketHandler http.Handler
	websocketPrefix  string
	connections      *connections

	mutex    sync.Mutex
	listener net.Listener
	server   *http.Server
}

// NewServer returns a server for the address (host:port).
func NewServer(address string, timeouts Timeouts, dispatcher *dispatch.Dispatcher) *Server {
	return &Server{address: address, timeouts: timeouts, dispatcher: dispatcher, connections: newConnections()}
}

// EnableHTTP serves JSON-RPC over HTTP on the server, and the head on
// /health.
func (server *Server) EnableHTTP(router Router, config HTTPConfig, health Health) {
	server.httpHandler = newHTTPHandler(router, config, server.timeouts.Write)
	server.httpPrefix = config.PathPrefix
	server.health = health
}

// EnableWebSocket serves JSON-RPC over WebSocket on the server, with
// subscriptions.
func (server *Server) EnableWebSocket(router Router, config WebSocketConfig) {
	server.websocketHandler = newWebSocketEndpoint(router, config, server.dispatcher, server.connections)
	server.websocketPrefix = config.PathPrefix
}

// Start listens and serves.
func (server *Server) Start() error {
	listener, err := net.Listen("tcp", server.address)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           server,
		ReadTimeout:       server.timeouts.Read,
		ReadHeaderTimeout: server.timeouts.ReadHeader,
		WriteTimeout:      server.timeouts.Write,
		IdleTimeout:       server.timeouts.Idle,
		ConnState:         server.countConnections,
	}
	httpServer.Protocols = new(http.Protocols)
	httpServer.Protocols.SetHTTP1(true)
	httpServer.Protocols.SetUnencryptedHTTP2(true)

	server.mutex.Lock()
	server.listener = listener
	server.server = httpServer
	server.mutex.Unlock()

	started := server.dispatcher.Run("http server", func(context.Context) {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("RPC server failed", "address", listener.Addr(), "err", err)
		}
	})
	if !started {
		_ = listener.Close()
		return dispatch.ErrStopping
	}
	if server.httpHandler != nil {
		log.Info("HTTP server started", "endpoint", listener.Addr(), "prefix", server.httpPrefix)
	}
	if server.websocketHandler != nil {
		log.Info("WebSocket enabled", "url", "ws://"+listener.Addr().String()+server.websocketPrefix)
	}
	return nil
}

// Address returns the address the server listens on, with the port the
// system chose for port 0.
func (server *Server) Address() string {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	if server.listener == nil {
		return server.address
	}
	return server.listener.Addr().String()
}

// shutdownTimeout is how long Stop lets the requests in flight finish before
// it closes their connections, as go-ethereum's node does.
const shutdownTimeout = 5 * time.Second

// Stop stops listening, lets the requests in flight finish for a while and
// closes the connections; the WebSocket connections, which net/http no
// longer tracks once upgraded, are ended and waited for.
func (server *Server) Stop() {
	server.mutex.Lock()
	httpServer := server.server
	server.server = nil
	server.mutex.Unlock()
	if httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Warn("HTTP server graceful shutdown timed out")
		_ = httpServer.Close()
	}
	server.connections.stop()
	log.Info("HTTP server stopped", "endpoint", server.Address())
}

func (server *Server) countConnections(_ net.Conn, state http.ConnState) {
	connections := server.dispatcher.Metrics().Connections.WithLabelValues(HTTP)
	switch state {
	case http.StateNew:
		connections.Inc()
	case http.StateClosed, http.StateHijacked:
		connections.Dec()
	}
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if server.websocketHandler != nil && isWebsocket(request) {
		if checkPath(request, server.websocketPrefix) {
			server.websocketHandler.ServeHTTP(response, request)
		}
		return
	}
	if server.httpHandler != nil {
		if request.URL.Path == "/health" && server.health != nil {
			server.serveHealth(response)
			return
		}
		if checkPath(request, server.httpPrefix) {
			server.httpHandler.ServeHTTP(response, request)
			return
		}
	}
	response.WriteHeader(http.StatusNotFound)
}

// serveHealth reports the served head and how far it lags the committed one.
func (server *Server) serveHealth(response http.ResponseWriter) {
	served, committed := server.health.Health()
	response.Header().Set("content-type", contentType)
	_ = json.NewEncoder(response).Encode(map[string]uint64{
		"head":      served,
		"committed": committed,
		"lag":       committed - served,
	})
}

// isWebsocket tells whether a request asks for an upgrade to WebSocket.
func isWebsocket(request *http.Request) bool {
	return strings.EqualFold(request.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(request.Header.Get("Connection")), "upgrade")
}

// checkPath tells whether the path of a request is under the prefix; without
// a prefix it must be the root.
func checkPath(request *http.Request, prefix string) bool {
	if prefix == "" {
		return request.URL.Path == "/"
	}
	return len(request.URL.Path) >= len(prefix) && request.URL.Path[:len(prefix)] == prefix
}
