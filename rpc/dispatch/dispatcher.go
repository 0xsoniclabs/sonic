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

// Package dispatch is the JSON-RPC runtime of the RPC module: the method
// table, the decoding of parameters, the deadline, the recovery of panics and
// the accounting of every call, and the only place that starts goroutines.
// It knows nothing of the chain.
package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// maxMethodNameLength bounds the method names the dispatcher looks up, as in
// go-ethereum.
const maxMethodNameLength = 2048

// Config holds the request deadline, the one limit the dispatcher applies.
type Config struct {
	// RequestTimeout is the deadline of every request (--rpc.timeout); zero
	// means none.
	RequestTimeout time.Duration
}

// DefaultConfig is the one definition of the default request deadline.
func DefaultConfig() Config {
	return Config{RequestTimeout: 5 * time.Second}
}

// Encoded is a result a handler already encoded, for instance to check its
// size; the dispatcher hands it on unchanged.
type Encoded []byte

// Call is one decoded request of a transport.
type Call struct {
	Method string
	Params json.RawMessage
}

// Dispatcher serves every request of every transport, and owns every
// goroutine of the module. Methods exist only by registration: Add and
// AddSubscription are called before the first request.
type Dispatcher struct {
	cfg     Config
	metrics *Metrics

	methods       map[string]*Method
	subscriptions map[string]map[string]*Method // namespace, kind

	// ctx is the base of every request and goroutine; Stop cancels it once
	// the grace period ended.
	ctx    context.Context
	cancel context.CancelFunc

	// mu takes admission and the count Stop waits on together, so nothing is
	// admitted once Stop began.
	mu       sync.Mutex
	started  bool
	stopping bool
	requests sync.WaitGroup
	all      sync.WaitGroup
	running  map[*work]struct{}
	loops    []loop // added before Start
}

// loop is a long-lived loop waiting for Start.
type loop struct {
	name string
	f    func(ctx context.Context)
}

// work is one admitted request, goroutine or loop, for the log of Stop.
type work struct {
	kind, name string
	start      time.Time
}

// New creates a dispatcher without methods.
func New(cfg Config, metrics *Metrics) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		cfg:           cfg,
		metrics:       metrics,
		methods:       make(map[string]*Method),
		subscriptions: make(map[string]map[string]*Method),
		ctx:           ctx,
		cancel:        cancel,
		running:       make(map[*work]struct{}),
	}
}

// Metrics returns the counters the dispatcher maintains.
func (d *Dispatcher) Metrics() *Metrics {
	return d.metrics
}

// Add registers a method under its full name, such as "eth_getBalance".
func (d *Dispatcher) Add(name string, m Method) {
	if _, dup := d.methods[name]; dup {
		panic(fmt.Sprintf("dispatch: method %s registered twice", name))
	}
	d.methods[name] = &m
}

// AddSubscription registers a kind of subscription of a namespace, served by
// "<namespace>_subscribe" with the kind as its first parameter.
func (d *Dispatcher) AddSubscription(namespace, kind string, m Method) {
	kinds := d.subscriptions[namespace]
	if kinds == nil {
		kinds = make(map[string]*Method)
		d.subscriptions[namespace] = kinds
	}
	if _, dup := kinds[kind]; dup {
		panic(fmt.Sprintf("dispatch: subscription %s of %s registered twice", kind, namespace))
	}
	kinds[kind] = &m
}

// Methods lists the names of the registered methods, the subscription
// methods of each namespace with subscriptions included.
func (d *Dispatcher) Methods() []string {
	names := make([]string, 0, len(d.methods)+2*len(d.subscriptions))
	for name := range d.methods {
		names = append(names, name)
	}
	for ns := range d.subscriptions {
		names = append(names, ns+"_subscribe", ns+"_unsubscribe")
	}
	sort.Strings(names)
	return names
}

// Namespaces lists the namespaces of the registered methods.
func (d *Dispatcher) Namespaces() []string {
	set := make(map[string]bool)
	for name := range d.methods {
		set[NamespaceOf(name)] = true
	}
	for ns := range d.subscriptions {
		set[ns] = true
	}
	names := make([]string, 0, len(set))
	for ns := range set {
		names = append(names, ns)
	}
	sort.Strings(names)
	return names
}

// methodKey carries the metric label of the running method in its context.
type methodKey struct{}

