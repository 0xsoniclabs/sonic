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
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/rs/cors"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	// DefaultBodyLimit is the largest accepted HTTP request body.
	DefaultBodyLimit = 5 * 1024 * 1024
	contentType      = "application/json"
)

// https://www.jsonrpc.org/historical/json-rpc-over-http.html#id13
var acceptedContentTypes = []string{contentType, "application/json-rpc", "application/jsonrequest"}

// Timeouts are the timeouts of the HTTP server.
type Timeouts struct {
	Read       time.Duration
	ReadHeader time.Duration
	Write      time.Duration
	Idle       time.Duration
}

// DefaultTimeouts are those of the node's HTTP server.
var DefaultTimeouts = Timeouts{
	Read:       30 * time.Second,
	ReadHeader: 30 * time.Second,
	Write:      30 * time.Second,
	Idle:       120 * time.Second,
}

func (t *Timeouts) sanitize() {
	fix := func(name string, v *time.Duration, def time.Duration) {
		if *v < time.Second {
			log.Warn("Sanitizing invalid HTTP "+name+" timeout", "provided", *v, "updated", def)
			*v = def
		}
	}
	fix("read", &t.Read, DefaultTimeouts.Read)
	fix("read header", &t.ReadHeader, DefaultTimeouts.ReadHeader)
	fix("write", &t.Write, DefaultTimeouts.Write)
	fix("idle", &t.Idle, DefaultTimeouts.Idle)
}

// Health is the answer of the /health endpoint.
type Health struct {
	// Head is the number of the block served as latest.
	Head uint64 `json:"head"`
	// Committed is the number of the newest committed block.
	Committed uint64 `json:"committed"`
	// Lag is the number of committed blocks the head is behind.
	Lag uint64 `json:"lag"`
}

// HTTPConfig configures the HTTP transport.
type HTTPConfig struct {
	Host    string
	Port    int
	Prefix  string
	Cors    []string
	VHosts  []string
	Modules []string

	BodyLimit        int
	Timeouts         Timeouts
	BatchLimit       int
	BatchResponseMax int

	// DisableRPC makes the listener serve WebSocket only.
	DisableRPC bool

	// WS, when set, makes the listener serve WebSocket connections on the path
	// WSPrefix.
	WS       *WSConfig
	WSPrefix string

	// Health reports head and lag for /health; nil disables the endpoint.
	Health func() Health
}

// HTTPServer serves JSON-RPC over HTTP.
type HTTPServer struct {
	cfg      HTTPConfig
	d        *dispatch.Dispatcher
	endpoint *Endpoint
	metrics  *dispatch.Metrics

	front *Front
	ws    *WSHandler

	mu       sync.Mutex
	srv      *http.Server
	listener net.Listener
	done     chan struct{}
}

// NewHTTPServer creates an HTTP transport that serves through the front.
func NewHTTPServer(cfg HTTPConfig, front *Front) (*HTTPServer, error) {
	if err := validatePrefix("HTTP", cfg.Prefix); err != nil {
		return nil, err
	}
	if cfg.BodyLimit <= 0 {
		cfg.BodyLimit = DefaultBodyLimit
	}
	if cfg.Timeouts == (Timeouts{}) {
		cfg.Timeouts = DefaultTimeouts
	}
	cfg.Timeouts.sanitize()
	var ws *WSHandler
	if cfg.WS != nil {
		ws = NewWSHandler(*cfg.WS, front)
	}
	return &HTTPServer{
		cfg:   cfg,
		d:     front.Dispatcher(),
		front: front,
		ws:    ws,
		endpoint: NewEndpoint(EndpointConfig{
			Name:             "http",
			Modules:          cfg.Modules,
			BatchLimit:       cfg.BatchLimit,
			BatchResponseMax: cfg.BatchResponseMax,
		}, front),
		metrics: front.Dispatcher().Metrics(),
	}, nil
}

// validatePrefix checks the path prefix option.
func validatePrefix(what, path string) error {
	if path == "" {
		return nil
	}
	if path[0] != '/' {
		return fmt.Errorf(`%s RPC path prefix %q does not contain leading "/"`, what, path)
	}
	if strings.ContainsAny(path, "?#") {
		return fmt.Errorf("%s RPC path prefix %q contains URL meta-characters", what, path)
	}
	return nil
}

// checkPath checks whether a request URL matches a path prefix.
func checkPath(r *http.Request, path string) bool {
	if path == "" {
		return r.URL.Path == "/"
	}
	return len(r.URL.Path) >= len(path) && r.URL.Path[:len(path)] == path
}

// Start opens the listener and serves in the background.
func (s *HTTPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port)))
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           s.handler(),
		ReadTimeout:       s.cfg.Timeouts.Read,
		ReadHeaderTimeout: s.cfg.Timeouts.ReadHeader,
		WriteTimeout:      s.cfg.Timeouts.Write,
		IdleTimeout:       s.cfg.Timeouts.Idle,
		ConnState:         s.trackConnections,
	}
	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)
	srv.Protocols.SetUnencryptedHTTP2(true)
	done := make(chan struct{})
	if !s.d.Go("http-serve", func(context.Context) {
		defer close(done)
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed", "err", err)
		}
	}) {
		_ = listener.Close()
		return errors.New("dispatcher is stopping")
	}
	s.srv, s.listener, s.done = srv, listener, done
	log.Info("HTTP server started", "endpoint", listener.Addr(), "prefix", s.cfg.Prefix,
		"cors", strings.Join(s.cfg.Cors, ","), "vhosts", strings.Join(s.cfg.VHosts, ","))
	return nil
}

