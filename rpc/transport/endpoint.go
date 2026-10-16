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

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Router serves one method call.
type Router interface {
	Call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, *dispatch.Error)
	Subscribe(ctx context.Context, ns string, allowed bool, params json.RawMessage) (json.RawMessage, *dispatch.Error)
}

// writer sends JSON to the peer. isError marks error responses, which an HTTP
// transport flushes at once.
type writer interface {
	write(ctx context.Context, v any, isError bool) error
}

// EndpointConfig describes one transport's view on the methods.
type EndpointConfig struct {
	// Name is the transport label of the metrics: "http", "ws" or "ipc".
	Name string
	// Modules are the namespaces served on the transport; empty means all.
	Modules []string
	// BatchLimit is the largest accepted batch, 0 = unlimited.
	BatchLimit int
	// BatchResponseMax is the largest cumulated batch response in bytes, 0 = unlimited.
	BatchResponseMax int
	// Notifications is true if the transport has a connection to notify.
	Notifications bool
	// Local is true for the transport that is local to the node (IPC): it also
	// serves the namespaces of the legacy handler.
	Local bool
}

// Endpoint applies the JSON-RPC 2.0 envelope rules of one transport: it tells
// calls from notifications, enforces the batch limits, enables namespaces and
// hands each call to the router.
type Endpoint struct {
	cfg     EndpointConfig
	router  Router
	modules map[string]bool
}

// NewEndpoint creates an endpoint.
func NewEndpoint(cfg EndpointConfig, router Router) *Endpoint {
	e := &Endpoint{cfg: cfg, router: router}
	if len(cfg.Modules) > 0 {
		e.modules = map[string]bool{}
		for _, m := range cfg.Modules {
			e.modules[m] = true
		}
	}
	return e
}

// Config returns the configuration of the endpoint.
func (e *Endpoint) Config() EndpointConfig { return e.cfg }

func (e *Endpoint) enabled(namespace string) bool {
	return e.modules == nil || e.modules[namespace]
}

// handleBatch answers a batch of messages.
func (e *Endpoint) handleBatch(ctx context.Context, msgs []*Message, w writer) {
	if len(msgs) == 0 {
		resp := errorMessage(dispatch.NewError(dispatch.CodeInvalidRequest, "empty batch"))
		_ = w.write(ctx, resp, true)
		return
	}
	if e.cfg.BatchLimit != 0 && len(msgs) > e.cfg.BatchLimit {
		resp := errorMessage(dispatch.NewError(dispatch.CodeInvalidRequest, "batch too large"))
		// The protocol has no way to report an error for the entire batch: use
		// the id of the first call.
		for _, msg := range msgs {
			if msg.isCall() {
				resp.ID = msg.ID
				break
			}
		}
		_ = w.write(ctx, []*Message{resp}, true)
		return
	}
	var calls []*Message
	for _, msg := range msgs {
		if !msg.isResponse() {
			calls = append(calls, msg)
		}
	}
	if len(calls) == 0 {
		return
	}
	responses := make([]*Message, 0, len(calls))
	size := 0
	for i, msg := range calls {
		if ctx.Err() != nil {
			break
		}
		resp := e.handleCallMsg(ctx, msg)
		if resp != nil {
			responses = append(responses, resp)
		}
		if resp != nil && e.cfg.BatchResponseMax != 0 {
			size += len(resp.Result)
			if size > e.cfg.BatchResponseMax {
				err := dispatch.NewError(dispatch.CodeTooLarge, "response too large")
				for _, rest := range calls[i+1:] {
					if !rest.isNotification() {
						responses = append(responses, rest.errorResponse(err))
					}
				}
				_ = w.write(ctx, responses, true)
				activate(responses)
				return
			}
		}
	}
	if len(responses) > 0 {
		_ = w.write(ctx, responses, false)
		activate(responses)
	}
}

// activate lets the notifications of the subscriptions created by the
// responses out; it runs after the responses were written.
func activate(responses []*Message) {
	for _, resp := range responses {
		if resp.activate != nil {
			resp.activate()
		}
	}
}

// handleMsg answers a single message.
func (e *Endpoint) handleMsg(ctx context.Context, msg *Message, w writer) {
	if msg.isResponse() {
		return
	}
	resp := e.handleCallMsg(ctx, msg)
	if resp != nil {
		_ = w.write(ctx, resp, resp.Error != nil)
		activate([]*Message{resp})
	}
}

