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

package wgmutex

import (
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)
	if m == nil {
		t.Fatal("expected non-nil WgMutex")
	}
	if m.RWMutex != mu {
		t.Fatal("unexpected RWMutex")
	}
}

func TestLock_WaitsForWaitGroup(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)

	wg.Add(1)
	done := make(chan struct{})
	go func() {
		m.Lock()
		close(done)
		m.Unlock()
	}()

	// The goroutine should be blocked because wg has count 1
	select {
	case <-done:
		t.Fatal("Lock should block while WaitGroup is not done")
	default:
	}

	wg.Done()
	<-done
}

func TestRLock_WaitsForWaitGroup(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)

	wg.Add(1)
	done := make(chan struct{})
	go func() {
		m.RLock()
		close(done)
		m.RUnlock()
	}()

	select {
	case <-done:
		t.Fatal("RLock should block while WaitGroup is not done")
	default:
	}

	wg.Done()
	<-done
}

func TestLock_BasicMutexBehavior(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)

	m.Lock()
	_ = 0 // exercise lock/unlock cycle
	m.Unlock()
}

func TestRLock_BasicMutexBehavior(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)

	m.RLock()
	_ = 0 // exercise rlock/runlock cycle
	m.RUnlock()
}