// Serve runs one call: it looks the method up, decodes and counts its
// parameters, attaches the deadline, runs the handler under recover, and
// encodes and counts the result. The error it returns is the JSON-RPC error
// of the answer.
func (d *Dispatcher) Serve(ctx context.Context, call Call) (json.RawMessage, error) {
	w, ok := d.admit(true, "request", call.Method)
	if !ok {
		d.metrics.requests.WithLabelValues(d.label(call.Method), resultRejected).Inc()
		return nil, errStopped
	}
	defer d.release(w, true)
	d.metrics.inflight.Inc()
	defer d.metrics.inflight.Dec()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(d.ctx, cancel)
	defer stop()

	start := time.Now()
	label := d.label(call.Method)
	res, result, err := d.serve(ctx, label, call)
	d.metrics.requests.WithLabelValues(label, result).Inc()
	d.metrics.duration.WithLabelValues(label).Observe(time.Since(start).Seconds())
	if err == nil {
		d.metrics.responseBytes.WithLabelValues(label).Observe(float64(len(res)))
	}
	return res, err
}

// Has reports whether the dispatcher serves a method: a registered method, or
// the subscription methods of a namespace with subscriptions.
func (d *Dispatcher) Has(name string) bool {
	return d.label(name) != otherMethod
}

// forwardedMethod labels the calls the node's own handler answers.
const forwardedMethod = "forwarded"

// Forward runs a call the module does not serve through forward, the node's
// own handler, under the same admission, deadline, recovery and accounting
// as every other call.
func (d *Dispatcher) Forward(ctx context.Context, call Call, forward func(ctx context.Context, call Call) (json.RawMessage, error)) (json.RawMessage, error) {
	w, ok := d.admit(true, "request", call.Method)
	if !ok {
		d.metrics.requests.WithLabelValues(forwardedMethod, resultRejected).Inc()
		return nil, errStopped
	}
	defer d.release(w, true)
	d.metrics.inflight.Inc()
	defer d.metrics.inflight.Dec()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(d.ctx, cancel)
	defer stop()
	if d.cfg.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, d.cfg.RequestTimeout)
		defer cancel()
	}

	start := time.Now()
	m := Method{handler: func(ctx context.Context, _ Args) (any, error) {
		res, err := forward(ctx, call)
		if err != nil {
			return nil, err
		}
		return Encoded(res), nil
	}}
	res, err := d.run(ctx, forwardedMethod, &m, nil)
	result := resultOK
	switch {
	case err == errPanic:
		result = resultPanic
	case err != nil:
		result = resultError
	}
	d.metrics.requests.WithLabelValues(forwardedMethod, result).Inc()
	d.metrics.duration.WithLabelValues(forwardedMethod).Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, err
	}
	enc := json.RawMessage(res.(Encoded))
	d.metrics.responseBytes.WithLabelValues(forwardedMethod).Observe(float64(len(enc)))
	return enc, nil
}

// serve answers a call and classifies the answer for the metrics.
func (d *Dispatcher) serve(ctx context.Context, label string, call Call) (json.RawMessage, string, error) {
	var (
		m    *Method
		args Args
		err  error
		name = call.Method
	)
	switch {
	case strings.HasSuffix(name, "_subscribe"):
		n, ok := NotifierFrom(ctx)
		if !ok {
			return nil, resultError, ErrNotificationsUnsupported
		}
		if len(name) > maxMethodNameLength {
			return nil, resultError, &Error{Code: CodeInvalidRequest, Message: fmt.Sprintf("subscription name too long: %d > %d", len(name), maxMethodNameLength)}
		}
		kind, err := SubscriptionKind(call.Params)
		if err != nil {
			return nil, resultInvalidParams, &Error{Code: CodeInvalidParams, Message: err.Error()}
		}
		namespace := NamespaceOf(name)
		if m = d.subscriptions[namespace][kind]; m == nil {
			return nil, resultNotFound, SubscriptionNotFound(namespace, kind)
		}
		if args, err = m.decode(call.Params, 1); err != nil {
			return nil, resultInvalidParams, &Error{Code: CodeInvalidParams, Message: err.Error()}
		}
		ctx = WithNotifier(ctx, n)

	case strings.HasSuffix(name, "_unsubscribe"):
		m = &unsubscribe
		if args, err = m.decode(call.Params, 0); err != nil {
			return nil, resultInvalidParams, &Error{Code: CodeInvalidParams, Message: err.Error()}
		}

	default:
		if len(name) > maxMethodNameLength {
			return nil, resultError, &Error{Code: CodeInvalidRequest, Message: fmt.Sprintf("method name too long: %d > %d", len(name), maxMethodNameLength)}
		}
		if m = d.methods[name]; m == nil {
			return nil, resultNotFound, MethodNotFound(name)
		}
		if args, err = m.decode(call.Params, 0); err != nil {
			return nil, resultInvalidParams, &Error{Code: CodeInvalidParams, Message: err.Error()}
		}
	}

	if d.cfg.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.cfg.RequestTimeout)
		defer cancel()
	}
	ctx = context.WithValue(ctx, methodKey{}, label)

	res, err := d.run(ctx, label, m, args)
	if err != nil {
		if err == errPanic {
			return nil, resultPanic, err
		}
		return nil, resultError, err
	}
	if enc, ok := res.(Encoded); ok {
		return json.RawMessage(enc), resultOK, nil
	}
	enc, err := json.Marshal(res)
	if err != nil {
		return nil, resultError, &Error{Code: CodeInternal, Message: err.Error()}
	}
	return enc, resultOK, nil
}

