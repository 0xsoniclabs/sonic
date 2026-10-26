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
	"compress/gzip"
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

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

var acceptedContentTypes = []string{"application/json", "application/json-rpc", "application/jsonrequest"}

// httpHandler serves JSON-RPC over HTTP, one frame per request.
type httpHandler struct {
	frames  frames
	timeout time.Duration // the write timeout less the margin; zero: none
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// An empty GET is the health check of load balancers.
	if r.Method == http.MethodGet && r.ContentLength == 0 && r.URL.RawQuery == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if code, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	w.Header().Set("content-type", "application/json")

	var raw json.RawMessage
	dec := json.NewDecoder(io.LimitReader(r.Body, httpBodyLimit))
	dec.UseNumber()
	hw := &httpWriter{w: w}
	if err := dec.Decode(&raw); err != nil {
		if msg := readErrorMessage(err); msg != "" {
			hw.write((&response{id: nullID, err: &parseErr{msg}}).encode(), true)
		}
		return
	}
	h.frames.serveFrame(r.Context(), raw, hw, h.timeout, nil)
	hw.wait()
}

type parseErr struct{ msg string }

func (e *parseErr) Error() string  { return e.msg }
func (e *parseErr) ErrorCode() int { return dispatch.CodeParse }

// validateRequest refuses what go-ethereum's server refuses before reading.
func validateRequest(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > httpBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Errorf("content length too large (%d>%d)", r.ContentLength, httpBodyLimit)
	}
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
	return http.StatusUnsupportedMediaType, errors.New("invalid content type, only application/json is supported")
}

// httpWriter writes the answer of one request once. An answer of the
// envelope, such as a timeout, is written uncompressed with its length and
// flushed, so that it reaches the client before the server's write timeout.
type httpWriter struct {
	mu    sync.Mutex
	w     http.ResponseWriter
	wrote bool
}

func (hw *httpWriter) write(msg []byte, transportError bool) {
	hw.mu.Lock()
	defer hw.mu.Unlock()
	if hw.wrote {
		return
	}
	hw.wrote = true
	if transportError {
		hw.w.Header().Set("content-length", strconv.Itoa(len(msg)))
		hw.w.Header().Set("transfer-encoding", "identity")
		_, _ = hw.w.Write(msg)
		if f, ok := hw.w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	_, _ = hw.w.Write(append(msg, '\n'))
}

// wait synchronises with a write of the timer, so that the server does not
// finish the response while the timer writes it.
func (hw *httpWriter) wait() {
	hw.mu.Lock()
	defer hw.mu.Unlock()
	hw.wrote = true
}

// newCorsHandler answers cross-origin requests for the allowed origins, as
// go-ethereum's handler does with rs/cors: POST and GET, any header, a
// max-age of 600 seconds, and "*" for every origin.
func newCorsHandler(next http.Handler, allowedOrigins []string) http.Handler {
	if len(allowedOrigins) == 0 {
		return next
	}
	c := newCors(allowedOrigins)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			c.preflight(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		c.actual(w, r)
		next.ServeHTTP(w, r)
	})
}

type cors struct {
	all       bool
	origins   []string
	wildcards [][2]string
}

func newCors(allowed []string) *cors {
	c := &cors{}
	for _, origin := range allowed {
		origin = strings.ToLower(origin)
		if origin == "*" {
			return &cors{all: true}
		}
		if i := strings.IndexByte(origin, '*'); i >= 0 {
			c.wildcards = append(c.wildcards, [2]string{origin[:i], origin[i+1:]})
		} else {
			c.origins = append(c.origins, origin)
		}
	}
	return c
}

