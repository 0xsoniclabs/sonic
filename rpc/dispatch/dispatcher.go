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

// Package dispatch is the JSON-RPC runtime of the RPC module, which knows
// nothing of the chain. Its Dispatcher holds the methods, serves each call
// through the same stages (decoding, deadline, handler, encoding, metrics)
// under one recover, and starts every goroutine of the module, so that
// shutdown can wait for all of them.
package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

// Config holds the settings of the dispatcher.
type Config struct {
	// RequestTimeout bounds every call the dispatcher serves.
	RequestTimeout time.Duration
}

// DefaultConfig is the configuration the node uses unless told otherwise.
func DefaultConfig() Config {
	return Config{RequestTimeout: 5 * time.Second}
}

// Handler serves a method. It takes the positional parameters of a call as
// one struct, whose n-th field is the n-th parameter, and returns a typed
// result.
type Handler[Params, Result any] func(ctx context.Context, params Params) (Result, error)

// Encoded is a result already encoded as JSON, which is passed on unchanged.
// A handler returns it when it encoded its answer to check its size.
type Encoded []byte

// Response is the answer to a call: its encoded result, or an error.
type Response struct {
	Result json.RawMessage
	Error  *Error
}

// Dispatcher serves the registered methods and owns every goroutine of the
// module.
type Dispatcher struct {
	config  Config
	methods map[string]*method
	metrics *metrics

	// root is cancelled when Stop ends its grace period.
	root       context.Context
	cancelRoot context.CancelFunc

	mutex    sync.Mutex
	stopping bool
	running  map[*task]struct{}
	requests sync.WaitGroup // admitted calls
	all      sync.WaitGroup // admitted calls, goroutines and loops
}

// method is a registered method. serve decodes the parameters, applies the
// deadline and runs the handler, in that order.
type method struct {
	serve func(ctx context.Context, params json.RawMessage) (any, error)
}

// task is work Stop waits for.
type task struct {
	name  string
	start time.Time
}

// New creates a dispatcher without methods. Its metrics are registered with
// the registerer.
func New(config Config, registerer prometheus.Registerer) *Dispatcher {
	root, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		config:     config,
		methods:    map[string]*method{},
		metrics:    newMetrics(registerer),
		root:       root,
		cancelRoot: cancel,
		running:    map[*task]struct{}{},
	}
}

// Add registers a method. It is a function, because Go methods take no type
// parameters; both types are inferred from the handler. It panics on a
// method registered twice and on a parameter struct it cannot serve, so a
// wrong registration fails at start.
func Add[Params, Result any](dispatcher *Dispatcher, name string, handler Handler[Params, Result]) {
	if _, exists := dispatcher.methods[name]; exists {
		panic(fmt.Sprintf("method %s is registered twice", name))
	}
	layout := newParamsLayout(reflect.TypeFor[Params]())
	timeout := dispatcher.config.RequestTimeout
	dispatcher.methods[name] = &method{
		serve: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var params Params
			if err := layout.decode(raw, &params); err != nil {
				return nil, InvalidParams(err.Error())
			}
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return handler(ctx, params)
		},
	}
}

// Methods lists the names of the registered methods, sorted.
func (d *Dispatcher) Methods() []string {
	return slices.Sorted(maps.Keys(d.methods))
}

// Serves tells whether a method is registered.
func (d *Dispatcher) Serves(name string) bool {
	_, found := d.methods[name]
	return found
}