// unsubscribe serves "<namespace>_unsubscribe" of every namespace, as
// go-ethereum does: it ends a subscription of the calling connection.
var unsubscribe = Func(func(ctx context.Context, p struct {
	ID string `rpc:"id"`
}) (any, error) {
	if n, ok := NotifierFrom(ctx); ok && n.Unsubscribe(p.ID) {
		return true, nil
	}
	return nil, ErrSubscriptionNotFound
})

// run calls a handler under recover: a panic is logged with its stack,
// counted, and answered -32603.
func (d *Dispatcher) run(ctx context.Context, label string, m *Method, args Args) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			d.recovered(label, r)
			res, err = nil, errPanic
		}
	}()
	return m.handler(ctx, args)
}

// recovered logs and counts a recovered panic.
func (d *Dispatcher) recovered(label string, r any) {
	log.Error("RPC "+label+" crashed", "err", r, "stack", string(debug.Stack()))
	d.metrics.panics.WithLabelValues(label).Inc()
}

// label is the metric label of a method: its name if the dispatcher serves
// it, otherwise "other".
func (d *Dispatcher) label(name string) string {
	if _, ok := d.methods[name]; ok {
		return name
	}
	if strings.HasSuffix(name, "_subscribe") || strings.HasSuffix(name, "_unsubscribe") {
		if _, ok := d.subscriptions[NamespaceOf(name)]; ok {
			return name
		}
	}
	return otherMethod
}

// admit takes one unit of work unless Stop began; the check and the count
// Stop waits on are taken under one lock.
func (d *Dispatcher) admit(request bool, kind, name string) (*work, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopping {
		return nil, false
	}
	if request {
		d.requests.Add(1)
	}
	d.all.Add(1)
	w := &work{kind: kind, name: name, start: time.Now()}
	d.running[w] = struct{}{}
	return w, true
}

func (d *Dispatcher) release(w *work, request bool) {
	d.mu.Lock()
	delete(d.running, w)
	d.mu.Unlock()
	if request {
		d.requests.Done()
	}
	d.all.Done()
}

// Stop refuses new work, waits up to grace for the requests in flight, then
// cancels everything left and waits for it: the node closes its stores once
// Stop returns. Work that ignores cancellation (an EVM run on Tosca's
// interpreters, a Core call) bounds that last wait; Stop logs what it still
// waits for at the end of the grace period and at every grace period after.
func (d *Dispatcher) Stop(grace time.Duration) {
	d.mu.Lock()
	if d.stopping {
		d.mu.Unlock()
		return
	}
	d.stopping = true
	d.mu.Unlock()

	drained := make(chan struct{})
	go func() {
		d.requests.Wait()
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(grace):
		d.logRunning("RPC requests still running after the grace period")
	}
	d.cancel()

	done := make(chan struct{})
	go func() {
		d.all.Wait()
		close(done)
	}()
	ticker := time.NewTicker(max(grace, time.Second))
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			d.logRunning("RPC shutdown still waiting")
		}
	}
}

// logRunning logs every admitted unit of work still running, with its age.
func (d *Dispatcher) logRunning(msg string) {
	d.mu.Lock()
	running := make([]*work, 0, len(d.running))
	for w := range d.running {
		running = append(running, w)
	}
	d.mu.Unlock()
	sort.Slice(running, func(i, j int) bool { return running[i].start.Before(running[j].start) })
	for _, w := range running {
		log.Warn(msg, "kind", w.kind, "name", w.name, "age", time.Since(w.start).Round(time.Millisecond))
	}
}
