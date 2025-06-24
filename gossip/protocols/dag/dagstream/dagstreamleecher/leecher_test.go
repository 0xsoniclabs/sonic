// Copyright 2025 Sonic Operations Ltd
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

package dagstreamleecher

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

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
	epoch := idx.Epoch(1)
	leecher := New(epoch, rand.IntN(2) == 0, config, Callbacks{
		IsProcessed: func(id hash.Event) bool {
			return rand.IntN(2) == 0
		},
		RequestChunk: func(peer string, r dagstream.Request) error {
			requests <- peerRequest{peer, r}
			return nil
		},
		Suspend: func(peer string) bool {
			return rand.IntN(10) == 0
		},
		PeerEpoch: func(peer string) idx.Epoch {
			return 1 + epoch/2 + idx.Epoch(rand.IntN(int(epoch*2)))
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
				err := leecher.NotifyChunkReceived(req.request.Session.ID, hash.FakeEvent(), rand.IntN(5) == 0)
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
