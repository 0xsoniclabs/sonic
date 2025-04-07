package dagordering

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestEventsBuffer(t *testing.T) {
	for try := int64(0); try < 1000; try++ {
		testEventsBuffer(t, try)
	}
}

func testEventsBuffer(t *testing.T, try int64) {
	t.Helper()
	nodes := consensustest.GenNodes(5)

	var ordered consensus.Events
	r := rand.New(rand.NewSource(try)) // nolint:gosec
	_ = consensustest.ForEachRandEvent(nodes, 10, 3, r, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e consensus.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(consensus.Frame(e.Seq()))
			return nil
		},
	})

	checked := 0

	processed := make(map[consensus.EventHash]consensus.Event)
	limit := consensus.Metric{
		Num:  uint32(len(ordered)),
		Size: ordered.Metric().Size,
	}
	buffer := New(limit, Callback{

		Process: func(e consensus.Event) error {
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
			processed[e.ID()] = e
			return nil
		},

		Released: func(e consensus.Event, peer string, err error) {
			if err != nil {
				t.Fatalf("%s unexpectedly dropped with '%s'", e.String(), err)
			}
		},

		Exists: func(id consensus.EventHash) bool {
			return processed[id] != nil
		},

		Get: func(id consensus.EventHash) consensus.Event {
			return processed[id]
		},

		Check: func(e consensus.Event, parents consensus.Events) error {
			checked++
			if e.Frame() != consensus.Frame(e.Seq()) {
				return errors.New("malformed event frame")
			}
			return nil
		},
	})

	// shuffle events
	for _, rnd := range r.Perm(len(ordered)) {
		e := ordered[rnd]
		buffer.PushEvent(e, "")
	}

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

func TestEventsBufferReleasing(t *testing.T) {
	for try := int64(0); try < 100; try++ {
		testEventsBufferReleasing(t, 200, try)
	}
}

func testEventsBufferReleasing(t *testing.T, maxEvents int, try int64) {
	t.Helper()
	nodes := consensustest.GenNodes(5)
	eventsPerNode := 1 + rand.Intn(maxEvents)/5 // nolint:gosec

	var ordered consensus.Events
	_ = consensustest.ForEachRandEvent(nodes, eventsPerNode, 3, rand.New(rand.NewSource(try)), consensustest.ForEachEvent{ // nolint:gosec
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e consensus.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(consensus.Frame(e.Seq()))
			return nil
		},
	})

	released := uint32(0)

	processed := make(map[consensus.EventHash]consensus.Event)
	var mutex sync.Mutex
	limit := consensus.Metric{
		Num:  uint32(rand.Intn(maxEvents)),       // nolint:gosec
		Size: uint64(rand.Intn(maxEvents * 100)), // nolint:gosec
	}
	buffer := New(limit, Callback{
		Process: func(e consensus.Event) error {
			mutex.Lock()
			defer mutex.Unlock()
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
			processed[e.ID()] = e
			return nil
		},

		Released: func(e consensus.Event, peer string, err error) {
			mutex.Lock()
			defer mutex.Unlock()
			atomic.AddUint32(&released, 1)
		},

		Exists: func(e consensus.EventHash) bool {
			mutex.Lock()
			defer mutex.Unlock()
			return processed[e] != nil
		},

		Get: func(e consensus.EventHash) consensus.Event {
			mutex.Lock()
			defer mutex.Unlock()
			return processed[e]
		},

		Check: func(e consensus.Event, parents consensus.Events) error {
			mutex.Lock()
			defer mutex.Unlock()
			if rand.Intn(10) == 0 { // nolint:gosec
				return errors.New("testing error")
			}
			if rand.Intn(10) == 0 { // nolint:gosec
				time.Sleep(time.Microsecond * 100)
			}
			return nil
		},
	})

	// duplicate some events
	ordered = append(ordered, ordered[:rand.Intn(len(ordered))]...) // nolint:gosec
	// shuffle events
	wg := sync.WaitGroup{}
	for _, rnd := range rand.Perm(len(ordered)) {
		e := ordered[rnd]
		wg.Add(1)
		go func() {
			defer wg.Done()
			buffer.PushEvent(e, "")
			if rand.Intn(10) == 0 { // nolint:gosec
				buffer.Clear()
			}
		}()
	}
	wg.Wait()
	buffer.Clear()

	// everything is released
	if uint32(len(ordered)) != released {
		t.Fatal("not all the events were released", len(ordered), released)
	}
}
