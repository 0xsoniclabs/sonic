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
	"errors"
	"fmt"
	"strings"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// The names of the transports.
const (
	HTTP      = "http"
	WebSocket = "ws"
	IPC       = "ipc"
)

// Forwarder hands a call the module does not serve to the node's own
// handler.
type Forwarder interface {
	Forward(ctx context.Context, method string, parameters []json.RawMessage) (json.RawMessage, error)
}

// Front routes the calls of the transports to the dispatcher. It serves on
// each transport only the namespaces enabled there, answers rpc_modules and
// the subscription methods of any namespace as go-ethereum's server does,
// and hands a call of a method the module does not serve to the node's own
// handler.
type Front struct {
	dispatcher *dispatch.Dispatcher
	enabled    map[string]map[string]bool
	forwarder  Forwarder
}

// NewFront returns a front over the dispatcher. enabled lists the namespaces
// each transport serves; a transport with an empty list serves every
// namespace, as go-ethereum's node does.
func NewFront(dispatcher *dispatch.Dispatcher, enabled map[string][]string) *Front {
	front := &Front{dispatcher: dispatcher, enabled: map[string]map[string]bool{}}
	for transport, namespaces := range enabled {
		if len(namespaces) == 0 {
			continue
		}
		set := map[string]bool{}
		for _, namespace := range namespaces {
			set[namespace] = true
		}
		front.enabled[transport] = set
	}
	return front
}

// Forward makes the front hand every call of a method the module does not
// serve to the forwarder.
func (front *Front) Forward(forwarder Forwarder) {
	front.forwarder = forwarder
}

func (front *Front) forward(ctx context.Context, method string, parameters json.RawMessage) (json.RawMessage, error) {
	elements, err := splitParameters(parameters)
	if err != nil {
		return nil, err
	}
	return front.forwarder.Forward(ctx, method, elements)
}

// splitParameters returns the elements of a parameter array, none for an
// absent or null one.
func splitParameters(raw json.RawMessage) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, null) {
		return nil, nil
	}
	if trimmed[0] != '[' {
		return nil, dispatch.InvalidParameters("non-array args")
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(trimmed, &elements); err != nil {
		return nil, dispatch.InvalidParameters("%v", err)
	}
	return elements, nil
}

// Route serves one call of a transport.
func (front *Front) Route(ctx context.Context, transport string, method string, parameters json.RawMessage) (json.RawMessage, *dispatch.Error) {
	if method == "rpc_modules" {
		return front.modules(ctx, transport)
	}
	if strings.HasSuffix(method, subscribeSuffix) {
		return front.subscribe(ctx, transport, method, parameters)
	}
	if strings.HasSuffix(method, unsubscribeSuffix) {
		return front.unsubscribe(ctx, parameters)
	}
	namespace, _, found := strings.Cut(method, "_")
	if !found || !front.serves(transport, namespace) {
		return nil, dispatch.MethodNotFound(method)
	}
	if front.dispatcher.Has(method) || front.forwarder == nil {
		return front.dispatcher.Serve(ctx, method, parameters)
	}
	return front.dispatcher.Forward(ctx, method, parameters, front.forward)
}

func (front *Front) serves(transport, namespace string) bool {
	set, restricted := front.enabled[transport]
	return !restricted || set[namespace]
}

// modules answers rpc_modules: the namespaces the transport serves, the
// module's and those of the node's handler, with the version every namespace
// reports.
func (front *Front) modules(ctx context.Context, transport string) (json.RawMessage, *dispatch.Error) {
	modules := map[string]string{"rpc": "1.0"}
	for _, method := range front.dispatcher.Methods() {
		namespace, _, _ := strings.Cut(method, "_")
		if front.serves(transport, namespace) {
			modules[namespace] = "1.0"
		}
	}
	if front.forwarder != nil {
		forwarded, err := front.forwarder.Forward(ctx, "rpc_modules", nil)
		if err != nil {
			return nil, dispatch.ToError(err)
		}
		var namespaces map[string]string
		if err := json.Unmarshal(forwarded, &namespaces); err != nil {
			return nil, dispatch.ToError(err)
		}
		for namespace, version := range namespaces {
			if front.serves(transport, namespace) {
				modules[namespace] = version
			}
		}
	}
	encoded, err := json.Marshal(modules)
	if err != nil {
		return nil, dispatch.ToError(err)
	}
	return encoded, nil
}

// subscribe serves a subscription of any namespace. A transport without
// notifications refuses it before anything else, and an unknown one is named
// with its namespace, as go-ethereum's server does.
func (front *Front) subscribe(ctx context.Context, transport, method string, parameters json.RawMessage) (json.RawMessage, *dispatch.Error) {
	if _, ok := dispatch.CallFrom(ctx); !ok {
		return nil, dispatch.ErrNotificationsUnsupported
	}
	if len(method) > maxMethodNameLength {
		return nil, invalidRequest(fmt.Sprintf("subscription name too long: %d > %d", len(method), maxMethodNameLength))
	}
	name, err := subscriptionName(parameters)
	if err != nil {
		return nil, dispatch.InvalidParameters("%v", err)
	}
	namespace, _, _ := strings.Cut(method, "_")
	if !front.serves(transport, namespace) || !front.dispatcher.Has(method) {
		return nil, dispatch.SubscriptionNotFound(namespace, name)
	}
	return front.dispatcher.Serve(ctx, method, parameters)
}

// subscriptionName reads the name of a subscription, its first parameter.
func subscriptionName(parameters json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(parameters))
	if token, _ := decoder.Token(); token != json.Delim('[') {
		return "", errors.New("non-array args")
	}
	token, _ := decoder.Token()
	name, ok := token.(string)
	if !ok {
		return "", errors.New("expected subscription name as first argument")
	}
	return name, nil
}

// errSubscriptionNotFound answers the unsubscription of an id the connection
// does not have.
var errSubscriptionNotFound = &dispatch.Error{Code: dispatch.CodeDefault, Message: "subscription not found"}

var unsubscribeParameters = []dispatch.Parameter{dispatch.Required[string]("subscription")}

// unsubscribe ends a subscription of the connection, of any namespace and on
// any transport, as go-ethereum's server does.
func (front *Front) unsubscribe(ctx context.Context, parameters json.RawMessage) (json.RawMessage, *dispatch.Error) {
	arguments, err := dispatch.DecodeParameters(parameters, unsubscribeParameters)
	if err != nil {
		return nil, dispatch.ToError(err)
	}
	call, ok := dispatch.CallFrom(ctx)
	if !ok || !call.Unsubscribe(dispatch.Argument[string](arguments, 0)) {
		return nil, errSubscriptionNotFound
	}
	return json.RawMessage("true"), nil
}
