package dagordering

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/dag/tdag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

func TestEventsBuffer(t *testing.T) {
	for try := int64(0); try < 1000; try++ {
		testEventsBuffer(t, try)
	}
}

func testEventsBuffer(t *testing.T, try int64) {
	t.Helper()
	nodes := tdag.GenNodes(5)

	var ordered dag.Events
	r := rand.New(rand.NewSource(try)) // nolint:gosec
	_ = tdag.ForEachRandEvent(nodes, 10, 3, r, tdag.ForEachEvent{
		Process: func(e dag.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e dag.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(idx.Frame(e.Seq()))
			return nil
		},
	})

	checked := 0

	processed := make(map[hash.Event]dag.Event)
	limit := dag.Metric{
		Num:  idx.Event(len(ordered)),
		Size: ordered.Metric().Size,
	}
	buffer := New(limit, Callback{

		Process: func(e dag.Event) error {
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

		Released: func(e dag.Event, peer string, err error) {
			if err != nil {
				t.Fatalf("%s unexpectedly dropped with '%s'", e.String(), err)
			}
		},

		Exists: func(id hash.Event) bool {
			return processed[id] != nil
		},

		Get: func(id hash.Event) dag.Event {
			return processed[id]
		},

		Check: func(e dag.Event, parents dag.Events) error {
			checked++
			if e.Frame() != idx.Frame(e.Seq()) {
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
	nodes := tdag.GenNodes(5)
	eventsPerNode := 1 + rand.Intn(maxEvents)/5 // nolint:gosec

	var ordered dag.Events
	_ = tdag.ForEachRandEvent(nodes, eventsPerNode, 3, rand.New(rand.NewSource(try)), tdag.ForEachEvent{ // nolint:gosec
		Process: func(e dag.Event, name string) {
			ordered = append(ordered, e)
		},
		Build: func(e dag.MutableEvent, name string) error {
			e.SetEpoch(1)
			e.SetFrame(idx.Frame(e.Seq()))
			return nil
		},
	})

	released := uint32(0)

	processed := make(map[hash.Event]dag.Event)
	var mutex sync.Mutex
	limit := dag.Metric{
		Num:  idx.Event(rand.Intn(maxEvents)),    // nolint:gosec
		Size: uint64(rand.Intn(maxEvents * 100)), // nolint:gosec
	}
	buffer := New(limit, Callback{
		Process: func(e dag.Event) error {
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

		Released: func(e dag.Event, peer string, err error) {
			mutex.Lock()
			defer mutex.Unlock()
			atomic.AddUint32(&released, 1)
		},

		Exists: func(e hash.Event) bool {
			mutex.Lock()
			defer mutex.Unlock()
			return processed[e] != nil
		},

		Get: func(e hash.Event) dag.Event {
			mutex.Lock()
			defer mutex.Unlock()
			return processed[e]
		},

		Check: func(e dag.Event, parents dag.Events) error {
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
