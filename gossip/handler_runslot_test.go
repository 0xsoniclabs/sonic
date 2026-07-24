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

package gossip

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquireRunSlot_SucceedsWhenNotStopping(t *testing.T) {
	h := newTestHandler()

	require.True(t, h.acquireRunSlot())
	require.Equal(t, 1, h.runningPeers)

	require.True(t, h.acquireRunSlot())
	require.Equal(t, 2, h.runningPeers)

	h.releaseRunSlot()
	h.releaseRunSlot()
	require.Equal(t, 0, h.runningPeers)
}

func TestAcquireRunSlot_FailsAfterStopping(t *testing.T) {
	h := newTestHandler()

	h.peerCond.L.Lock()
	h.stopping = true
	h.peerCond.L.Unlock()

	require.False(t, h.acquireRunSlot())
	require.Equal(t, 0, h.runningPeers)
}

func TestStop_BlocksUntilRunSlotsReleased(t *testing.T) {
	h := newTestHandler()

	// Simulate a peer holding a run slot.
	require.True(t, h.acquireRunSlot())

	stopDone := make(chan struct{})
	go func() {
		stopLogic(h, stopDone)
	}()

	// Stop should not complete while the slot is held.
	select {
	case <-stopDone:
		t.Fatal("stop completed while run slot is still held")
	case <-time.After(50 * time.Millisecond):
		// expected
	}

	// Release the slot — Stop should now unblock.
	h.releaseRunSlot()

	select {
	case <-stopDone:
		// expected
	case <-time.After(time.Second):
		t.Fatal("stop did not complete after releasing run slot")
	}
}

func TestStop_BlocksUntilAllRunSlotsReleased(t *testing.T) {
	h := newTestHandler()

	const n = 5
	for i := 0; i < n; i++ {
		require.True(t, h.acquireRunSlot())
	}

	stopDone := make(chan struct{})
	go func() {
		stopLogic(h, stopDone)
	}()

	// Release slots one by one; stop should not complete until all are gone.
	for i := 0; i < n-1; i++ {
		h.releaseRunSlot()
		select {
		case <-stopDone:
			t.Fatal("stop completed with slots still held")
		case <-time.After(20 * time.Millisecond):
		}
	}

	// Release the last slot.
	h.releaseRunSlot()

	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("stop did not complete after all slots released")
	}
}

func TestAcquireRunSlot_ConcurrentWithStop(t *testing.T) {
	// Hammer acquireRunSlot and releaseRunSlot from many goroutines
	// while triggering a stop, verifying no panics or deadlocks.
	h := newTestHandler()

	const goroutines = 50
	var wg sync.WaitGroup
	ready := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			if h.acquireRunSlot() {
				// Simulate some work.
				time.Sleep(time.Millisecond)
				h.releaseRunSlot()
			}
		}()
	}

	// Let all goroutines race.
	close(ready)

	// Give some goroutines time to acquire slots.
	time.Sleep(5 * time.Millisecond)

	// Trigger stop.
	stopLogic(h, make(chan struct{}))

	// All goroutines must finish.
	wg.Wait()
	require.Equal(t, 0, h.runningPeers)
}

// newTestHandler creates a minimal handler with only the peerCond
// initialized, sufficient for testing acquireRunSlot/releaseRunSlot.
func newTestHandler() *handler {
	h := &handler{}
	h.peerCond = sync.NewCond(&sync.Mutex{})
	return h
}

// stopLogic replicates the Stop() shutdown logic in isolation.
func stopLogic(h *handler, stopDone chan struct{}) {
	h.peerCond.L.Lock()
	h.stopping = true
	for h.runningPeers > 0 {
		h.peerCond.Wait()
	}
	h.peerCond.L.Unlock()
	close(stopDone)
}
