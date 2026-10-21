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

package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Handler serves one method. Methods registered with the typed helpers
// (Add0 ... Add5) never see raw arguments; Handler is the form the dispatcher
// stores them in.
type Handler func(ctx context.Context, args Args) (any, error)

// method is an entry of the method table.
type method struct {
	name    string
	params  []string // names of the positional parameters
	minArgs int      // number of required parameters
	handler Handler
}

// Dispatcher is the entry point of every request. It is the only place that
// starts goroutines for request work. Methods exist by registration only:
// nothing is reflected off a struct, so a lifecycle or helper method of a
// namespace can never become callable.
type Dispatcher struct {
	cfg     Config
	metrics *Metrics

	mu      sync.RWMutex
	methods map[string]*method
	subs    map[string]*subscription

	root     context.Context
	cancel   context.CancelFunc
	gate     sync.RWMutex // makes the stopping check and the count of new work atomic towards Stop
	stopping atomic.Bool
	inflight sync.WaitGroup // requests being served
	tasks    sync.WaitGroup // goroutines started through Go and Loop
	stopOnce sync.Once
}

// New creates a dispatcher without methods.
func New(cfg Config, metrics *Metrics) *Dispatcher {
	if metrics == nil {
		metrics = NewMetrics(nil)
	}
	root, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		cfg:     cfg,
		metrics: metrics,
		methods: map[string]*method{},
		subs:    map[string]*subscription{},
		root:    root,
		cancel:  cancel,
	}
}

// Config returns the configuration the dispatcher was created with.
func (d *Dispatcher) Config() Config { return d.cfg }

// Metrics returns the collectors of the dispatcher.
func (d *Dispatcher) Metrics() *Metrics { return d.metrics }

// Add registers a method with a raw handler. The parameter names end in "?"
// when the parameter is optional; optional parameters must come last.
func (d *Dispatcher) Add(name string, h Handler, params ...string) {
	m := &method{name: name, handler: h}
	for i, p := range params {
		if strings.HasSuffix(p, "?") {
			p = strings.TrimSuffix(p, "?")
		} else {
			if i != m.minArgs {
				panic(fmt.Sprintf("rpc: method %s: required parameter %q after an optional one", name, p))
			}
			m.minArgs++
		}
		m.params = append(m.params, p)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.methods[name]; exists {
		panic(fmt.Sprintf("rpc: method %s registered twice", name))
	}
	d.methods[name] = m
}

// Methods returns the names of all registered methods, including the
// subscription entry points.
func (d *Dispatcher) Methods() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	names := make([]string, 0, len(d.methods)+len(d.subs))
	for name := range d.methods {
		names = append(names, name)
	}
	return names
}

// Signature returns the parameter names of a registered method (without the
// marks of optional ones) and the number of required parameters.
func (d *Dispatcher) Signature(name string) (params []string, required int, ok bool) {
	m := d.lookup(name)
	if m == nil {
		return nil, 0, false
	}
	return append([]string(nil), m.params...), m.minArgs, true
}

// Namespaces returns the namespaces that have methods or subscriptions.
func (d *Dispatcher) Namespaces() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	set := map[string]bool{}
	for name := range d.methods {
		ns, _, _ := strings.Cut(name, "_")
		set[ns] = true
	}
	for key := range d.subs {
		ns, _, _ := strings.Cut(key, "/")
		set[ns] = true
	}
	names := make([]string, 0, len(set))
	for ns := range set {
		names = append(names, ns)
	}
	sort.Strings(names)
	return names
}

// Has reports whether a method is registered.
func (d *Dispatcher) Has(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.methods[name]
	return ok
}

func (d *Dispatcher) lookup(name string) *method {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.methods[name]
}

// Args are the positional parameters of a request, counted against the
// signature of the method they are served by.
type Args struct {
	raw    []json.RawMessage
	method *method
	// countErr is the error of a parameter count that does not fit the
	// method. go-ethereum reports it after the decoding errors of the
	// parameters that are there, so it is surfaced by Check.
	countErr *Error
}

// Check returns the error of a parameter count that does not fit the method.
// Handlers decode their parameters first and call Check last; the typed Add
// helpers do both.
func (a Args) Check() error {
	if a.countErr != nil {
		return a.countErr
	}
	return nil
}

// Len returns the number of parameters the client sent.
func (a Args) Len() int { return len(a.raw) }

