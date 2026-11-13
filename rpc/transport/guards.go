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
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// virtualHostHandler refuses a request whose Host header names a host the
// operator did not allow, against DNS rebinding. An IP address and a request
// without a Host are served.
type virtualHostHandler struct {
	hosts map[string]bool
	next  http.Handler
}

func newVirtualHostHandler(hosts []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, host := range hosts {
		allowed[strings.ToLower(host)] = true
	}
	return &virtualHostHandler{hosts: allowed, next: next}
}

func (h *virtualHostHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Host == "" {
		h.next.ServeHTTP(response, request)
		return
	}
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host // no port, or too many colons
	}
	if net.ParseIP(host) != nil || h.hosts["*"] || h.hosts[host] {
		h.next.ServeHTTP(response, request)
		return
	}
	http.Error(response, "invalid host specified", http.StatusForbidden)
}

var gzipWriters = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// gzipResponseWriter compresses a response, unless the handler asked for its
// bytes as they are with Transfer-Encoding identity.
type gzipResponseWriter struct {
	response http.ResponseWriter
	gzip     *gzip.Writer
	// contentLength is the length of the uncompressed response, if known.
	contentLength uint64
	hasLength     bool
	written       uint64
	initialized   bool
}

// initialize decides, just before the headers are written, whether to
// compress.
func (w *gzipResponseWriter) initialize() {
	if w.initialized {
		return
	}
	w.initialized = true
	header := w.response.Header()
	if length, err := strconv.ParseUint(header.Get("content-length"), 10, 64); err == nil {
		w.hasLength = true
		w.contentLength = length
	}
	if header.Get("transfer-encoding") == "identity" {
		return
	}
	w.gzip = gzipWriters.Get().(*gzip.Writer)
	w.gzip.Reset(w.response)
	header.Del("content-length")
	header.Set("content-encoding", "gzip")
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.response.Header()
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.initialize()
	w.response.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	w.initialize()
	if w.gzip == nil {
		return w.response.Write(data)
	}
	written, err := w.gzip.Write(data)
	w.written += uint64(written)
	if w.hasLength && w.written >= w.contentLength {
		// The whole response is written: close the stream, so that the
		// client sees its end even if the response is flushed now.
		err = w.gzip.Close()
	}
	return written, err
}

func (w *gzipResponseWriter) Flush() {
	if w.gzip != nil {
		_ = w.gzip.Flush()
	}
	if flusher, ok := w.response.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipResponseWriter) close() {
	if w.gzip == nil {
		return
	}
	_ = w.gzip.Close()
	gzipWriters.Put(w.gzip)
	w.gzip = nil
}

// newGzipHandler compresses the responses to clients that accept gzip.
func newGzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(response, request)
			return
		}
		wrapper := &gzipResponseWriter{response: response}
		defer wrapper.close()
		next.ServeHTTP(wrapper, request)
	})
}
