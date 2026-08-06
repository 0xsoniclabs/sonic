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
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// The tests in this file exercise the real peer-run entry point -- the Run
// closure MakeProtocols hands to the p2p server -- against the real handler
// teardown, with only the network mocked out. They cover the part the
// peerRunTracker unit tests cannot: that handler.Stop really does unpark
// everything a peer run can be parked on, which is the assumption that lets
// Service.Stop wait for the runs after the handler teardown instead of before.
//
// All of them are worth running with -race, and none of them may hang: each
// shutdown runs under a watchdog that fails with goroutine stacks instead.

const shutdownWatchdog = 30 * time.Second

// TestPeerRun_HandlerStopUnparksRunBlockedOnMessageSemaphore parks a peer run
// in the message semaphore and then runs the production shutdown sequence.
// The parked run cannot time out on its own -- DataSemaphore.Acquire only
// re-checks its deadline when the condition variable is signalled -- so if
// handler.Stop stopped terminating the semaphore, waitForRuns would never
// return.
func TestPeerRun_HandlerStopUnparksRunBlockedOnMessageSemaphore(t *testing.T) {
	h, svc, run := newPeerRunTestSetup(t, 8)

	// Saturate the message semaphore, so the first message of the session
	// below parks the peer run instead of being handled.
	require.True(t, h.msgSemaphore.Acquire(h.config.Protocol.MsgsSemaphoreLimit, time.Second))

	session := newTestSession(h, 1)
	runDone := make(chan error, 1)
	go func() {
		runDone <- run(newTestP2PPeer(), session)
	}()

	<-session.delivered
	waitUntilSomeGoroutineIsIn(t, "datasemaphore.(*DataSemaphore).Acquire")

	runWithWatchdog(t, "handler shutdown", svc.stopPeerHandling)

	select {
	case <-runDone:
	case <-time.After(time.Second):
		t.Fatalf("peer run outlived the shutdown wait:\n%s", goroutineStacks())
	}

	require.Equal(t, p2p.DiscQuitting, run(newTestP2PPeer(), newTestSession(h, 1)),
		"peer runs must be refused after the shutdown")
}

// TestPeerRun_WaitingBeforeTheHandlerStopBlocks is the counter-example that
// justifies the ordering in Service.Stop: with a peer run parked in the
// message semaphore, waiting for the runs before handler.Stop does not
// terminate. Reordering Service.Stop to wait first would deadlock the node's
// shutdown, and this test would fail.
func TestPeerRun_WaitingBeforeTheHandlerStopBlocks(t *testing.T) {
	h, svc, run := newPeerRunTestSetup(t, 8)

	require.True(t, h.msgSemaphore.Acquire(h.config.Protocol.MsgsSemaphoreLimit, time.Second))

	session := newTestSession(h, 1)
	go func() {
		_ = run(newTestP2PPeer(), session)
	}()

	<-session.delivered
	waitUntilSomeGoroutineIsIn(t, "datasemaphore.(*DataSemaphore).Acquire")

	svc.peerRuns.refuseNewRuns()

	waited := make(chan struct{})
	go func() {
		defer close(waited)
		svc.peerRuns.waitForRuns()
	}()

	select {
	case <-waited:
		t.Fatal("waiting for peer runs returned before the handler was stopped")
	case <-time.After(200 * time.Millisecond):
	}

	runWithWatchdog(t, "handler shutdown", func() {
		h.Stop()
		<-waited
	})
}

// TestPeerRun_ConcurrentRunsAreDrainedAndRefused hammers the peer-run gate
// with sessions starting and finishing while the shutdown happens, which is
// what the p2p server does to the Service: it keeps starting peer handlers
// until the node stops it, long after the Service came down. Run with -race.
func TestPeerRun_ConcurrentRunsAreDrainedAndRefused(t *testing.T) {
	const connectors = 16

	h, svc, run := newPeerRunTestSetup(t, 64)

	var completed atomic.Int64
	var refused atomic.Int64
	var connecting sync.WaitGroup

	for range connectors {
		connecting.Add(1)
		go func() {
			defer connecting.Done()
			for {
				err := run(newTestP2PPeer(), newTestSession(h, 4))
				if errors.Is(err, p2p.DiscQuitting) {
					refused.Add(1)
					return
				}
				completed.Add(1)
			}
		}()
	}

	// Let the sessions get going, so the shutdown lands in the middle of them.
	require.Eventually(t, func() bool { return completed.Load() > connectors },
		10*time.Second, time.Millisecond, "peer sessions did not make progress")

	runWithWatchdog(t, "handler shutdown", svc.stopPeerHandling)

	runWithWatchdog(t, "connector shutdown", connecting.Wait)
	require.EqualValues(t, connectors, refused.Load())
}