// Raw returns the undecoded parameter i, or nil if it is absent.
func (a Args) Raw(i int) json.RawMessage {
	if i < 0 || i >= len(a.raw) {
		return nil
	}
	return a.raw[i]
}

func (a Args) name(i int) string {
	if a.method != nil && i < len(a.method.params) {
		return a.method.params[i]
	}
	return fmt.Sprint(i)
}

// Decode decodes the parameter i into v. An absent parameter leaves v alone.
// A failure is a -32602 naming the parameter.
func (a Args) Decode(i int, v any) error {
	raw := a.Raw(i)
	if raw == nil {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return InvalidParams("parameter %d (%s): %v", i, a.name(i), err)
	}
	return nil
}

// decode decodes the parameter i as a T.
func decode[T any](a Args, i int) (v T, err error) {
	err = a.Decode(i, &v)
	return v, err
}

// SplitParams parses the "params" member of a request into its elements. A
// missing or null member has no elements; anything but an array is an error.
func SplitParams(params json.RawMessage) ([]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(params))
	tok, err := dec.Token()
	switch {
	case err == io.EOF || tok == nil && err == nil:
		// "params" is optional and may be null.
		return nil, nil
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
	default:
		return nil, errors.New("non-array args")
	}
	var raw []json.RawMessage
	for dec.More() {
		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return nil, err
		}
		raw = append(raw, item)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return raw, nil
}

// bind splits the parameters and counts them against the signature of the
// method. Only a syntax error fails here; a wrong count is kept in the Args so
// the order of go-ethereum is kept: decoding errors of the parameters that are
// there, then too many parameters, then missing ones.
func (m *method) bind(params json.RawMessage) (Args, *Error) {
	raw, err := SplitParams(params)
	if err != nil {
		return Args{}, InvalidParams("%v", err)
	}
	args := Args{raw: raw, method: m}
	if len(raw) > len(m.params) {
		args.countErr = InvalidParams("too many arguments, want at most %d", len(m.params))
	} else if len(raw) < m.minArgs {
		args.countErr = InvalidParams("missing value for required argument %d", len(raw))
	}
	return args, nil
}

// Serve serves one request: look the method up, decode and count the
// parameters, attach the deadline, run the handler under recover, and encode
// and count the response. It returns the encoded result or the error object.
// Serve runs on the caller's goroutine.
func (d *Dispatcher) Serve(ctx context.Context, name string, params json.RawMessage) (json.RawMessage, *Error) {
	start := time.Now()
	m := d.lookup(name)
	if m == nil {
		d.metrics.Requests.WithLabelValues(otherLabel, ResultNotFound).Inc()
		return nil, MethodNotFound(name)
	}
	label := m.name

	if !d.admit(&d.inflight) {
		d.metrics.Requests.WithLabelValues(label, ResultRejected).Inc()
		return nil, NewError(CodeDefault, "server is shutting down")
	}
	defer d.inflight.Done()
	d.metrics.Inflight.Inc()
	defer d.metrics.Inflight.Dec()

	args, perr := m.bind(params)
	if perr != nil {
		d.metrics.Requests.WithLabelValues(label, ResultInvalidParams).Inc()
		return nil, perr
	}

	ctx, cancel := d.requestContext(ctx, label)
	defer cancel()

	result, err := d.run(ctx, m, args)
	if err != nil {
		res := ResultError
		switch {
		case isPanic(err):
			res = ResultPanic
		case ToError(err).Code == CodeInvalidParams:
			res = ResultInvalidParams
		case ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
			res = ResultTimeout
		case ctx.Err() != nil:
			res = ResultCancelled
		}
		d.observe(label, res, start, 0)
		return nil, panicToError(err)
	}
	encoded, merr := json.Marshal(result)
	if merr != nil {
		d.observe(label, ResultError, start, 0)
		return nil, NewError(CodeInternal, "%v", merr)
	}
	d.observe(label, ResultOK, start, len(encoded))
	return encoded, nil
}

func (d *Dispatcher) observe(label, result string, start time.Time, size int) {
	d.metrics.Requests.WithLabelValues(label, result).Inc()
	d.metrics.Duration.WithLabelValues(label).Observe(time.Since(start).Seconds())
	if result == ResultOK {
		d.metrics.ResponseBytes.WithLabelValues(label).Observe(float64(size))
	}
}

type methodKey struct{}

