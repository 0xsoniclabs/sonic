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
	m.Unlock()
	// Should not deadlock
}

func TestRLock_BasicMutexBehavior(t *testing.T) {
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	m := New(mu, wg)

	m.RLock()
	m.RUnlock()
	// Should not deadlock
}
