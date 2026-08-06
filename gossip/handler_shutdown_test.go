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

// The tests in this file drive the real peer run entry point -- the Run closure
// MakeProtocols hands to the p2p server -- against the real Service.Stop, with
// only the network mocked out. They cover what the peerRunTracker unit tests
// cannot: that the teardown really does release the runs it then waits for.
// Both are worth running with -race, and neither may hang: the shutdown runs
// under a watchdog that fails with goroutine stacks instead.

const shutdownWatchdog = 30 * time.Second

// TestServiceStop_ReleasesAPeerRunParkedOnTheMessageSemaphore parks a real peer
// run in the message semaphore and requires the full Service shutdown to come
// down anyway. The parked run cannot time out on its own -- Acquire only
// re-checks its deadline when the semaphore is signalled -- so it is released
// only if Stop tears the handler down before waiting for the runs.
func TestServiceStop_ReleasesAPeerRunParkedOnTheMessageSemaphore(t *testing.T) {
	env, run, newSession := newPeerRunTestSetup(t, 8)
	handler := env.handler

	// Saturate the message semaphore, so the first message of the session
	// below parks the peer run instead of being handled.
	require.True(t, handler.msgSemaphore.TryAcquire(handler.config.Protocol.MsgsSemaphoreLimit))

	session := newSession(1)
	runDone := make(chan error, 1)
	go func() {
		runDone <- run(newTestP2PPeer(), session)
	}()

	<-session.delivered
	waitUntilSomeGoroutineIsIn(t, "datasemaphore.(*DataSemaphore).Acquire")

	stopWithWatchdog(t, env)

	select {
	case <-runDone:
	case <-time.After(time.Second):
		t.Fatalf("peer run outlived the shutdown wait:\n%s", goroutineStacks())
	}

	require.Equal(t, p2p.DiscQuitting, run(newTestP2PPeer(), newSession(1)),
		"peer runs must be refused after the shutdown")
}

// TestServiceStop_DrainsAndRefusesConcurrentPeerRuns hammers the peer run gate
// with sessions starting and finishing while the shutdown happens, which is
// what the p2p server does to the Service: it keeps starting peer handlers
// until the node stops it, long after the Service came down. Run with -race.
func TestServiceStop_DrainsAndRefusesConcurrentPeerRuns(t *testing.T) {
	const connectors = 16

	env, run, newSession := newPeerRunTestSetup(t, 64)

	var completed atomic.Int64
	var refused atomic.Int64
	var connecting sync.WaitGroup

	for range connectors {
		connecting.Add(1)
		go func() {
			defer connecting.Done()
			for {
				err := run(newTestP2PPeer(), newSession(4))
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

	stopWithWatchdog(t, env)

	runWithWatchdog(t, "connector shutdown", connecting.Wait)
	require.EqualValues(t, connectors, refused.Load())
}

type peerRun func(*p2p.Peer, p2p.MsgReadWriter) error

// newPeerRunTestSetup returns a Service with a started handler, the production
// peer run entry point, and a factory for the sessions to feed it. Only the
// network is a stand-in; the gate, the handler, its semaphores and the Stop
// under test are the real ones.
func newPeerRunTestSetup(t *testing.T, maxPeers int) (*testEnv, peerRun, func(messages int) *testSession) {
	t.Helper()

	env := newTestEnv(2, 1, t)
	env.handler.Start(maxPeers)

	// The Service's own dial candidates are not passed in: MakeProtocols would
	// close them with the returned cleanup, and Service.Stop closes them too.
	protocols, cleanup := MakeProtocols(env.Service, env.handler, enode.IterNodes(nil))
	t.Cleanup(cleanup)
	require.NotEmpty(t, protocols)

	// The handshake values are read once, so that sessions created while the
	// Service shuts down do not touch the store it is committing.
	networkID := env.handler.NetworkID
	genesis := common.Hash(env.store.GetGenesisID())
	newSession := func(messages int) *testSession {
		return &testSession{
			networkID: networkID,
			genesis:   genesis,
			version:   ProtocolVersions[0],
			messages:  messages,
			delivered: make(chan struct{}),
		}
	}

	return env, protocols[0].Run, newSession
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

// stopWithWatchdog runs the production shutdown and requires it to come down.
func stopWithWatchdog(t *testing.T, env *testEnv) {
	t.Helper()

	stopped := make(chan error, 1)
	runWithWatchdog(t, "Service.Stop", func() { stopped <- env.Stop() })
	require.NoError(t, <-stopped)
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
