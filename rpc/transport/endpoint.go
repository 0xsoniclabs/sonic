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
	"sync"
	"time"

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
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	b := &batchResponses{calls: calls}
	// The calls still unanswered when the time of the request is up get a
	// timeout error at once, as with go-ethereum's server.
	var timer *time.Timer
	if timeout, ok := requestTimeout(ctx); ok {
		timer = time.AfterFunc(timeout, func() {
			cancel()
			b.respondWithError(parent, w, requestTimedOut())
		})
	}
	size := 0
	for ctx.Err() == nil {
		msg := b.next()
		if msg == nil {
			break
		}
		resp := e.handleCallMsg(ctx, msg)
		b.push(resp)
		if resp != nil && e.cfg.BatchResponseMax != 0 {
			size += len(resp.Result)
			if size > e.cfg.BatchResponseMax {
				b.respondWithError(parent, w, dispatch.NewError(dispatch.CodeTooLarge, "response too large"))
				break
			}
		}
	}
	if timer != nil {
		timer.Stop()
	}
	b.write(parent, w)
	b.activate()
}

// batchResponses collects the answers of a batch. They are written once: all
// of them, or, if the batch is cut short, those so far with an error for every
// call without an answer, the running one included.
type batchResponses struct {
	mu    sync.Mutex
	calls []*Message // the calls without an answer; the first may be running
	resp  []*Message
	wrote bool
}

// next returns the call to run next, nil if there is none.
func (b *batchResponses) next() *Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.wrote || len(b.calls) == 0 {
		return nil
	}
	return b.calls[0]
}

// push adds the answer of the call next returned.
func (b *batchResponses) push(resp *Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if resp != nil {
		b.resp = append(b.resp, resp)
	}
	b.calls = b.calls[1:]
}

// respondWithError answers the calls without an answer with the error and
// writes the responses.
func (b *batchResponses) respondWithError(ctx context.Context, w writer, err *dispatch.Error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.wrote {
		return
	}
	for _, msg := range b.calls {
		if !msg.isNotification() {
			b.resp = append(b.resp, msg.errorResponse(err))
		}
	}
	b.flush(ctx, w, true)
}

// write writes the responses unless they were written already.
func (b *batchResponses) write(ctx context.Context, w writer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flush(ctx, w, false)
}

func (b *batchResponses) flush(ctx context.Context, w writer, isError bool) {
	if b.wrote {
		return
	}
	b.wrote = true
	if len(b.resp) > 0 {
		_ = w.write(ctx, b.resp, isError)
	}
}

func (b *batchResponses) activate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	activate(b.resp)
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

// handleMsg answers a single message. A call still running when the time of
// the request is up is answered with a timeout error at once and its late
// answer is dropped, as with go-ethereum's server.
func (e *Endpoint) handleMsg(ctx context.Context, msg *Message, w writer) {
	if msg.isResponse() {
		return
	}
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var responded sync.Once
	var timer *time.Timer
	if timeout, ok := requestTimeout(ctx); ok {
		timer = time.AfterFunc(timeout, func() {
			cancel()
			responded.Do(func() {
				_ = w.write(parent, msg.errorResponse(requestTimedOut()), true)
			})
		})
	}
	resp := e.handleCallMsg(ctx, msg)
	if timer != nil {
		timer.Stop()
	}
	// Taken even without an answer: a timeout error being written is done
	// before the transport finishes the request.
	responded.Do(func() {
		if resp != nil {
			_ = w.write(parent, resp, resp.Error != nil)
		}
	})
	if resp != nil {
		activate([]*Message{resp})
	}
}

type requestTimeoutKey struct{}

// withRequestTimeout sets the time a transport allows for a request: its
// answer has to be written before the transport cuts the connection.
func withRequestTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, requestTimeoutKey{}, timeout)
}

// requestTimeout returns how long a request may run before it is answered
// with a timeout error: the earlier of the deadline of the context and the
// time the transport allows.
func requestTimeout(ctx context.Context) (time.Duration, bool) {
	timeout, ok := ctx.Value(requestTimeoutKey{}).(time.Duration)
	if deadline, has := ctx.Deadline(); has {
		if left := time.Until(deadline); !ok || left < timeout {
			timeout, ok = left, true
		}
	}
	return timeout, ok
}

// requestTimedOut is the error of a call that ran out of time.
func requestTimedOut() *dispatch.Error {
	return dispatch.NewError(dispatch.CodeTimeout, "request timed out")
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
