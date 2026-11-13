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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/log"
)

// metadataNamespace is the namespace of rpc_modules, which every transport
// serves, as go-ethereum's servers do.
const metadataNamespace = "rpc"

// forwardedNamespaces are the namespaces whose methods the module does not
// serve itself and IPC forwards to the node's own handler: the node's
// administration and its profiling.
var forwardedNamespaces = []string{"admin", "debug"}

// unforwarded are the methods of the node's handler that are not forwarded:
// the module owns every endpoint, so the node starts and stops none.
var unforwarded = []string{
	"admin_startHTTP", "admin_startRPC", "admin_startWS",
	"admin_stopHTTP", "admin_stopRPC", "admin_stopWS",
}

// Front routes the calls of one transport: the methods of the namespaces it
// enables to the dispatcher and, on IPC, the node's administration and
// profiling to the node's own handler.
type Front struct {
	dispatcher *dispatch.Dispatcher
	// enabled lists the namespaces served; nil serves all.
	enabled map[string]bool
	// node is the node's handler, which only IPC forwards to.
	node http.Handler
	// modules is what rpc_modules answers on this transport.
	modules map[string]string
}

// NewFront routes the namespaces of an API list, as --http.api and --ws.api
// give them; an empty list enables all. The node's namespaces in the list
// are not served on the transport, and are ignored with a warning.
func NewFront(dispatcher *dispatch.Dispatcher, list []string, flag string) *Front {
	front := &Front{dispatcher: dispatcher, modules: map[string]string{}}
	served := servedNamespaces(dispatcher)
	if len(list) > 0 {
		front.enabled = map[string]bool{metadataNamespace: true}
	}
	for _, namespace := range list {
		switch {
		case namespace == "ftm":
			log.Warn(fmt.Sprintf("The 'ftm' API is not served by the RPC module, use 'eth' instead (--%s).", flag))
		case namespace == "admin":
			log.Warn(fmt.Sprintf("The 'admin' API is served on IPC only by the RPC module (--%s).", flag))
		case !slices.Contains(served, namespace):
			log.Error("Unavailable modules in API list", "flag", flag, "unavailable", namespace, "available", served)
		default:
			front.enabled[namespace] = true
		}
	}
	for _, namespace := range served {
		if front.serves(namespace) {
			front.modules[namespace] = "1.0"
		}
	}
	return front
}

// NewIPCFront routes every namespace of the module, and forwards the node's
// administration and profiling to its handler.
func NewIPCFront(dispatcher *dispatch.Dispatcher, node http.Handler) *Front {
	front := &Front{dispatcher: dispatcher, node: node, modules: map[string]string{}}
	for _, namespace := range servedNamespaces(dispatcher) {
		front.modules[namespace] = "1.0"
	}
	for namespace := range front.nodeModules() {
		front.modules[namespace] = "1.0"
	}
	return front
}

// servedNamespaces lists the namespaces of the dispatcher's methods.
func servedNamespaces(dispatcher *dispatch.Dispatcher) []string {
	var namespaces []string
	for _, method := range dispatcher.Methods() {
		namespace, _, _ := strings.Cut(method, "_")
		if !slices.Contains(namespaces, namespace) {
			namespaces = append(namespaces, namespace)
		}
	}
	return namespaces
}

// nodeModules asks the node's handler for its namespaces.
func (f *Front) nodeModules() map[string]string {
	call := &message{Version: version, ID: json.RawMessage("1"), Method: "rpc_modules"}
	var modules map[string]string
	if answer := f.forward(context.Background(), call); answer.Error == nil {
		_ = json.Unmarshal(answer.Result, &modules)
	}
	return modules
}

func (f *Front) serves(namespace string) bool {
	return f.enabled == nil || f.enabled[namespace]
}

// forwards tells whether a method goes to the node's handler.
func (f *Front) forwards(msg *message) bool {
	return f.node != nil &&
		slices.Contains(forwardedNamespaces, msg.namespace()) &&
		!f.dispatcher.Serves(msg.Method) &&
		!slices.Contains(unforwarded, msg.Method)
}

// call answers a call or a notification, as go-ethereum's server checks it:
// subscriptions first, then the length of the method's name.
func (f *Front) call(ctx context.Context, msg *message) *message {
	switch {
	case strings.HasSuffix(msg.Method, subscribeMethodSuffix):
		return msg.errorResponse(errNotificationsUnsupported)
	case len(msg.Method) > maxMethodNameLength:
		return msg.errorResponse(invalidRequest(fmt.Sprintf("method name too long: %d > %d", len(msg.Method), maxMethodNameLength)))
	case f.forwards(msg):
		return f.forward(ctx, msg)
	case f.serves(msg.namespace()):
		return msg.answer(f.dispatcher.Serve(context.WithValue(ctx, modulesKey{}, f.modules), msg.Method, msg.Params))
	default:
		return msg.errorResponse(dispatch.MethodNotFound(msg.Method))
	}
}

// forward has the node's handler answer a call, as an HTTP request in
// process. The handler decodes the call, recovers its panics and bounds it
// with its own limit; the dispatcher does not count it.
func (f *Front) forward(ctx context.Context, msg *message) *message {
	body, err := json.Marshal(&message{Version: version, ID: msg.ID, Method: msg.Method, Params: msg.Params})
	if err != nil {
		return msg.errorResponse(&dispatch.Error{Code: dispatch.CodeInternal, Message: err.Error()})
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		return msg.errorResponse(&dispatch.Error{Code: dispatch.CodeInternal, Message: err.Error()})
	}
	request.Header.Set("content-type", contentType)
	response := &bufferedResponse{header: http.Header{}}
	f.node.ServeHTTP(response, request)
	if msg.ID == nil {
		return nil
	}
	answer := new(message)
	if err := json.Unmarshal(response.body.Bytes(), answer); err != nil {
		return msg.errorResponse(&dispatch.Error{Code: dispatch.CodeInternal, Message: "the node's handler answered no message"})
	}
	return answer
}

// bufferedResponse holds the response of the node's handler.
type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
}

func (r *bufferedResponse) Header() http.Header            { return r.header }
func (r *bufferedResponse) Write(data []byte) (int, error) { return r.body.Write(data) }
func (r *bufferedResponse) WriteHeader(int)                {}

type modulesKey struct{}

// Modules is what rpc_modules answers on the transport of a call: its
// namespaces, each with go-ethereum's version string.
func Modules(ctx context.Context) map[string]string {
	modules, _ := ctx.Value(modulesKey{}).(map[string]string)
	return modules
}
