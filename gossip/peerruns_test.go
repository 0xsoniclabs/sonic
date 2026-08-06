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
	"testing/synctest"

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

// TestPeerRunTracker_WaitBlocksUntilRunsReturn runs in a synctest bubble:
// synctest.Wait returns once every other goroutine is durably blocked, so the
// waiter has provably had its chance to run before each assertion -- no
// sleeps, and a wait that never returns fails with stacks instead of hanging.
func TestPeerRunTracker_WaitBlocksUntilRunsReturn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const runs = 5

		var tracker peerRunTracker
		for range runs {
			require.True(t, tracker.acquireRun())
		}
		tracker.refuseNewRuns()

		var waitReturned atomic.Bool
		go func() {
			tracker.waitForRuns()
			waitReturned.Store(true)
		}()

		for range runs - 1 {
			tracker.releaseRun()
			synctest.Wait()
			require.False(t, waitReturned.Load(),
				"wait returned while peer handlers were still running")
		}

		tracker.releaseRun()
		synctest.Wait()
		require.True(t, waitReturned.Load(),
			"wait did not return after all peer handlers returned")
	})
}

// TestPeerRunTracker_NoRunOutlivesTheWait is the regression test for the
// shutdown race: a peer handler must never register itself after the shutdown
// has started waiting. Before the fix, the gate check and the WaitGroup
// increment were two separate steps, so a handler could slip in between them
// -- tripping "WaitGroup misuse: Add called concurrently with Wait" or, worse,
// silently outliving the shutdown.
//
// No synctest here: the point is real parallelism. Each round is a fresh
// head-to-head race between one arriving run and the shutdown, since the
// dangerous interleaving needs the wait to start on an empty tracker. Run with
// -race.
func TestPeerRunTracker_NoRunOutlivesTheWait(t *testing.T) {
	const rounds = 5000

	for range rounds {
		var tracker peerRunTracker

		var waitReturned atomic.Bool
		var escaped atomic.Bool

		start := make(chan struct{})
		var done sync.WaitGroup
		done.Add(2)

		go func() {
			defer done.Done()
			<-start
			if !tracker.acquireRun() {
				return
			}
			if waitReturned.Load() {
				escaped.Store(true)
			}
			tracker.releaseRun()
		}()

		go func() {
			defer done.Done()
			<-start
			tracker.refuseNewRuns()
			tracker.waitForRuns()
			waitReturned.Store(true)
		}()

		close(start)
		done.Wait()

		require.False(t, escaped.Load(),
			"a peer handler was still running after the shutdown wait returned")
		require.False(t, tracker.acquireRun())
	}
}