// requestContext attaches the deadline of the request and ties it to the
// lifetime of the dispatcher.
func (d *Dispatcher) requestContext(ctx context.Context, method string) (context.Context, context.CancelFunc) {
	ctx = context.WithValue(ctx, methodKey{}, method)
	var cancel context.CancelFunc
	if d.cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, d.cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	stop := context.AfterFunc(d.root, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

// MethodOf returns the method a request context belongs to.
func MethodOf(ctx context.Context) string {
	if m, ok := ctx.Value(methodKey{}).(string); ok {
		return m
	}
	return otherLabel
}

// PanicError is the error of a handler or child goroutine that panicked.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string { return "method handler crashed" }

func isPanic(err error) bool {
	var p *PanicError
	return errors.As(err, &p)
}

func panicToError(err error) *Error {
	if isPanic(err) {
		return NewError(CodeInternal, "method handler crashed")
	}
	return ToError(err)
}

// recovered turns a recovered value into a PanicError, logging it once with
// its stack and counting it under the label. The label is a method name or a
// "task:"/"loop:" name, all fixed by the code.
func (d *Dispatcher) recovered(label, what string, value any) *PanicError {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Error("RPC "+what+" crashed", "panic", fmt.Sprintf("%v", value), "stack", string(buf))
	d.metrics.Panics.WithLabelValues(label).Inc()
	return &PanicError{Value: value, Stack: buf}
}

// run calls the handler under recover.
func (d *Dispatcher) run(ctx context.Context, m *method, args Args) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			result, err = nil, d.recovered(m.name, "method "+m.name, r)
		}
	}()
	return m.handler(ctx, args)
}

// Go runs fn on a goroutine tracked by the dispatcher, under recover. It
// returns false and does not run fn when the dispatcher is stopping. Stop
// waits for fn: it must return when the context of the dispatcher ends.
func (d *Dispatcher) Go(name string, fn func(ctx context.Context)) bool {
	if !d.admit(&d.tasks) {
		return false
	}
	go func() {
		defer d.tasks.Done()
		defer func() {
			if r := recover(); r != nil {
				d.recovered("task:"+name, "task "+name, r)
			}
		}()
		fn(d.root)
	}()
	return true
}

// Loop runs fn as a long-lived loop. A panic restarts it with an exponential
// backoff and counts in rpc_loop_restarts_total; a loop is never silently dead.
// fn returns when its context ends.
func (d *Dispatcher) Loop(name string, fn func(ctx context.Context)) bool {
	if !d.admit(&d.tasks) {
		return false
	}
	go func() {
		defer d.tasks.Done()
		backoff := 50 * time.Millisecond
		for d.root.Err() == nil {
			panicked := d.runLoop(name, fn)
			if !panicked {
				return
			}
			d.metrics.LoopRestarts.WithLabelValues(name).Inc()
			select {
			case <-d.root.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 10*time.Second)
		}
	}()
	return true
}

func (d *Dispatcher) runLoop(name string, fn func(ctx context.Context)) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			d.recovered("loop:"+name, "loop "+name, r)
		}
	}()
	fn(d.root)
	return false
}

// admit counts new work in the group unless the dispatcher is stopping. Once
// Stop marked the dispatcher as stopping, no group grows any more, so Stop
// never waits on a count that may still rise.
func (d *Dispatcher) admit(group *sync.WaitGroup) bool {
	d.gate.RLock()
	defer d.gate.RUnlock()
	if d.stopping.Load() {
		return false
	}
	group.Add(1)
	return true
}

// Context returns the context of the dispatcher; it ends when Stop begins to
// cancel what is left in flight.
func (d *Dispatcher) Context() context.Context { return d.root }

// Stop drains the requests in flight for up to grace, cancels what is left, and
// waits for every goroutine the dispatcher started. New requests are refused as
// soon as Stop is called.
func (d *Dispatcher) Stop(grace time.Duration) {
	d.stopOnce.Do(func() {
		d.gate.Lock()
		d.stopping.Store(true)
		d.gate.Unlock()
		drained := make(chan struct{})
		go func() {
			d.inflight.Wait()
			close(drained)
		}()
		select {
		case <-drained:
		case <-time.After(grace):
		}
		d.cancel()
		<-drained
		d.tasks.Wait()
	})
}

// Stopping reports whether Stop was called.
func (d *Dispatcher) Stopping() bool { return d.stopping.Load() }
