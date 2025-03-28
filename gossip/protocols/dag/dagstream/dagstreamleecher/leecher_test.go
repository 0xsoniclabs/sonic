package dagstreamleecher

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
)

func TestLeecherNoDeadlocks(t *testing.T) {
	for try := 0; try < 10; try++ {
		testLeecherNoDeadlocks(t, 1+rand.IntN(500))
	}
}

type peerRequest struct {
	peer    string
	request dagstream.Request
}

func testLeecherNoDeadlocks(t *testing.T, maxPeers int) {
	requests := make(chan peerRequest, 1000)
	config := LiteConfig()
	config.RecheckInterval = time.Millisecond * 5
	config.MinSessionRestart = 2 * time.Millisecond * 5
	config.MaxSessionRestart = 5 * time.Millisecond * 5
	config.BaseProgressWatchdog = 3 * time.Millisecond * 5
	config.Session.RecheckInterval = time.Millisecond
	epoch := consensus.Epoch(1)
	leecher := New(epoch, rand.IntN(2) == 0, config, Callbacks{
		IsProcessed: func(id consensus.EventHash) bool {
			return rand.IntN(2) == 0
		},
		RequestChunk: func(peer string, r dagstream.Request) error {
			requests <- peerRequest{peer, r}
			return nil
		},
		Suspend: func(peer string) bool {
			return rand.IntN(10) == 0
		},
		PeerEpoch: func(peer string) consensus.Epoch {
			return 1 + epoch/2 + consensus.Epoch(rand.IntN(int(epoch*2)))
		},
	})
	terminated := false
	for i := 0; i < maxPeers*2; i++ {
		peer := strconv.Itoa(rand.IntN(maxPeers))
		coin := rand.IntN(100)
		if coin <= 50 {
			err := leecher.RegisterPeer(peer)
			if !terminated {
				require.NoError(t, err)
			}
		} else if coin <= 60 {
			err := leecher.UnregisterPeer(peer)
			if !terminated {
				require.NoError(t, err)
			}
		} else if coin <= 65 {
			epoch++
			leecher.OnNewEpoch(epoch)
		} else if coin <= 70 {
			leecher.ForceSyncing()
		} else {
			time.Sleep(time.Millisecond)
		}
		select {
		case req := <-requests:
			if rand.IntN(10) != 0 {
				err := leecher.NotifyChunkReceived(req.request.Session.ID, consensus.FakeEvent(), rand.IntN(5) == 0)
				if !terminated {
					require.NoError(t, err)
				}
			}
		default:
		}
		if !terminated && rand.IntN(maxPeers*2) == 0 {
			terminated = true
			leecher.Terminate()
		}
	}
	if !terminated {
		leecher.Stop()
	} else {
		leecher.Wg.Wait()
	}
}
