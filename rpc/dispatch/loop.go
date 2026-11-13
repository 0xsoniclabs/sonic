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
	"fmt"
	"runtime/debug"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// The backoff between the restarts of a loop that panics.
var (
	loopFirstBackoff = 100 * time.Millisecond
	loopMaxBackoff   = 30 * time.Second
)

// Loop starts a long-lived loop, such as the head loop, which runs until its
// context is cancelled by Stop. A loop that panics is restarted with a
// growing backoff, and each restart is counted and logged; a restarted loop
// begins as at start, so it subscribes again. Loop answers false once Stop
// has begun, without starting anything.
func (d *Dispatcher) Loop(name string, run func(ctx context.Context)) bool {
	return d.Go(name, func(ctx context.Context) {
		backoff := loopFirstBackoff
		for restarts := 0; ; restarts++ {
			if !d.runLoopOnce(name, ctx, run) {
				return
			}
			d.metrics.loopRestarts.WithLabelValues(name).Inc()
			log.Error("RPC loop restarts after a panic", "loop", name, "restarts", restarts+1, "backoff", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(2*backoff, loopMaxBackoff)
		}
	})
}

// runLoopOnce runs a loop and answers whether it panicked.
func (d *Dispatcher) runLoopOnce(name string, ctx context.Context, run func(ctx context.Context)) (panicked bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Error(fmt.Sprintf("RPC loop %s crashed: %v\n%s", name, recovered, debug.Stack()))
			panicked = true
		}
	}()
	run(ctx)
	return false
}
