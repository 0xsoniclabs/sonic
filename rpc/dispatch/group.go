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
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Group runs the children of one request. The first child that fails or
// panics cancels its siblings, and Wait returns its error once every child
// ended.
type Group struct {
	dispatcher *Dispatcher
	name       string
	cancel     context.CancelCauseFunc
	children   sync.WaitGroup
	once       sync.Once
	err        error
}

// Group returns a group for the children of the request ctx belongs to, and
// the context the children run under.
func (dispatcher *Dispatcher) Group(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{dispatcher: dispatcher, name: methodOf(ctx), cancel: cancel}, ctx
}

// Go starts a child. A child that panics fails with ErrPanic.
func (group *Group) Go(ctx context.Context, run func(ctx context.Context) error) {
	group.children.Add(1)
	go func() {
		defer group.children.Done()
		if err := group.call(ctx, run); err != nil {
			group.fail(err)
		}
	}()
}

func (group *Group) call(ctx context.Context, run func(ctx context.Context) error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			group.dispatcher.recovered(group.name, value)
			err = ErrPanic
		}
	}()
	return run(ctx)
}

func (group *Group) fail(err error) {
	group.once.Do(func() {
		group.err = err
		group.cancel(err)
	})
}

// Wait waits for every child and returns the error of the first that failed.
func (group *Group) Wait() error {
	group.children.Wait()
	group.cancel(nil)
	return group.err
}

// The backoff of a loop that restarts: it doubles from the first value to
// the last.
const (
	firstBackoff = 100 * time.Millisecond
	lastBackoff  = 10 * time.Second
)

// Loop runs a long-lived loop until Stop. A run that panics or fails is
// restarted after a backoff that grows while the loop keeps failing, counted
// and logged each time, so a loop is never silently dead. The run function
// takes everything it needs (subscriptions, the head) itself on each run: what
// a panicking run held was released by its defers. It answers false, running
// nothing, once Stop began.
func (dispatcher *Dispatcher) Loop(name string, run func(ctx context.Context) error) bool {
	dispatcher.admission.Lock()
	defer dispatcher.admission.Unlock()
	if dispatcher.stopping {
		return false
	}
	dispatcher.looping.Add(1)
	go func() {
		defer dispatcher.looping.Done()
		dispatcher.repeat(dispatcher.loops, name, run)
	}()
	return true
}

// Run runs a background task once, until it returns: the server of a
// listener, which returns when the listener closes. Stop waits for it with
// the loops; a panic in it is logged and counted. It answers false, running
// nothing, once Stop began.
func (dispatcher *Dispatcher) Run(name string, run func(ctx context.Context)) bool {
	dispatcher.admission.Lock()
	defer dispatcher.admission.Unlock()
	if dispatcher.stopping {
		return false
	}
	dispatcher.looping.Add(1)
	go func() {
		defer dispatcher.looping.Done()
		defer func() {
			if value := recover(); value != nil {
				dispatcher.recovered(name, value)
			}
		}()
		run(dispatcher.loops)
	}()
	return true
}

func (dispatcher *Dispatcher) repeat(ctx context.Context, name string, run func(ctx context.Context) error) {
	backoff := firstBackoff
	for {
		err := dispatcher.runLoop(ctx, name, run)
		if ctx.Err() != nil {
			return
		}
		dispatcher.metrics.LoopRestarts.WithLabelValues(name).Inc()
		log.Error("RPC loop ended, restarting", "loop", name, "err", err, "backoff", backoff)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(2*backoff, lastBackoff)
	}
}

// errLoopReturned is the cause of a restart when a loop returned without an
// error before the module stopped.
var errLoopReturned = errors.New("loop returned")

func (dispatcher *Dispatcher) runLoop(ctx context.Context, name string, run func(ctx context.Context) error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			dispatcher.recovered(name, value)
			err = ErrPanic
		}
	}()
	if err := run(ctx); err != nil {
		return err
	}
	return errLoopReturned
}
