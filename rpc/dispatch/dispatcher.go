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

// Package dispatch is the JSON-RPC runtime of the module and nothing of the
// chain: the table of methods, the decoding of parameters, the request
// deadline, the recovery of panics, the counting of every call, the
// goroutines of request work and the long-lived loops, and their stop.
package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Handler answers a call with the decoded arguments of its method.
type Handler func(ctx context.Context, arguments Arguments) (any, error)

// Encoded is a result already encoded as JSON. The dispatcher hands it on
// unchanged; a handler that must encode its result to check its size returns
// the bytes as Encoded instead of encoding twice.
type Encoded []byte

// MarshalJSON returns the encoded bytes.
func (encoded Encoded) MarshalJSON() ([]byte, error) {
	return encoded, nil
}

// Config holds the settings of the dispatcher.
type Config struct {
	// RequestTimeout is the deadline of every request. Stop waits as long for
	// the requests in flight before it cancels them.
	RequestTimeout time.Duration
}

// DefaultConfig returns the default settings, those of the reference.
func DefaultConfig() Config {
	return Config{RequestTimeout: 5 * time.Second}
}

type method struct {
	handler    Handler
	parameters []Parameter
}

// Dispatcher is the one entry point of every call on every transport and the
// only place that starts goroutines for the module's work.
type Dispatcher struct {
	config  Config
	metrics *Metrics

	methodsMutex sync.RWMutex
	methods      map[string]method

	// work is cancelled when the grace period of Stop ends; loops when the
	// work has ended.
	work       context.Context
	cancelWork context.CancelFunc
	loops      context.Context
	cancelLoop context.CancelFunc

	admission sync.Mutex
	stopping  bool
	running   sync.WaitGroup
	looping   sync.WaitGroup
	tracked   map[*tracked]struct{}
}

// tracked is a piece of admitted work that Stop waits for.
type tracked struct {
	name  string
	start time.Time
}

// New returns a dispatcher with no methods.
func New(config Config, metrics *Metrics) *Dispatcher {
	work, cancelWork := context.WithCancel(context.Background())
	loops, cancelLoop := context.WithCancel(context.Background())
	return &Dispatcher{
		config:     config,
		metrics:    metrics,
		methods:    map[string]method{},
		work:       work,
		cancelWork: cancelWork,
		loops:      loops,
		cancelLoop: cancelLoop,
		tracked:    map[*tracked]struct{}{},
	}
}

// Metrics returns the metrics the dispatcher counts into.
func (dispatcher *Dispatcher) Metrics() *Metrics {
	return dispatcher.metrics
}

// Add registers a method. A method exists only because a call of Add names
// it. Adding a name twice is a programming error.
func (dispatcher *Dispatcher) Add(name string, handler Handler, parameters ...Parameter) {
	dispatcher.methodsMutex.Lock()
	defer dispatcher.methodsMutex.Unlock()
	if _, found := dispatcher.methods[name]; found {
		panic(fmt.Sprintf("method %s registered twice", name))
	}
	dispatcher.methods[name] = method{handler: handler, parameters: parameters}
}

// Has tells whether a method is registered.
func (dispatcher *Dispatcher) Has(name string) bool {
	dispatcher.methodsMutex.RLock()
	defer dispatcher.methodsMutex.RUnlock()
	_, found := dispatcher.methods[name]
	return found
}

