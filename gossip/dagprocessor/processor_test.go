package dagprocessor

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"

	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/sonic/utils/datasemaphore"
)

func TestProcessor(t *testing.T) {
	for try := 0; try < 500; try++ {
		testProcessor(t)
	}
}

var maxGroupSize = consensus.Metric{
	Num:  50,
	Size: 50 * 50,
}

func shuffleEventsIntoChunks(inEvents consensus.Events) []consensus.Events {
	if len(inEvents) == 0 {
		return nil
	}
	var chunks []consensus.Events
	var lastChunk consensus.Events
	var lastChunkSize consensus.Metric
	for _, rnd := range rand.Perm(len(inEvents)) {
		e := inEvents[rnd]
		if rand.Intn(10) == 0 || lastChunkSize.Num+1 >= maxGroupSize.Num || lastChunkSize.Size+uint64(e.Size()) >= maxGroupSize.Size { // nolint:gosec
			chunks = append(chunks, lastChunk)
			lastChunk = consensus.Events{}
		}
		lastChunk = append(lastChunk, e)
		lastChunkSize.Num++
		lastChunkSize.Size += uint64(e.Size())
	}
	chunks = append(chunks, lastChunk)
	return chunks
}

func testProcessor(t *testing.T) {
	t.Helper()
	nodes := consensustest.GenNodes(5)

	var ordered consensus.Events
	_ = consensustest.ForEachRandEvent(nodes, 10, 3, nil, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e consensus.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(consensus.Frame(e.Seq()))
			return nil
		},
	})

	limit := consensus.Metric{
		Num:  consensus.Seq(len(ordered)),
		Size: ordered.Metric().Size,
	}
	semaphore := datasemaphore.New(limit, func(received consensus.Metric, processing consensus.Metric, releasing consensus.Metric) {
		t.Fatal("events semaphore inconsistency")
	})
	config := DefaultConfig(cachescale.Identity)
	config.EventsBufferLimit = limit

	checked := 0

	highestLamport := consensus.Lamport(0)
	processed := make(map[consensus.EventHash]consensus.Event)
	mu := sync.RWMutex{}
	processor := New(semaphore, config, Callback{
		Event: EventCallback{
			Process: func(e consensus.Event) error {
				mu.Lock()
				defer mu.Unlock()
				if _, ok := processed[e.ID()]; ok {
					t.Fatalf("%s already processed", e.String())
					return nil
				}
				for _, p := range e.Parents() {
					if _, ok := processed[p]; !ok {
						t.Fatalf("got %s before parent %s", e.String(), p.String())
						return nil
					}
				}
				if highestLamport < e.Lamport() {
					highestLamport = e.Lamport()
				}
				processed[e.ID()] = e
				return nil
			},

			Released: func(e consensus.Event, peer string, err error) {
				if err != nil {
					t.Fatalf("%s unexpectedly dropped with '%s'", e.String(), err)
				}
			},

			Exists: func(e consensus.EventHash) bool {
				mu.RLock()
				defer mu.RUnlock()
				return processed[e] != nil
			},

			Get: func(id consensus.EventHash) consensus.Event {
				mu.RLock()
				defer mu.RUnlock()
				return processed[id]
			},

			CheckParents: func(e consensus.Event, parents consensus.Events) error {
				mu.RLock()
				defer mu.RUnlock()
				checked++
				if e.Frame() != consensus.Frame(e.Seq()) {
					return errors.New("malformed event frame")
				}
				return nil
			},
			CheckParentless: func(e consensus.Event, checked func(err error)) {
				checked(nil)
			},
		},
		HighestLamport: func() consensus.Lamport {
			return highestLamport
		},
	})
	// shuffle events
	chunks := shuffleEventsIntoChunks(ordered)

	// process events
	processor.Start()
	wg := sync.WaitGroup{}
	for _, chunk := range chunks {
		wg.Add(1)
		err := processor.Enqueue("", chunk, rand.Intn(2) == 0, func(events consensus.EventHashes) {}, func() { // nolint:gosec
			wg.Done()
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	processor.Stop()

	// everything is processed
	for _, e := range ordered {
		if _, ok := processed[e.ID()]; !ok {
			t.Fatal("event wasn't processed")
		}
	}
	if checked != len(processed) {
		t.Fatal("not all the events were checked")
	}
}

func TestProcessorReleasing(t *testing.T) {
	for try := int64(0); try < 100; try++ {
		testProcessorReleasing(t, 200, try)
	}
}

func testProcessorReleasing(t *testing.T, maxEvents int, try int64) {
	t.Helper()
	nodes := consensustest.GenNodes(5)

	var ordered consensus.Events
	_ = consensustest.ForEachRandEvent(nodes, 10, 3, rand.New(rand.NewSource(try)), consensustest.ForEachEvent{ // nolint:gosec
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e consensus.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(consensus.Frame(e.Seq()))
			return nil
		},
	})

	limit := consensus.Metric{
		Num:  consensus.Seq(rand.Intn(maxEvents)), // nolint:gosec
		Size: uint64(rand.Intn(maxEvents * 100)),  // nolint:gosec
	}
	limitPlus1group := consensus.Metric{
		Num:  limit.Num + maxGroupSize.Num,
		Size: limit.Size + maxGroupSize.Size,
	}
	semaphore := datasemaphore.New(limitPlus1group, func(received consensus.Metric, processing consensus.Metric, releasing consensus.Metric) {
		t.Fatal("events semaphore inconsistency")
	})
	config := DefaultConfig(cachescale.Identity)
	config.EventsBufferLimit = limit

	released := uint32(0)

	highestLamport := consensus.Lamport(0)
	processed := make(map[consensus.EventHash]consensus.Event)
	mu := sync.RWMutex{}
	processor := New(semaphore, config, Callback{
		Event: EventCallback{
			Process: func(e consensus.Event) error {
				mu.Lock()
				defer mu.Unlock()
				if _, ok := processed[e.ID()]; ok {
					t.Fatalf("%s already processed", e.String())
					return nil
				}
				for _, p := range e.Parents() {
					if _, ok := processed[p]; !ok {
						t.Fatalf("got %s before parent %s", e.String(), p.String())
						return nil
					}
				}
				if rand.Intn(10) == 0 { // nolint:gosec
					return errors.New("testing error")
				}
				if rand.Intn(10) == 0 { // nolint:gosec
					time.Sleep(time.Microsecond * 100)
				}
				if highestLamport < e.Lamport() {
					highestLamport = e.Lamport()
				}
				processed[e.ID()] = e
				return nil
			},

			Released: func(e consensus.Event, peer string, err error) {
				mu.Lock()
				defer mu.Unlock()
				atomic.AddUint32(&released, 1)
			},

			Exists: func(e consensus.EventHash) bool {
				mu.RLock()
				defer mu.RUnlock()
				return processed[e] != nil
			},

			Get: func(id consensus.EventHash) consensus.Event {
				mu.RLock()
				defer mu.RUnlock()
				return processed[id]
			},

			CheckParents: func(e consensus.Event, parents consensus.Events) error {
				if rand.Intn(10) == 0 { // nolint:gosec
					return errors.New("testing error")
				}
				if rand.Intn(10) == 0 { // nolint:gosec
					time.Sleep(time.Microsecond * 100)
				}
				return nil
			},
			CheckParentless: func(e consensus.Event, checked func(err error)) {
				var err error
				if rand.Intn(10) == 0 { // nolint:gosec
					err = errors.New("testing error")
				}
				if rand.Intn(10) == 0 { // nolint:gosec
					time.Sleep(time.Microsecond * 100)
				}
				checked(err)
			},
		},
		HighestLamport: func() consensus.Lamport {
			return highestLamport
		},
	})
	// duplicate some events
	ordered = append(ordered, ordered[:rand.Intn(len(ordered))]...) // nolint:gosec
	// shuffle events
	chunks := shuffleEventsIntoChunks(ordered)

	// process events
	processor.Start()
	wg := sync.WaitGroup{}
	for _, chunk := range chunks {
		wg.Add(1)
		err := processor.Enqueue("", chunk, rand.Intn(2) == 0, func(events consensus.EventHashes) {}, func() { // nolint:gosec
			wg.Done()
		})
		if err != nil {
			t.Fatal(err)
		}
		if rand.Intn(10) == 0 { // nolint:gosec
			processor.Clear()
		}
	}
	wg.Wait()
	processor.Clear()
	if processor.eventsSemaphore.Processing().Num != 0 {
		t.Fatal("not all the events were released", processor.eventsSemaphore.Processing().Num)
	}
	processor.Stop()

	// everything is released
	if uint32(len(ordered)) != released {
		t.Fatal("not all the events were released", len(ordered), released)
	}
}
