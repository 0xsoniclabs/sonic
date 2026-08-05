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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPeerRunTracker_AcquireIsGrantedWhileOpen(t *testing.T) {
	var tracker peerRunTracker

	require.True(t, tracker.acquireRun())
	require.True(t, tracker.acquireRun())

	tracker.releaseRun()
	tracker.releaseRun()

	tracker.refuseNewRuns()
	tracker.waitForRuns()
}

func TestPeerRunTracker_AcquireIsRefusedAfterRefuseNewRuns(t *testing.T) {
	var tracker peerRunTracker
	tracker.refuseNewRuns()

	require.False(t, tracker.acquireRun())
}

func TestPeerRunTracker_WaitBlocksUntilRunsReturn(t *testing.T) {
	const runs = 5

	var tracker peerRunTracker
	for range runs {
		require.True(t, tracker.acquireRun())
	}
	tracker.refuseNewRuns()

	waited := make(chan struct{})
	go func() {
		defer close(waited)
		tracker.waitForRuns()
	}()

	for range runs - 1 {
		tracker.releaseRun()
		select {
		case <-waited:
			t.Fatal("wait returned while peer handlers were still running")
		case <-time.After(10 * time.Millisecond):
		}
	}

	tracker.releaseRun()

	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("wait did not return after all peer handlers returned")
	}
}

// TestPeerRunTracker_NoRunOutlivesTheWait is the regression test for the
// shutdown race: a peer handler must never register itself after the shutdown
// has started waiting. Before the fix, the gate check and the WaitGroup
// increment were two separate steps, so a handler could slip in between them
// -- tripping "WaitGroup misuse: Add called concurrently with Wait" or, worse,
// silently outliving the shutdown.
func TestPeerRunTracker_NoRunOutlivesTheWait(t *testing.T) {
	const attempts = 200

	var tracker peerRunTracker

	var running atomic.Int32
	var done sync.WaitGroup
	release := make(chan struct{})

	for range attempts {
		done.Add(1)
		go func() {
			defer done.Done()
			if !tracker.acquireRun() {
				return
			}
			running.Add(1)
			<-release
			running.Add(-1)
			tracker.releaseRun()
		}()
	}

	go func() {
		time.Sleep(time.Millisecond)
		close(release)
	}()

	tracker.refuseNewRuns()
	tracker.waitForRuns()
	require.Zero(t, running.Load())

	done.Wait()
	require.False(t, tracker.acquireRun())
}