// handleCallMsg executes a call message and returns the answer; notifications
// are executed and have no answer.
func (e *Endpoint) handleCallMsg(ctx context.Context, msg *Message) *Message {
	switch {
	case msg.isNotification():
		e.handleCall(ctx, msg)
		return nil
	case msg.isCall():
		return e.handleCall(ctx, msg)
	case msg.hasValidID():
		return msg.errorResponse(dispatch.NewError(dispatch.CodeInvalidRequest, "invalid request"))
	default:
		return errorMessage(dispatch.NewError(dispatch.CodeInvalidRequest, "invalid request"))
	}
}

func (e *Endpoint) handleCall(ctx context.Context, msg *Message) *Message {
	if msg.isSubscribe() {
		return e.handleSubscribe(ctx, msg)
	}
	if msg.isUnsubscribe() {
		return e.handleUnsubscribe(ctx, msg)
	}
	if len(msg.Method) > maxMethodNameLength {
		return msg.errorResponse(dispatch.NewError(dispatch.CodeInvalidRequest,
			"method name too long: %d > %d", len(msg.Method), maxMethodNameLength))
	}
	if msg.Method == "rpc_modules" {
		return e.handleModules(msg)
	}
	if !e.enabled(msg.namespace()) {
		return msg.errorResponse(dispatch.MethodNotFound(msg.Method))
	}
	if e.cfg.Local {
		ctx = withIPC(ctx)
	}
	result, err := e.router.Call(ctx, msg.Method, msg.Params)
	if err != nil {
		return msg.errorResponse(err)
	}
	if len(result) == 0 {
		result = null
	}
	return msg.response(result)
}

// handleModules answers rpc_modules: the namespaces the transport serves.
func (e *Endpoint) handleModules(msg *Message) *Message {
	namer, ok := e.router.(interface {
		Namespaces(func(string) bool) map[string]string
	})
	if !ok {
		return msg.errorResponse(dispatch.MethodNotFound(msg.Method))
	}
	raw, err := json.Marshal(namer.Namespaces(e.enabled))
	if err != nil {
		return msg.errorResponse(dispatch.NewError(dispatch.CodeInternal, "%v", err))
	}
	return msg.response(raw)
}

func (e *Endpoint) handleSubscribe(ctx context.Context, msg *Message) *Message {
	notifier, ok := dispatch.NotifierFrom(ctx)
	if !e.cfg.Notifications || !ok {
		return msg.errorResponse(dispatch.ErrNotificationsUnsupported)
	}
	if len(msg.Method) > maxMethodNameLength {
		return msg.errorResponse(dispatch.NewError(dispatch.CodeInvalidRequest,
			"subscription name too long: %d > %d", len(msg.Method), maxMethodNameLength))
	}
	ns := msg.namespace()
	result, err := e.router.Subscribe(ctx, ns, e.enabled(ns), msg.Params)
	if err != nil {
		return msg.errorResponse(err)
	}
	resp := msg.response(result)
	var id string
	if json.Unmarshal(result, &id) == nil {
		// The client may be notified once it has the id.
		resp.activate = func() { notifier.Activate(id) }
	}
	return resp
}

func (e *Endpoint) handleUnsubscribe(ctx context.Context, msg *Message) *Message {
	raw, err := dispatch.SplitParams(msg.Params)
	switch {
	case err != nil:
		return msg.errorResponse(dispatch.InvalidParams("%v", err))
	case len(raw) > 1:
		return msg.errorResponse(dispatch.InvalidParams("too many arguments, want at most 1"))
	case len(raw) == 0:
		return msg.errorResponse(dispatch.InvalidParams("missing value for required argument 0"))
	}
	var id string
	if err := json.Unmarshal(raw[0], &id); err != nil {
		return msg.errorResponse(dispatch.InvalidParams("invalid argument 0: %v", err))
	}
	if notifier, ok := dispatch.NotifierFrom(ctx); ok && notifier.Unsubscribe(id) {
		return msg.response(json.RawMessage("true"))
	}
	return msg.errorResponse(dispatch.NewError(dispatch.CodeDefault, "%v", dispatch.ErrSubscriptionNotFound))
}