// Serve answers one call. Everything it runs is under one recover: a panic
// in decoding, in the handler or in encoding answers -32603.
func (d *Dispatcher) Serve(ctx context.Context, name string, params json.RawMessage) (response Response) {
	method, found := d.methods[name]
	if !found {
		d.metrics.countUnknown()
		return Response{Error: MethodNotFound(name)}
	}
	release, admitted := d.admit(name, &d.requests)
	if !admitted {
		return Response{Error: errShuttingDown}
	}
	defer release()

	start := time.Now()
	d.metrics.inflight.Inc()
	defer func() {
		if recovered := recover(); recovered != nil {
			response = d.crashed(name, recovered, debug.Stack())
		}
		d.metrics.inflight.Dec()
		d.metrics.observe(name, response, time.Since(start))
	}()

	ctx, stop := d.cancelledByStop(ctx)
	defer stop()
	result, err := method.serve(ctx, params)
	if err != nil {
		if crash, isCrash := errors.AsType[*childCrash](err); isCrash {
			return d.crashed(name, crash.recovered, crash.stack)
		}
		return Response{Error: ToError(err)}
	}
	return encode(result)
}

// crashed logs and counts a panic of a call, and answers it.
func (d *Dispatcher) crashed(name string, recovered any, stack []byte) Response {
	log.Error(fmt.Sprintf("RPC method %s crashed: %v\n%s", name, recovered, stack))
	d.metrics.panics.WithLabelValues(name).Inc()
	return Response{Error: errHandlerCrashed}
}

func encode(result any) Response {
	if encoded, isEncoded := result.(Encoded); isEncoded {
		return Response{Result: json.RawMessage(encoded)}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return Response{Error: &Error{Code: CodeInternal, Message: err.Error()}}
	}
	return Response{Result: encoded}
}

// cancelledByStop derives a context that is also cancelled when Stop gives up
// waiting.
func (d *Dispatcher) cancelledByStop(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(d.root, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

// Go runs work outside a call, such as a connection or a forwarded call, in
// a goroutine Stop waits for. Its context is cancelled when Stop gives up
// waiting. A panic of the work is logged and ends only the work. Go answers
// false once Stop has begun, without running anything.
func (d *Dispatcher) Go(name string, work func(ctx context.Context)) bool {
	release, admitted := d.admit(name, nil)
	if !admitted {
		return false
	}
	go func() {
		defer release()
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error(fmt.Sprintf("RPC work %s crashed: %v\n%s", name, recovered, debug.Stack()))
			}
		}()
		work(d.root)
	}()
	return true
}

// admit counts work Stop waits for, under the same lock as the check that
// Stop has not begun. Calls are counted in requests as well.
func (d *Dispatcher) admit(name string, requests *sync.WaitGroup) (release func(), admitted bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.stopping {
		return nil, false
	}
	work := &task{name: name, start: time.Now()}
	d.running[work] = struct{}{}
	d.all.Add(1)
	if requests != nil {
		requests.Add(1)
	}
	return func() {
		d.mutex.Lock()
		delete(d.running, work)
		d.mutex.Unlock()
		if requests != nil {
			requests.Done()
		}
		d.all.Done()
	}, true
}

// Stop refuses new work, lets the calls in flight finish until the request
// timeout has passed, then cancels everything and waits for it. While it
// waits, it logs every so often what it is waiting for: work that ignores
// cancellation, such as an EVM run of an interpreter that never polls.
func (d *Dispatcher) Stop() {
	d.mutex.Lock()
	d.stopping = true
	d.mutex.Unlock()

	d.waitLogging(&d.requests, d.config.RequestTimeout)
	d.cancelRoot()
	d.waitLogging(&d.all, 0)
}

// stopLogInterval is how often Stop logs the work it waits for.
var stopLogInterval = 5 * time.Second

// waitLogging waits for a wait group, at most for limit unless limit is zero,
// logging the running work every stopLogInterval.
func (d *Dispatcher) waitLogging(group *sync.WaitGroup, limit time.Duration) {
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	var expired <-chan time.Time
	if limit > 0 {
		timer := time.NewTimer(limit)
		defer timer.Stop()
		expired = timer.C
	}
	ticker := time.NewTicker(stopLogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-expired:
			return
		case <-ticker.C:
			d.logRunning()
		}
	}
}

func (d *Dispatcher) logRunning() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for work := range d.running {
		log.Warn("RPC shutdown waits for work", "work", work.name, "age", time.Since(work.start))
	}
}
