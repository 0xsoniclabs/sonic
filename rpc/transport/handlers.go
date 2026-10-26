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

// virtualHostHandler checks the Host header against the configured virtual
// hosts, which prevents DNS rebinding: a request to an IP address or to a
// listed name passes, any other name is refused.
type virtualHostHandler struct {
	hosts map[string]struct{}
	next  http.Handler
}

func newVirtualHostHandler(hosts []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		allowed[strings.ToLower(host)] = struct{}{}
	}
	return &virtualHostHandler{hosts: allowed, next: next}
}

func (handler *virtualHostHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// A request without a Host header comes from no browser.
	if request.Host == "" {
		handler.next.ServeHTTP(response, request)
		return
	}
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host
	}
	if net.ParseIP(host) != nil {
		handler.next.ServeHTTP(response, request)
		return
	}
	if _, wildcard := handler.hosts["*"]; wildcard {
		handler.next.ServeHTTP(response, request)
		return
	}
	if _, listed := handler.hosts[host]; listed {
		handler.next.ServeHTTP(response, request)
		return
	}
	http.Error(response, "invalid host specified", http.StatusForbidden)
}

var gzipWriters = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

// gzipResponseWriter compresses a response unless it says
// "transfer-encoding: identity", which the error responses written at the
// deadline do.
type gzipResponseWriter struct {
	response http.ResponseWriter

	compressor    *gzip.Writer
	contentLength uint64
	written       uint64
	hasLength     bool
	initialized   bool
}

// initialize runs before the headers are written and decides whether the
// response is compressed.
func (writer *gzipResponseWriter) initialize() {
	if writer.initialized {
		return
	}
	writer.initialized = true
	header := writer.response.Header()
	if length := header.Get("content-length"); len(length) > 0 {
		if parsed, err := strconv.ParseUint(length, 10, 64); err == nil {
			writer.hasLength = true
			writer.contentLength = parsed
		}
	}
	if header.Get("transfer-encoding") == "identity" {
		return
	}
	writer.compressor = gzipWriters.Get().(*gzip.Writer)
	writer.compressor.Reset(writer.response)
	header.Del("content-length")
	header.Set("content-encoding", "gzip")
}

func (writer *gzipResponseWriter) Header() http.Header {
	return writer.response.Header()
}

func (writer *gzipResponseWriter) WriteHeader(status int) {
	writer.initialize()
	writer.response.WriteHeader(status)
}

func (writer *gzipResponseWriter) Write(data []byte) (int, error) {
	writer.initialize()
	if writer.compressor == nil {
		return writer.response.Write(data)
	}
	written, err := writer.compressor.Write(data)
	writer.written += uint64(written)
	if writer.hasLength && writer.written >= writer.contentLength {
		// The whole response is written: close the stream so that the client
		// sees its end even if the response is flushed now.
		err = writer.compressor.Close()
	}
	return written, err
}

func (writer *gzipResponseWriter) Flush() {
	if writer.compressor != nil {
		_ = writer.compressor.Flush()
	}
	if flusher, ok := writer.response.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (writer *gzipResponseWriter) close() {
	if writer.compressor == nil {
		return
	}
	_ = writer.compressor.Close()
	gzipWriters.Put(writer.compressor)
	writer.compressor = nil
}

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
