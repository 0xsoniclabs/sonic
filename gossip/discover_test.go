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
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStartENRUpdater_UpdatesNodeRecordOnNewBlock(t *testing.T) {
	require := require.New(t)
	svc := newENRUpdaterTestService(t)

	record := newBlockingNodeRecord()
	close(record.release) // < do not block the updater in this test
	StartENRUpdater(svc, record)

	svc.feed.notifyAboutNewBlock(newTestBlock(1), nil)

	select {
	case entry := <-record.updates:
		// The entry is derived from data read from the gossip store, so its
		// delivery proves that the updater is a store reader.
		require.IsType(&enrEntry{}, entry)
	case <-time.After(30 * time.Second):
		require.Fail("node record was not updated")
	}

	svc.stopStoreReaders()
}

// TestStartENRUpdater_ShutdownWaitsForRunningUpdater covers the shutdown
// ordering the updater depends on: it reads from the gossip store on every
// chain head notification, so it must have terminated before Service.Stop
// closes the store. Without the join, the updater could still be accessing the
// store while Store.Close nils out its tables and caches.
func TestStartENRUpdater_ShutdownWaitsForRunningUpdater(t *testing.T) {
	require := require.New(t)
	svc := newENRUpdaterTestService(t)

	record := newBlockingNodeRecord()
	StartENRUpdater(svc, record)

	svc.feed.notifyAboutNewBlock(newTestBlock(1), nil)

	// Wait for the updater to enter the notification handler. It is pinned
	// there, mid-iteration, until the test releases it.
	<-record.updates

	stopped := runAsync(svc.stopStoreReaders)

	select {
	case <-stopped:
		require.Fail("shutdown returned while the ENR updater was still running")
	case <-time.After(100 * time.Millisecond):
		// all good, the shutdown is waiting for the updater
	}

	close(record.release)

	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		require.Fail("shutdown did not return after the ENR updater finished")
	}
}

// TestStartENRUpdater_ShutdownTerminatesWithBackloggedNotifications exercises
// the shutdown path while the feed producer is blocked in a send to the
// updater's full notification channel. Stopping the feed has to unblock the
// producer, the producer has to unblock the feed's shutdown, and only then can
// the updater be signaled -- a chain in which every step waits for the previous
// one.
func TestStartENRUpdater_ShutdownTerminatesWithBackloggedNotifications(t *testing.T) {
	require := require.New(t)
	svc := newENRUpdaterTestService(t)

	record := newBlockingNodeRecord()
	StartENRUpdater(svc, record)

	// The updater's notification channel holds 10 events; sending more than
	// that blocks the feed's producer goroutine inside the notification send
	// while the updater is pinned in the handler.
	const notifications = 25
	for i := range notifications {
		svc.feed.notifyAboutNewBlock(newTestBlock(int64(i+1)), nil)
	}
	<-record.updates

	stopped := runAsync(svc.stopStoreReaders)

	close(record.release)

	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		require.Fail("shutdown deadlocked with backlogged notifications")
	}
}

func TestStartENRUpdater_ShutdownIsDeadlockFreeWhileNotificationsAreInFlight(t *testing.T) {
	require := require.New(t)

	// The updater is stopped at an arbitrary point of its notification
	// processing, covering the interleavings between an in-flight notification
	// and the shutdown signal.
	for _, delay := range []time.Duration{0, time.Microsecond, 10 * time.Microsecond, time.Millisecond} {
		svc := newENRUpdaterTestService(t)

		record := newBlockingNodeRecord()
		close(record.release) // < do not block the updater in this test
		StartENRUpdater(svc, record)

		for i := range 25 {
			svc.feed.notifyAboutNewBlock(newTestBlock(int64(i+1)), nil)
		}
		time.Sleep(delay)

		stopped := runAsync(svc.stopStoreReaders)

		select {
		case <-stopped:
		case <-time.After(30 * time.Second):
			require.Failf("shutdown deadlocked", "delay: %v", delay)
		}
	}
}

func TestStartENRUpdater_ShutdownDoesNotBlockIfUpdaterWasNeverStarted(t *testing.T) {
	require := require.New(t)
	svc := newENRUpdaterTestService(t)

	// Service.Stop runs the same sequence when Service.Start failed before the
	// ENR updater was started.
	stopped := runAsync(svc.stopStoreReaders)

	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		require.Fail("shutdown blocked although no store reader was started")
	}
}

func runAsync(f func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	return done
}

func newTestBlock(number int64) *evmcore.EvmBlock {
	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{Number: big.NewInt(number)},
	}
}

func newENRUpdaterTestService(tb testing.TB) *Service {
	tb.Helper()

	store := newInMemoryStoreWithGenesisData(tb, opera.GetSonicUpgrades(), 1, 1)

	archive := NewMockArchiveBlockHeightSource(gomock.NewController(tb))
	archive.EXPECT().GetArchiveBlockHeight().Return(^uint64(0), false, nil).AnyTimes()

	svc := &Service{store: store}
	svc.feed.Start(archive)

	return svc
}

// blockingNodeRecord is a node record whose update blocks the ENR updater
// inside a notification handler until the test releases it.
type blockingNodeRecord struct {
	updates chan enr.Entry
	release chan struct{}
}

func newBlockingNodeRecord() *blockingNodeRecord {
	return &blockingNodeRecord{
		updates: make(chan enr.Entry, 1024),
		release: make(chan struct{}),
	}
}

func (r *blockingNodeRecord) Set(entry enr.Entry) {
	r.updates <- entry
	<-r.release
}
