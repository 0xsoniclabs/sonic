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
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Backoff of a loop restarted after a panic: it doubles from the first value
// up to the last, and starts over once a run lasted a minute.
const (
	loopBackoffFirst = 100 * time.Millisecond
	loopBackoffMax   = 10 * time.Second
	loopHealthyRun   = time.Minute
)

// Go runs f on a goroutine of the dispatcher. Stop cancels the context f gets
// and waits for f to return; a panic in f is logged and counted, never fatal.
// It reports false, and does not run f, once Stop began.
func (d *Dispatcher) Go(name string, f func(ctx context.Context)) bool {
	w, ok := d.admit(false, "goroutine", name)
	if !ok {
		return false
	}
	go func() {
		defer d.release(w, false)
		defer func() {
			if r := recover(); r != nil {
				d.recovered(name, r)
			}
		}()
		f(d.ctx)
	}()
	return true
}

// Loop runs a long-lived loop from Start until Stop; a loop added after Start
// runs at once. When f panics, the loop is counted in rpc_loop_restarts_total
// and run again after a backoff; when f returns, the loop ends. A restarted f
// starts from nothing: whatever its previous run acquired was released by the
// panic, so f takes its subscriptions itself.
func (d *Dispatcher) Loop(name string, f func(ctx context.Context)) bool {
	d.mu.Lock()
	if !d.started && !d.stopping {
		d.loops = append(d.loops, loop{name: name, f: f})
		d.mu.Unlock()
		return true
	}
	d.mu.Unlock()
	return d.startLoop(name, f)
}

// Start runs the loops added so far.
func (d *Dispatcher) Start() {
	d.mu.Lock()
	d.started = true
	loops := d.loops
	d.loops = nil
	d.mu.Unlock()
	for _, l := range loops {
		d.startLoop(l.name, l.f)
	}
}

func (d *Dispatcher) startLoop(name string, f func(ctx context.Context)) bool {
	w, ok := d.admit(false, "loop", name)
	if !ok {
		return false
	}
	go func() {
		defer d.release(w, false)
		backoff := loopBackoffFirst
		for restarts := 0; ; restarts++ {
			start := time.Now()
			if !d.runLoop(name, f) {
				return
			}
			d.metrics.loopRestarts.WithLabelValues(name).Inc()
			if time.Since(start) >= loopHealthyRun {
				backoff = loopBackoffFirst
			}
			log.Error("RPC loop restarts after a panic", "loop", name, "restarts", restarts+1, "backoff", backoff)
			select {
			case <-d.ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(2*backoff, loopBackoffMax)
		}
	}()
	return true
}

// runLoop runs one pass of a loop and reports whether it panicked.
func (d *Dispatcher) runLoop(name string, f func(ctx context.Context)) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			d.recovered(name, r)
			panicked = d.ctx.Err() == nil
		}
	}()
	f(d.ctx)
	return false
}

// Group runs the children of one request. A child that panics or fails
// cancels its siblings; Wait waits for all of them and returns the first
// failure. Children run under the request's admission, which Stop waits on.
type Group struct {
	d      *Dispatcher
	label  string
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
	err    error
}

// Group creates a group for the request of ctx, and the context its children
// run under.
func (d *Dispatcher) Group(ctx context.Context) (*Group, context.Context) {
	label, _ := ctx.Value(methodKey{}).(string)
	ctx, cancel := context.WithCancel(ctx)
	return &Group{d: d, label: label, cancel: cancel}, ctx
}

// Go runs one child.
func (g *Group) Go(f func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				g.d.recovered(g.label, r)
				g.fail(errPanic)
			}
		}()
		if err := f(); err != nil {
			g.fail(err)
		}
	}()
}

func (g *Group) fail(err error) {
	g.once.Do(func() {
		g.err = err
		g.cancel()
	})
}

// Wait waits for every child and returns the first failure.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.cancel()
	return g.err
}
