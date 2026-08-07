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
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestServiceFeed_SubscribeNewBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockArchiveBlockHeightSource(ctrl)

	store.EXPECT().GetArchiveBlockHeight().Return(uint64(12), false, nil).AnyTimes()

	feed := ServiceFeed{}
	feed.Start(store)

	consumer := make(chan evmcore.ChainHeadNotify, 1)
	feed.SubscribeNewBlock(consumer)

	// There should be no signal delivered until there is a notification.
	select {
	case <-consumer:
		t.Fatal("expected no notification to be sent")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	feed.notifyAboutNewBlock(&evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(12),
		},
	}, nil)

	// The notification should be delivered.
	select {
	case notification := <-consumer:
		if notification.Block.Number.Cmp(big.NewInt(12)) != 0 {
			t.Fatalf("expected block number 12, got %d", notification.Block.Number)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected notification to be sent")
	}

	feed.Stop()
}

func TestServiceFeed_BlocksInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockArchiveBlockHeightSource(ctrl)

	var startBlocknumber uint64 = 5
	mockBlockNumber := startBlocknumber
	expectedBlockNumber := startBlocknumber + 1

	// Increment expected block height
	store.EXPECT().GetArchiveBlockHeight().DoAndReturn(func() (uint64, bool, error) {
		mockBlockNumber++
		return mockBlockNumber, false, nil
	}).AnyTimes()

	feed := ServiceFeed{}
	feed.Start(store)

	consumer := make(chan evmcore.ChainHeadNotify, 5)
	feed.SubscribeNewBlock(consumer)

	// Emit blocks
	blockNumbers := []int64{8, 6, 7, 10, 9}
	for _, blockNumber := range blockNumbers {
		feed.notifyAboutNewBlock(&evmcore.EvmBlock{
			EvmHeader: evmcore.EvmHeader{
				Number: big.NewInt(blockNumber),
			},
		}, nil)
	}

	// The notification should be delivered in correct order
	for expectedBlockNumber <= startBlocknumber+uint64(len(blockNumbers)) {
		select {
		case notification := <-consumer:
			if notification.Block.Number.Cmp(big.NewInt(int64(expectedBlockNumber))) != 0 {
				t.Fatalf("expected block number %d, got %d", expectedBlockNumber, notification.Block.Number)
			}
			expectedBlockNumber++

		case <-time.After(3 * time.Second):
			t.Fatal("expected notification should be already received")
		}
	}

	feed.Stop()
}

// Service.Stop must tear the handler down before waiting for the peer handler
// runs: a run parked in DataSemaphore.Acquire is only woken by a Release or a
// Terminate, and the Terminates live in handler.Stop.
func TestServiceStop_ReleasesAPeerRunParkedOnTheMessageSemaphore(t *testing.T) {
	env := newTestEnv(2, 1, t)
	handler := env.handler
	handler.Start(10)

	require.True(t,
		handler.msgSemaphore.TryAcquire(handler.config.Protocol.MsgsSemaphoreLimit),
		"the message semaphore should start out empty")

	// A stand-in for a p2p Protocol.Run callback, parked the way handleMsg
	// parks when the semaphore is full.
	parked := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		if !env.peerRuns.acquireRun() {
			return
		}
		defer env.peerRuns.releaseRun()
		close(parked)
		handler.msgSemaphore.Acquire(dag.Metric{Num: 1, Size: 1}, time.Hour)
	}()
	<-parked
	time.Sleep(10 * time.Millisecond) // let the run park before Stop starts

	stopped := make(chan error, 1)
	go func() {
		stopped <- env.Stop()
	}()

	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(2 * time.Minute):
		t.Fatal("Service.Stop hangs while a peer handler run is parked on the message semaphore")
	}

	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("the parked peer handler run did not return after Service.Stop")
	}
}

type expectedBlockNotification struct {
	blockNumber uint64
}

func TestServiceFeed_ArchiveState(t *testing.T) {

	tests := map[string]struct {
		blockHeight          uint64
		emptyArchive         bool
		err                  error
		expectedNotification *expectedBlockNotification
	}{
		"empty archive": {
			blockHeight:          0,
			emptyArchive:         true,
			err:                  nil,
			expectedNotification: nil,
		},
		"non-empty archive": {
			blockHeight:          12,
			emptyArchive:         false,
			err:                  nil,
			expectedNotification: &expectedBlockNotification{blockNumber: 12},
		},
		"non-existing archive": {
			blockHeight:          12,
			emptyArchive:         true,
			err:                  evmstore.NoArchiveError,
			expectedNotification: &expectedBlockNotification{blockNumber: 12},
		},
		"different archive error": {
			blockHeight:          12,
			emptyArchive:         false,
			err:                  fmt.Errorf("some other error"),
			expectedNotification: nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			store := NewMockArchiveBlockHeightSource(ctrl)

			store.EXPECT().GetArchiveBlockHeight().Return(test.blockHeight, test.emptyArchive, test.err).AnyTimes()

			feed := ServiceFeed{}
			feed.Start(store)

			consumer := make(chan evmcore.ChainHeadNotify, 1)
			feed.SubscribeNewBlock(consumer)

			feed.notifyAboutNewBlock(&evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{
					Number: big.NewInt(int64(test.blockHeight)),
				},
			}, nil)

			// The notification should be delivered.
			select {
			case notification := <-consumer:
				if test.expectedNotification == nil {
					t.Fatal("expected notification to be sent")
				} else {
					if notification.Block.Number.Cmp(big.NewInt(int64(test.expectedNotification.blockNumber))) != 0 {
						t.Fatalf("expected block number %d, got %d", test.expectedNotification.blockNumber, notification.Block.Number)
					}
				}
			// no notification should be received
			case <-time.After(100 * time.Millisecond):
				if test.expectedNotification != nil {
					t.Fatal("expected no notification to be sent")
				}
			}

			feed.Stop()
		})
	}
}
