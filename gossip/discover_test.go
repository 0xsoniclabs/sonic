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

// TestStartENRUpdater_StopWaitsForUpdaterToFinishStoreAccess covers the
// shutdown ordering the updater depends on: it reads from the gossip store on
// every chain head notification, so it must be joined before the store is
// closed. Without the join the updater would still be accessing the store while
// Store.Close nils out its tables and caches.
func TestStartENRUpdater_StopWaitsForUpdaterToFinishStoreAccess(t *testing.T) {
	require := require.New(t)
	svc := newENRUpdaterTestService(t)

	record := newBlockingNodeRecord()
	stop := StartENRUpdater(svc, record)

	svc.feed.notifyAboutNewBlock(&evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{Number: big.NewInt(1)},
	}, nil)

	// Wait for the updater to be inside the handler that reads the store. The
	// entry it delivers is the result of those reads, so by the time the record
	// blocks, the handler is provably in the middle of its store access.
	entry := <-record.updates
	require.IsType(&enrEntry{}, entry)

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		stop()
	}()

	select {
	case <-stopped:
		require.Fail("stop returned while the updater was still accessing the store")
	case <-time.After(100 * time.Millisecond):
	}

	close(record.release)

	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		require.Fail("stop did not return after the updater finished")
	}

	svc.feed.Stop()
}

func TestStartENRUpdater_StopCanBeCalledRepeatedly(t *testing.T) {
	svc := newENRUpdaterTestService(t)
	stop := StartENRUpdater(svc, newBlockingNodeRecord())

	stop()
	stop()

	svc.feed.Stop()
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

// blockingNodeRecord is a node record whose update blocks the caller until the
// test releases it, pinning the ENR updater inside a notification handler.
type blockingNodeRecord struct {
	updates chan enr.Entry
	release chan struct{}
}

func newBlockingNodeRecord() *blockingNodeRecord {
	return &blockingNodeRecord{
		updates: make(chan enr.Entry),
		release: make(chan struct{}),
	}
}

func (r *blockingNodeRecord) Set(entry enr.Entry) {
	r.updates <- entry
	<-r.release
}