func (c *cors) allows(origin string) bool {
	if c.all {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.origins {
		if o == origin {
			return true
		}
	}
	for _, w := range c.wildcards {
		if len(origin) >= len(w[0])+len(w[1]) && strings.HasPrefix(origin, w[0]) && strings.HasSuffix(origin, w[1]) {
			return true
		}
	}
	return false
}

func corsMethodAllowed(method string) bool {
	return method == http.MethodOptions || method == http.MethodPost || method == http.MethodGet
}

func (c *cors) allowOrigin(headers http.Header, r *http.Request) {
	if c.all {
		headers["Access-Control-Allow-Origin"] = []string{"*"}
	} else {
		headers["Access-Control-Allow-Origin"] = r.Header["Origin"]
	}
}

func (c *cors) preflight(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers.Add("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	origin := r.Header.Get("Origin")
	if origin == "" || !c.allows(origin) || !corsMethodAllowed(r.Header.Get("Access-Control-Request-Method")) {
		return
	}
	c.allowOrigin(headers, r)
	headers["Access-Control-Allow-Methods"] = r.Header["Access-Control-Request-Method"]
	if reqHeaders, found := r.Header["Access-Control-Request-Headers"]; found && len(reqHeaders[0]) > 0 {
		headers["Access-Control-Allow-Headers"] = reqHeaders
	}
	headers["Access-Control-Max-Age"] = []string{"600"}
}

func (c *cors) actual(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers.Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin == "" || !c.allows(origin) || !corsMethodAllowed(r.Method) {
		return
	}
	c.allowOrigin(headers, r)
}

// newVHostHandler refuses a request whose Host header names a host not in the
// list, against DNS rebinding; an IP address and "*" pass, as in go-ethereum.
func newVHostHandler(vhosts []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{})
	for _, h := range vhosts {
		allowed[strings.ToLower(h)] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" {
			next.ServeHTTP(w, r)
			return
		}
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if net.ParseIP(host) != nil {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := allowed["*"]; ok {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := allowed[host]; ok {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "invalid host specified", http.StatusForbidden)
	})
}

var gzipPool = sync.Pool{New: func() any { return gzip.NewWriter(io.Discard) }}

// gzipWriter compresses a response for a client that accepts it, unless the
// response set "transfer-encoding: identity".
type gzipWriter struct {
	resp          http.ResponseWriter
	gz            *gzip.Writer
	contentLength uint64
	written       uint64
	hasLength     bool
	inited        bool
}

func (w *gzipWriter) init() {
	if w.inited {
		return
	}
	w.inited = true
	hdr := w.resp.Header()
	if length := hdr.Get("content-length"); len(length) > 0 {
		if n, err := strconv.ParseUint(length, 10, 64); err == nil {
			w.hasLength = true
			w.contentLength = n
		}
	}
	if hdr.Get("transfer-encoding") != "identity" {
		w.gz = gzipPool.Get().(*gzip.Writer)
		w.gz.Reset(w.resp)
		hdr.Del("content-length")
		hdr.Set("content-encoding", "gzip")
	}
}

func (w *gzipWriter) Header() http.Header { return w.resp.Header() }

func (w *gzipWriter) WriteHeader(status int) {
	w.init()
	w.resp.WriteHeader(status)
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	w.init()
	if w.gz == nil {
		return w.resp.Write(b)
	}
	n, err := w.gz.Write(b)
	w.written += uint64(n)
	if w.hasLength && w.written >= w.contentLength {
		err = w.gz.Close()
	}
	return n, err
}

func (w *gzipWriter) Flush() {
	if w.gz != nil {
		_ = w.gz.Flush()
	}
	if f, ok := w.resp.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *gzipWriter) close() {
	if w.gz == nil {
		return
	}
	_ = w.gz.Close()
	gzipPool.Put(w.gz)
	w.gz = nil
}

func newGzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{resp: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

// isWebsocket reports an upgrade request.
func isWebsocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// checkPath matches a request path to a prefix; without a prefix, only the
// root is served.
func checkPath(r *http.Request, prefix string) bool {
	if prefix == "" {
		return r.URL.Path == "/"
	}
	return len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix
}