// Methods returns the names of the registered methods, sorted.
func (dispatcher *Dispatcher) Methods() []string {
	dispatcher.methodsMutex.RLock()
	defer dispatcher.methodsMutex.RUnlock()
	names := make([]string, 0, len(dispatcher.methods))
	for name := range dispatcher.methods {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (dispatcher *Dispatcher) lookup(name string) (method, bool) {
	dispatcher.methodsMutex.RLock()
	defer dispatcher.methodsMutex.RUnlock()
	found, ok := dispatcher.methods[name]
	return found, ok
}

type methodKey struct{}

// Serve answers one call: it looks the method up, decodes its parameters,
// admits the call, runs the handler under the request deadline and recover,
// encodes the result and counts the call. The error it answers carries the
// JSON-RPC code.
func (dispatcher *Dispatcher) Serve(ctx context.Context, name string, parameters json.RawMessage) (json.RawMessage, *Error) {
	start := time.Now()
	entry, found := dispatcher.lookup(name)
	if !found {
		dispatcher.count(OtherMethod, ResultInvalid, start, nil)
		return nil, MethodNotFound(name)
	}
	return dispatcher.serve(ctx, start, name, name, entry, parameters)
}

// ForwardedMethod is the label of the calls the module hands to the node's
// own handler.
const ForwardedMethod = "forwarded"

// Forward answers a call the module does not serve by handing it to the
// node's handler, through the same admission, deadline, recover and counting
// as a registered method. Its parameters arrive undecoded.
func (dispatcher *Dispatcher) Forward(ctx context.Context, name string, parameters json.RawMessage, forward func(ctx context.Context, name string, parameters json.RawMessage) (json.RawMessage, error)) (json.RawMessage, *Error) {
	entry := method{
		handler: func(ctx context.Context, arguments Arguments) (any, error) {
			result, err := forward(ctx, name, Argument[json.RawMessage](arguments, 0))
			if err != nil {
				return nil, err
			}
			return Encoded(result), nil
		},
		parameters: []Parameter{wholeArray()},
	}
	return dispatcher.serve(ctx, time.Now(), ForwardedMethod, name, entry, parameters)
}

// serve runs the stages after the lookup. label is what the call is counted
// under, name what it is logged as.
func (dispatcher *Dispatcher) serve(ctx context.Context, start time.Time, label, name string, entry method, parameters json.RawMessage) (json.RawMessage, *Error) {
	arguments, err := DecodeParameters(parameters, entry.parameters)
	if err != nil {
		dispatcher.count(label, ResultInvalid, start, nil)
		return nil, ToError(err)
	}
	work, admitted := dispatcher.admit(name)
	if !admitted {
		dispatcher.count(label, ResultRejected, start, nil)
		return nil, ErrStopping
	}
	defer dispatcher.finish(work)

	ctx, cancel := dispatcher.requestContext(ctx)
	defer cancel()
	ctx = context.WithValue(ctx, methodKey{}, label)

	result, err := dispatcher.run(ctx, label, entry.handler, arguments)
	if err != nil {
		answer := ToError(err)
		outcome := ResultError
		if answer == ErrPanic {
			outcome = ResultPanic
		}
		dispatcher.count(label, outcome, start, nil)
		log.Debug("Served "+name, "duration", time.Since(start), "err", answer.Message)
		return nil, answer
	}
	dispatcher.count(label, ResultSuccess, start, result)
	log.Debug("Served "+name, "duration", time.Since(start))
	return result, nil
}

// requestContext applies the request deadline and makes the context end
// when Stop cancels the work in flight.
func (dispatcher *Dispatcher) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, dispatcher.config.RequestTimeout)
	stopCancel := context.AfterFunc(dispatcher.work, cancel)
	return ctx, func() {
		stopCancel()
		cancel()
	}
}

// run calls the handler and encodes its result under recover, so that a panic
// in either becomes ErrPanic and never leaves the dispatcher.
func (dispatcher *Dispatcher) run(ctx context.Context, name string, handler Handler, arguments Arguments) (result json.RawMessage, err error) {
	defer func() {
		if value := recover(); value != nil {
			dispatcher.recovered(name, value)
			result, err = nil, ErrPanic
		}
	}()
	answer, err := handler(ctx, arguments)
	if err != nil {
		return nil, err
	}
	return encode(answer)
}

func encode(answer any) (json.RawMessage, error) {
	if encoded, ok := answer.(Encoded); ok {
		return json.RawMessage(encoded), nil
	}
	encoded, err := json.Marshal(answer)
	if err != nil {
		return nil, &Error{Code: CodeInternalError, Message: err.Error()}
	}
	return encoded, nil
}

