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
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Health reports the served head for the /health endpoint.
type Health func() any

// Server holds the listeners of the endpoints.
type Server struct {
	d      *dispatch.Dispatcher
	router Router
	cfg    Config
	health Health

	mu      sync.Mutex
	servers []*endpoint
	ipc     net.Listener
}

// endpoint is one HTTP server: the HTTP endpoint, the WebSocket endpoint, or
// both when they share a port.
type endpoint struct {
	server   *http.Server
	listener net.Listener
	http     bool
	ws       bool
	wsPrefix string
}

// NewServer creates the endpoints of a configuration; Start opens them.
func NewServer(d *dispatch.Dispatcher, router Router, cfg Config, health Health) *Server {
	return &Server{d: d, router: router, cfg: cfg, health: health}
}

// Start opens the endpoints. WebSocket shares the HTTP server when the ports
// match, whatever the hosts, and the shared server listens on the WebSocket
// host, as go-ethereum's node does.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg.IPC != "" {
		l, err := listenIPC(s.cfg.IPC)
		if err != nil {
			return fmt.Errorf("opening IPC endpoint %s: %w", s.cfg.IPC, err)
		}
		s.ipc = l
		f := frames{router: s.router, kind: IPC, limits: s.cfg.Limits}
		s.d.Go("ipc accept", func(ctx context.Context) { serveIPC(ctx, s.d, f, l) })
		log.Info("IPC endpoint opened", "url", s.cfg.IPC)
	}

	httpCfg, wsCfg := s.cfg.HTTP, s.cfg.WS
	if httpCfg != nil {
		host := httpCfg.Host
		shared := wsCfg != nil && wsCfg.Port == httpCfg.Port
		if shared {
			host = wsCfg.Host
		}
		var ws *WSConfig
		if shared {
			ws = wsCfg
		}
		if err := s.listen(host, httpCfg.Port, httpCfg, ws); err != nil {
			return err
		}
		if shared {
			wsCfg = nil
		}
	}
	if wsCfg != nil {
		if err := s.listen(wsCfg.Host, wsCfg.Port, nil, wsCfg); err != nil {
			return err
		}
	}
	return nil
}

// listen opens one HTTP server for an HTTP endpoint, a WebSocket endpoint, or
// both.
func (s *Server) listen(host string, port int, httpCfg *HTTPConfig, wsCfg *WSConfig) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var (
		ws       http.Handler
		rpc      http.Handler
		wsPrefix string
		rpcPfx   string
		mux      = http.NewServeMux()
	)
	if wsCfg != nil {
		ws = newWSHandler(s.d, frames{router: s.router, kind: WS, limits: s.cfg.Limits}, wsCfg.Origins)
		wsPrefix = wsCfg.Prefix
	}
	server := &http.Server{}
	if httpCfg != nil {
		timeouts := httpCfg.Timeouts.sanitized(httpCfg.FallbackTimeouts)
		server.ReadTimeout = timeouts.Read
		server.ReadHeaderTimeout = timeouts.ReadHeader
		server.WriteTimeout = timeouts.Write
		server.IdleTimeout = timeouts.Idle
		h := &httpHandler{frames: frames{router: s.router, kind: HTTP, limits: s.cfg.Limits}}
		if timeouts.Write > 0 {
			h.timeout = timeouts.Write - timeoutMargin
		}
		rpc = newGzipHandler(newVHostHandler(httpCfg.Vhosts, newCorsHandler(h, httpCfg.Cors)))
		rpcPfx = httpCfg.Prefix
		mux.HandleFunc("/health", s.serveHealth)
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ws != nil && isWebsocket(r) {
			if checkPath(r, wsPrefix) {
				ws.ServeHTTP(w, r)
			}
			return
		}
		if rpc != nil {
			if handler, pattern := mux.Handler(r); pattern != "" {
				handler.ServeHTTP(w, r)
				return
			}
			if checkPath(r, rpcPfx) {
				rpc.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	e := &endpoint{server: server, listener: l, http: httpCfg != nil, ws: wsCfg != nil, wsPrefix: wsPrefix}
	s.servers = append(s.servers, e)
	s.d.Go("http serve", func(context.Context) {
		if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn("HTTP server failed", "endpoint", l.Addr(), "err", err)
		}
	})
	if e.ws {
		log.Info("WebSocket enabled", "url", fmt.Sprintf("ws://%v%s", l.Addr(), wsPrefix))
	}
	if e.http {
		log.Info("HTTP server started", "endpoint", l.Addr(), "prefix", rpcPfx)
	}
	return nil
}

func (s *Server) serveHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	enc, err := json.Marshal(s.health())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(enc)
}

// Close stops accepting connections and requests. HTTP requests in flight
// and stream connections end when the dispatcher stops.
func (s *Server) Close(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ipc != nil {
		_ = s.ipc.Close()
		s.ipc = nil
	}
	for _, e := range s.servers {
		if err := e.server.Shutdown(ctx); err != nil {
			_ = e.server.Close()
		}
	}
	s.servers = nil
}

// HTTPEndpoint is the address of the HTTP endpoint, empty if there is none.
func (s *Server) HTTPEndpoint() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.servers {
		if e.http {
			return "http://" + e.listener.Addr().String()
		}
	}
	return ""
}

// WSEndpoint is the address of the WebSocket endpoint, empty if there is
// none.
func (s *Server) WSEndpoint() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.servers {
		if e.ws {
			return "ws://" + e.listener.Addr().String() + e.wsPrefix
		}
	}
	return ""
}

// IPCEndpoint is the path of the IPC socket, empty if there is none.
func (s *Server) IPCEndpoint() string {
	return s.cfg.IPC
}
