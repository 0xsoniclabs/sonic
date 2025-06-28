package basestreamseeder

import (
	"bytes"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/basestream"
)

func defaultConfig() Config {
	return Config{
		SenderThreads:           8,
		MaxSenderTasks:          128,
		MaxPendingResponsesSize: 64 * 1024 * 1024,
		MaxResponsePayloadNum:   100000,
		MaxResponsePayloadSize:  16 * 1024 * 1024,
		MaxResponseChunks:       12,
	}
}

func TestSeederResponsesOrder(t *testing.T) {
	for try := 0; try < 25; try++ {
		testSeederResponsesOrder(t, 1+rand.Intn(70), 1+rand.Intn(70)) // nolint:gosec
	}
}

type ResponsesContainer struct {
	mu          sync.RWMutex
	peerSession map[string][]basestream.Response
}

func locatorOf(peer string, sessionID uint32) string {
	return peer + ":" + strconv.Itoa(int(sessionID))
}

func (rr *ResponsesContainer) Get(peer string, sessionID uint32) []basestream.Response {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.peerSession[locatorOf(peer, sessionID)]
}

func (rr *ResponsesContainer) Append(peer string, sessionID uint32, response basestream.Response) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.peerSession[locatorOf(peer, sessionID)] = append(rr.peerSession[locatorOf(peer, sessionID)], response)
}

func testSeederResponsesOrder(t *testing.T, maxPeers int, maxEvents int) {
	t.Helper()
	config := defaultConfig()
	config.MaxPendingResponsesSize = 5000

	events := make(consensus.Events, maxEvents)
	for i := range events {
		e := &consensus.TestEvent{}
		e.SetEpoch(consensus.Epoch(i / 10))
		e.SetLamport(consensus.Lamport(i / 2))
		var rID [24]byte
		copy(rID[:], big.NewInt(int64(i+1)).Bytes())
		events[i] = e.Build(rID)
	}

	// sort events
	sort.Slice(events, func(i, j int) bool {
		a, b := events[i], events[j]
		return bytes.Compare(a.ID().Bytes(), b.ID().Bytes()) > 0
	})

	seeder := New(config, Callbacks{
		ForEachItem: func(start basestream.Locator, rType basestream.RequestType,
			onKey func(key basestream.Locator) bool, onAppended func(items basestream.Payload) bool) basestream.Payload {

			res := testPayload{
				IDs:    consensus.EventHashes{},
				Events: consensus.Events{},
				Size:   0,
			}

			for _, e := range events {
				if bytes.Compare(e.ID().Bytes(), start.(testLocator).B) > 0 {
					continue
				}
				if !onKey(testLocator{e.ID().Bytes()}) {
					break
				}
				res.AddEvent(e.ID(), e)
				if !onAppended(res) {
					break
				}
			}

			return res
		},
	})

	responses := ResponsesContainer{
		peerSession: map[string][]basestream.Response{},
	}

	terminated := false
	seeder.Start()
	for i := 0; i < maxPeers; i++ {
		peer := strconv.Itoa(rand.Intn(maxPeers)) // nolint:gosec
		if rand.Intn(5) == 0 {                    // nolint:gosec
			err := seeder.UnregisterPeer(peer)
			if !terminated {
				require.NoError(t, err)
			}
		} else {
			sessionID := uint32(i)
			reqType := basestream.RequestType(0)
			if sessionID%2 == 0 {
				reqType = 1
			}
			startIdx := rand.Intn(len(events)) // nolint:gosec
			endIdx := startIdx
			if endIdx < len(events) {
				endIdx += rand.Intn(len(events) - endIdx) // nolint:gosec
			}
			err, peerErr := seeder.NotifyRequestReceived(Peer{
				ID: peer,
				SendChunk: func(response basestream.Response) error {
					if reqType == 0 {
						require.Empty(t, response.Payload.(testPayload).Events)
					} else {
						require.Equal(t, len(response.Payload.(testPayload).IDs), len(response.Payload.(testPayload).Events))
					}
					if !response.Done {
						require.NotEmpty(t, response.Payload.(testPayload).IDs)
					}
					time.Sleep(time.Duration(rand.Int63n(int64(time.Millisecond)))) // nolint:gosec
					responses.Append(peer, response.SessionID, response)
					return nil
				},
				Misbehaviour: func(err error) {},
			}, basestream.Request{
				Session: basestream.Session{
					ID:    sessionID,
					Start: testLocator{events[startIdx].ID().Bytes()},
					Stop:  testLocator{events[endIdx].ID().Bytes()},
				},
				Type:           reqType,
				MaxPayloadNum:  uint32(rand.Intn(10)),                                // nolint:gosec
				MaxPayloadSize: uint64(rand.Intn(5000)),                              // nolint:gosec
				MaxChunks:      uint32(rand.Intn(int(config.MaxResponseChunks + 1))), // nolint:gosec
			})
			require.NoError(t, peerErr)
			if !terminated {
				require.NoError(t, err)
			}
		}
		if !terminated && rand.Intn(maxPeers*2) == 0 { // nolint:gosec
			terminated = true
			seeder.Stop()
		}
		require.LessOrEqual(t, atomic.LoadInt64(&seeder.pendingResponsesSize), 2*config.MaxPendingResponsesSize)
	}
	time.Sleep(5 * time.Millisecond) // give some time for seeder to finish the some of the work
	if !terminated {
		seeder.Stop()
	}
	// check that all the responses were sent in a correct order
	for _, sessionResponses := range responses.peerSession {
		prev := consensus.EventHash{}
		done := false
		for _, r := range sessionResponses {
			require.False(t, done)
			ids := r.Payload.(testPayload).IDs
			if len(r.Payload.(testPayload).Events) != 0 {
				for _, e := range r.Payload.(testPayload).Events {
					ids = append(ids, e.ID())
					require.Equal(t, e.ID().Lamport(), e.Lamport())
				}
			}
			for _, id := range ids {
				require.Equal(t, -1, bytes.Compare(prev.Bytes(), id.Bytes()))
			}
			if r.Done {
				done = true
			}
		}
	}
}
