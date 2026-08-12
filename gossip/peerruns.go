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

import "sync"

// peerRunTracker tracks the peer handler goroutines that the p2p server starts
// on the Service's behalf. The Service cannot stop them directly: the p2p
// server keeps starting them until it is itself shut down, which the node only
// does after every service has come down. The tracker lets the Service refuse
// the late ones and wait out the running ones instead.
//
// The zero value is an open tracker with no runs in flight.
type peerRunTracker struct {
	// mu makes the closed check and the wg.Add a single step. As two steps, a
	// run could observe an open tracker, be descheduled, and reach wg.Add
	// after waitForRuns had already started on a zero counter -- which
	// sync.WaitGroup forbids, and which lets the run outlive the shutdown.
	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// acquireRun registers a peer handler goroutine and reports whether it may
// proceed. It returns false once the tracker is closed, in which case nothing
// was registered and the caller must not call releaseRun.
func (t *peerRunTracker) acquireRun() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return false
	}
	t.wg.Add(1)
	return true
}

// releaseRun reports that a goroutine registered by acquireRun has finished.
func (t *peerRunTracker) releaseRun() {
	t.wg.Done()
}

// refuseNewRuns makes all further acquireRun calls fail. It does not wait for
// the runs already in flight; see Service.stopPeerHandling for that.
func (t *peerRunTracker) refuseNewRuns() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
}

// waitForRuns blocks until every registered peer handler goroutine has
// returned. It must follow refuseNewRuns, otherwise an arriving run races the
// wait, and the teardown that releases the running ones -- see
// Service.stopPeerHandling.
func (t *peerRunTracker) waitForRuns() {
	t.wg.Wait()
}