// Addr returns the address the server listens on, "" before Start.
func (s *HTTPServer) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Stop stops accepting connections and waits for the requests in flight until
// the context ends.
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	srv, done := s.srv, s.done
	s.srv, s.listener = nil, nil
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Shutdown(ctx)
	if err != nil {
		_ = srv.Close()
	}
	<-done
	// Hijacked connections are not waited for by the server.
	if s.ws != nil {
		s.ws.Close()
	}
	return err
}

func (s *HTTPServer) trackConnections(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		s.metrics.Connections.WithLabelValues("http").Inc()
	case http.StateHijacked, http.StateClosed:
		s.metrics.Connections.WithLabelValues("http").Dec()
	}
}

// handler builds the handler chain: WebSocket upgrades first, then gzip,
// virtual host check and CORS around the RPC handler.
func (s *HTTPServer) handler() http.Handler {
	var rpcHandler http.Handler = http.HandlerFunc(s.serveRPC)
	rpcHandler = newCORSHandler(rpcHandler, s.cfg.Cors)
	rpcHandler = newVHostHandler(s.cfg.VHosts, rpcHandler)
	rpcHandler = newGzipHandler(rpcHandler)

	var wsHandler http.Handler
	if s.ws != nil {
		wsHandler = s.ws
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Health != nil && r.Method == http.MethodGet && r.URL.Path == "/health" {
			s.serveHealth(w)
			return
		}
		if isWebsocket(r) {
			if wsHandler != nil && checkPath(r, s.cfg.WSPrefix) {
				wsHandler.ServeHTTP(w, r)
			}
			return
		}
		if !s.cfg.DisableRPC && checkPath(r, s.cfg.Prefix) {
			rpcHandler.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

func newCORSHandler(next http.Handler, allowedOrigins []string) http.Handler {
	// CORS support is disabled unless the operator configures it.
	if len(allowedOrigins) == 0 {
		return next
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet},
		AllowedHeaders: []string{"*"},
		MaxAge:         600,
	})
	return c.Handler(next)
}

func (s *HTTPServer) serveHealth(w http.ResponseWriter) {
	h := s.cfg.Health()
	w.Header().Set("content-type", contentType)
	_ = json.NewEncoder(w).Encode(h)
}

// serveRPC serves JSON-RPC requests over HTTP.
func (s *HTTPServer) serveRPC(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS).
	if r.Method == http.MethodGet && r.ContentLength == 0 && r.URL.RawQuery == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if code, err := s.validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	w.Header().Set("content-type", contentType)
	body := io.LimitReader(r.Body, int64(s.cfg.BodyLimit))
	dec := json.NewDecoder(body)
	dec.UseNumber()
	out := &httpWriter{w: w}

	ctx := r.Context()
	// The response must go out before the server cuts the connection.
	if wt := s.cfg.Timeouts.Write; wt > 0 {
		ctx = withRequestTimeout(ctx, wt-100*time.Millisecond)
	}

	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		if msg := messageForReadError(err); msg != "" {
			out.write(ctx, errorMessage(dispatch.NewError(dispatch.CodeParse, "%s", msg)), true)
		}
		return
	}
	msgs, batch := parseMessages(raw)
	if batch {
		s.endpoint.handleBatch(ctx, msgs, out)
	} else {
		s.endpoint.handleMsg(ctx, msgs[0], out)
	}
}

func messageForReadError(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "read timeout"
		}
		return "read error"
	} else if err != io.EOF {
		return "parse error"
	}
	return ""
}

// validateRequest returns a non-zero response code and error if the request is
// invalid.
func (s *HTTPServer) validateRequest(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > int64(s.cfg.BodyLimit) {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, s.cfg.BodyLimit)
		return http.StatusRequestEntityTooLarge, err
	}
	// Allow OPTIONS (regardless of content-type).
	if r.Method == http.MethodOptions {
		return 0, nil
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("content-type")); err == nil {
		for _, accepted := range acceptedContentTypes {
			if accepted == mt {
				return 0, nil
			}
		}
	}
	return http.StatusUnsupportedMediaType, fmt.Errorf("invalid content type, only %s is supported", contentType)
}

// httpWriter writes the response of a request.
type httpWriter struct {
	w http.ResponseWriter
}

func (h *httpWriter) write(_ context.Context, v any, isError bool) error {
	if !isError {
		return json.NewEncoder(h.w).Encode(v)
	}
	// An error response may have to go out right before the write deadline: it
	// is flushed, carries its length and is exempt from compression and
	// chunking.
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.w.Header().Set("content-length", strconv.Itoa(len(data)))
	h.w.Header().Set("transfer-encoding", "identity")
	_, err = h.w.Write(data)
	if f, ok := h.w.(http.Flusher); ok {
		f.Flush()
	}
	return err
}
