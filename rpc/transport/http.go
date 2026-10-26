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
	"net/http"
	"strconv"
	"time"

	"github.com/rs/cors"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// The limits go-ethereum's server applies to the size of a request.
const (
	httpBodyLimit         = 5 * 1024 * 1024
	websocketMessageLimit = 32 * 1024 * 1024
)

// writeMargin is how long before the write timeout of the server a call still
// running is answered, so that the answer goes out before the server cuts the
// connection.
const writeMargin = 100 * time.Millisecond

const contentType = "application/json"

var acceptedContentTypes = []string{contentType, "application/json-rpc", "application/jsonrequest"}

// HTTPConfig is the HTTP endpoint as the node's configuration holds it.
type HTTPConfig struct {
	PathPrefix   string
	CORSOrigins  []string
	VirtualHosts []string
	Batch        BatchLimits
}

// httpEndpoint serves JSON-RPC over HTTP requests.
type httpEndpoint struct {
	served       connection
	writeTimeout time.Duration
}

// newHTTPHandler returns the handler of the HTTP endpoint with go-ethereum's
// stack around it: compression outermost, then the check of the Host header,
// then CORS.
func newHTTPHandler(router Router, config HTTPConfig, serverWriteTimeout time.Duration) http.Handler {
	endpoint := &httpEndpoint{
		served: connection{router: router, transport: HTTP, limits: config.Batch},
	}
	if serverWriteTimeout > 0 {
		endpoint.writeTimeout = serverWriteTimeout - writeMargin
	}
	return newGzipHandler(newVirtualHostHandler(config.VirtualHosts, newCORSHandler(endpoint, config.CORSOrigins)))
}

func newCORSHandler(next http.Handler, allowedOrigins []string) http.Handler {
	if len(allowedOrigins) == 0 {
		return next
	}
	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet},
		AllowedHeaders: []string{"*"},
		MaxAge:         600,
	}).Handler(next)
}

func (endpoint *httpEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {
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
	write := httpWriter(response)

	decoder := json.NewDecoder(io.LimitReader(request.Body, httpBodyLimit))
	decoder.UseNumber()
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		if text := readErrorText(err); text != "" {
			write(errorMessage(&dispatch.Error{Code: dispatch.CodeParseError, Message: text}), true)
		}
		return
	}
	messages, isBatch := parseMessages(raw)
	if isBatch {
		endpoint.served.serveBatch(request.Context(), messages, endpoint.writeTimeout, write)
		return
	}
	endpoint.served.serveSingle(request.Context(), messages[0], endpoint.writeTimeout, write)
}

// httpWriter writes a response. An error response, which may be written at
// the deadline, goes out at once with its length and without compression or
// chunking, so that it arrives whole before the server cuts the connection.
func httpWriter(response http.ResponseWriter) writer {
	return func(answer any, isError bool) {
		if !isError {
			_ = json.NewEncoder(response).Encode(answer)
			return
		}
		encoded, err := json.Marshal(answer)
		if err != nil {
			return
		}
		response.Header().Set("content-length", strconv.Itoa(len(encoded)))
		response.Header().Set("transfer-encoding", "identity")
		_, _ = response.Write(encoded)
		if flusher, ok := response.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// validateRequest refuses what go-ethereum's server refuses, with its status
// codes and texts.
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
	if mediaType, _, err := mime.ParseMediaType(request.Header.Get("content-type")); err == nil {
		for _, accepted := range acceptedContentTypes {
			if accepted == mediaType {
				return 0, nil
			}
		}
	}
	return http.StatusUnsupportedMediaType, fmt.Errorf("invalid content type, only %s is supported", contentType)
}