// recovered logs a recovered panic once with its stack and counts it.
func (dispatcher *Dispatcher) recovered(name string, value any) {
	log.Error("RPC method "+name+" crashed", "panic", value, "stack", string(debug.Stack()))
	dispatcher.metrics.Panics.WithLabelValues(name).Inc()
}

func (dispatcher *Dispatcher) count(name, result string, start time.Time, response json.RawMessage) {
	dispatcher.metrics.Requests.WithLabelValues(name, result).Inc()
	if result == ResultInvalid || result == ResultRejected {
		return
	}
	dispatcher.metrics.Duration.WithLabelValues(name).Observe(time.Since(start).Seconds())
	if response != nil {
		dispatcher.metrics.ResponseBytes.WithLabelValues(name).Observe(float64(len(response)))
	}
}

// methodOf returns the method a context serves, OtherMethod outside a call.
func methodOf(ctx context.Context) string {
	if name, ok := ctx.Value(methodKey{}).(string); ok {
		return name
	}
	return OtherMethod
}

// admit counts a piece of work Stop waits for, unless Stop began. The check
// and the count are taken under one lock, so no work starts after Stop.
func (dispatcher *Dispatcher) admit(name string) (*tracked, bool) {
	dispatcher.admission.Lock()
	defer dispatcher.admission.Unlock()
	if dispatcher.stopping {
		return nil, false
	}
	dispatcher.running.Add(1)
	work := &tracked{name: name, start: time.Now()}
	dispatcher.tracked[work] = struct{}{}
	dispatcher.metrics.Inflight.Inc()
	return work, true
}

func (dispatcher *Dispatcher) finish(work *tracked) {
	dispatcher.admission.Lock()
	delete(dispatcher.tracked, work)
	dispatcher.admission.Unlock()
	dispatcher.metrics.Inflight.Dec()
	dispatcher.running.Done()
}

// Go runs work on a goroutine of its own, admitted like a request, under
// recover, with a context that ends with ctx or when Stop cancels the work in
// flight. It answers false, running nothing, once Stop began.
func (dispatcher *Dispatcher) Go(ctx context.Context, name string, run func(ctx context.Context)) bool {
	work, admitted := dispatcher.admit(name)
	if !admitted {
		return false
	}
	ctx, cancel := context.WithCancel(ctx)
	stopCancel := context.AfterFunc(dispatcher.work, cancel)
	go func() {
		defer dispatcher.finish(work)
		defer stopCancel()
		defer cancel()
		defer func() {
			if value := recover(); value != nil {
				dispatcher.recovered(name, value)
			}
		}()
		run(ctx)
	}()
	return true
}

// Stop refuses new work, waits for the work in flight up to the request
// deadline, then cancels what is left and waits for it, and last stops the
// loops. The second wait has no timer: what ignores cancellation (an EVM run
// on an interpreter that does not poll, a call into the node) is waited for,
// because the node closes its stores once Stop returns. Each piece of work
// still running when the grace period ends is logged with its age.
func (dispatcher *Dispatcher) Stop() {
	dispatcher.admission.Lock()
	if dispatcher.stopping {
		dispatcher.admission.Unlock()
		return
	}
	dispatcher.stopping = true
	dispatcher.admission.Unlock()

	if !wait(&dispatcher.running, dispatcher.config.RequestTimeout) {
		dispatcher.logRunning()
		dispatcher.cancelWork()
		dispatcher.running.Wait()
	}
	dispatcher.cancelWork()
	dispatcher.cancelLoop()
	dispatcher.looping.Wait()
}

// Stopping tells whether Stop began.
func (dispatcher *Dispatcher) Stopping() bool {
	dispatcher.admission.Lock()
	defer dispatcher.admission.Unlock()
	return dispatcher.stopping
}

func (dispatcher *Dispatcher) logRunning() {
	dispatcher.admission.Lock()
	defer dispatcher.admission.Unlock()
	for work := range dispatcher.tracked {
		log.Warn("RPC stop waits for a request that ignores cancellation", "method", work.name, "age", time.Since(work.start))
	}
}

// wait waits for the group up to the timeout and tells whether it ended.
func wait(group *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}
