// Copyright 2025 Sonic Operations Ltd
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

package threads

import (
	"runtime/debug"
	"sync"
)

const GoroutinesPerThread = 0.8

// ThreadPool counts threads in use
type ThreadPool struct {
	mu   sync.Mutex
	cap  int
	left int
}

var GlobalPool ThreadPool

// init ThreadPool only on demand to give time to other packages
// call debug.SetMaxThreads() if they need
func (p *ThreadPool) init() {
	if p.cap == 0 {
		p.cap = int(getMaxThreads() * GoroutinesPerThread)
		p.left = p.cap
	}
}

// Cap returns the capacity of the pool
func (p *ThreadPool) Cap() int {
	if p.cap == 0 {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.init()
	}
	return p.cap
}

func (p *ThreadPool) Lock(want int) (got int, release func(count int)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.init()

	if want < 1 {
		want = 0
	}

	got = min(p.left, want)
	p.left -= got

	release = func(count int) {
		p.mu.Lock()
		defer p.mu.Unlock()

		if 0 > count || count > got {
			count = got
		}

		got -= count
		p.left += count
	}

	return
}

func getMaxThreads() float64 {
	was := debug.SetMaxThreads(10000)
	debug.SetMaxThreads(was)
	return float64(was)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
