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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/rs/cors"
)

const (
	contentType = "application/json"
	// httpBodyLimit is the largest body of an HTTP request.
	httpBodyLimit = 5 * 1024 * 1024
	// timeoutMargin is how long before the server's write timeout a call that
	// has not answered is answered "request timed out", so that the answer
	// still reaches the client.
	timeoutMargin = 100 * time.Millisecond
)

var acceptedContentTypes = []string{contentType, "application/json-rpc", "application/jsonrequest"}

// httpHandler serves JSON-RPC requests over HTTP, one request or batch per
// HTTP request, without subscriptions.
type httpHandler struct {
	front   *Front
	limits  BatchLimits
	timeout time.Duration
}

func (h *httpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// An empty GET is a health check of a load balancer.
	if request.Method == http.MethodGet && request.ContentLength == 0 && request.URL.RawQuery == "" {
		response.WriteHeader(http.StatusOK)
		return
	}
	if code, err := validateRequest(request); err != nil {
		http.Error(response, err.Error(), code)
		return
	}
	response.Header().Set("content-type", contentType)
	out := &httpWriter{response: response}
	decoder := json.NewDecoder(io.LimitReader(request.Body, httpBodyLimit))
	decoder.UseNumber()
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		if text := readErrorText(err); text != "" {
			out.write(errorMessage(invalidMessage(text)), true)
		}
		return
	}
	served := &session{front: h.front, out: out, limits: h.limits, timeout: h.timeout}
	messages, batch := parseMessages(raw)
	if batch {
		served.serveBatch(request.Context(), messages)
	} else {
		served.serveMessage(request.Context(), messages[0])
	}
}

// validateRequest answers the HTTP status and error of a request that is not
// a JSON-RPC request.
func validateRequest(request *http.Request) (int, error) {
	if request.Method == http.MethodPut || request.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if request.ContentLength > httpBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Errorf("content length too large (%d>%d)", request.ContentLength, httpBodyLimit)
	}
	if request.Method == http.MethodOptions {
		return 0, nil
	}
	if mediaType, _, err := mime.ParseMediaType(request.Header.Get("content-type")); err == nil && slices.Contains(acceptedContentTypes, mediaType) {
		return 0, nil
	}
	return http.StatusUnsupportedMediaType, fmt.Errorf("invalid content type, only %s is supported", contentType)
}

// readErrorText words a failure to read a request, empty for an empty one.
func readErrorText(err error) string {
	var netErr net.Error
	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		return "read timeout"
	case errors.As(err, &netErr):
		return "read error"
	case err != io.EOF:
		return "parse error"
	default:
		return ""
	}
}

// httpWriter writes the answer of an HTTP request.
type httpWriter struct {
	mutex    sync.Mutex
	response http.ResponseWriter
}

// write encodes an answer. An error answer is written whole with its length
// and flushed, without compression or chunks, because a timeout's answer
// must reach the client before the server's write timeout ends the response.
func (w *httpWriter) write(value any, isError bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if !isError {
		_ = json.NewEncoder(w.response).Encode(value)
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	w.response.Header().Set("content-length", strconv.Itoa(len(encoded)))
	w.response.Header().Set("transfer-encoding", "identity")
	_, _ = w.response.Write(encoded)
	if flusher, ok := w.response.(http.Flusher); ok {
		flusher.Flush()
	}
}

// newHTTPStack wraps the handler of JSON-RPC over HTTP in the checks of
// go-ethereum's HTTP server: CORS, then the virtual hosts, and compression
// outermost.
func newHTTPStack(handler http.Handler, corsOrigins, virtualHosts []string) http.Handler {
	if len(corsOrigins) > 0 {
		handler = cors.New(cors.Options{
			AllowedOrigins: corsOrigins,
			AllowedMethods: []string{http.MethodPost, http.MethodGet},
			AllowedHeaders: []string{"*"},
			MaxAge:         600,
		}).Handler(handler)
	}
	return newGzipHandler(newVirtualHostHandler(virtualHosts, handler))
}