// newPeerRunTestSetup returns a started handler, a Service holding the peer
// run tracker, and the production peer-run entry point: the Run closure that
// MakeProtocols hands to the p2p server. Only the store and the network are
// stand-ins; the gate, the handler and its semaphores are the real ones.
func newPeerRunTestSetup(t *testing.T, maxPeers int) (*handler, *Service, func(*p2p.Peer, p2p.MsgReadWriter) error) {
	t.Helper()

	h, err := makeStartedHandler(t, maxPeers)
	require.NoError(t, err)

	// MakeProtocols only reads the store out of the Service; everything else
	// it touches on it is the peer run gate under test.
	svc := &Service{store: h.store, handler: h}

	protocols, cleanup := MakeProtocols(svc, h, enode.IterNodes(nil))
	t.Cleanup(cleanup)
	require.NotEmpty(t, protocols)

	return h, svc, protocols[0].Run
}

// testSession is a p2p.MsgReadWriter standing in for a remote peer: it
// completes the handshake, delivers a fixed number of messages, and then
// reports EOF, which ends the session like a disconnect would.
type testSession struct {
	networkID uint64
	genesis   common.Hash
	version   uint
	messages  int

	// delivered is closed once the first message after the handshake has been
	// handed to the handler, i.e. once the run is about to enter handleMsg.
	delivered chan struct{}
	once      sync.Once

	mu    sync.Mutex
	reads int
}

func newTestSession(h *handler, messages int) *testSession {
	return &testSession{
		networkID: h.NetworkID,
		genesis:   common.Hash(h.store.GetGenesisID()),
		version:   ProtocolVersions[0],
		messages:  messages,
		delivered: make(chan struct{}),
	}
}

func (s *testSession) ReadMsg() (p2p.Msg, error) {
	s.mu.Lock()
	n := s.reads
	s.reads++
	s.mu.Unlock()

	if n == 0 {
		return encodeTestMsg(HandshakeMsg, &handshakeData{
			ProtocolVersion: uint32(s.version),
			NetworkID:       s.networkID,
			Genesis:         s.genesis,
		})
	}
	if n > s.messages {
		return p2p.Msg{}, io.EOF
	}
	defer s.once.Do(func() { close(s.delivered) })
	return encodeTestMsg(ProgressMsg, &PeerProgress{})
}

func (s *testSession) WriteMsg(p2p.Msg) error { return nil }

func encodeTestMsg(code uint64, payload any) (p2p.Msg, error) {
	encoded, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return p2p.Msg{}, err
	}
	return p2p.Msg{
		Code:    code,
		Size:    uint32(len(encoded)),
		Payload: bytes.NewReader(encoded),
	}, nil
}

// newTestP2PPeer returns a peer with a Sonic-like name, so that it is not
// rejected as useless before the handshake.
func newTestP2PPeer() *p2p.Peer {
	return p2p.NewPeer(randomID(), "Sonic/shutdown-test", []p2p.Cap{})
}

// runWithWatchdog fails the test with a full goroutine dump if the given
// shutdown step does not finish, instead of hanging until the test binary's
// own timeout kills the whole package.
func runWithWatchdog(t *testing.T, what string, step func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		step()
	}()

	select {
	case <-done:
	case <-time.After(shutdownWatchdog):
		t.Fatalf("%s deadlocked after %s:\n%s", what, shutdownWatchdog, goroutineStacks())
	}
}

// waitUntilSomeGoroutineIsIn blocks until some goroutine is parked in the
// given function, so that a test can be sure it reached the state it wants to
// exercise rather than racing past it.
func waitUntilSomeGoroutineIsIn(t *testing.T, function string) {
	t.Helper()

	needle := []byte(function)
	require.Eventually(t, func() bool {
		return bytes.Contains(goroutineStacks(), needle)
	}, 10*time.Second, time.Millisecond, "no goroutine reached %s", function)
}

func goroutineStacks() []byte {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}
