package gossip

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
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
