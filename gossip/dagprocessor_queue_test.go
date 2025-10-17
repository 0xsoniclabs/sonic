package gossip

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/gossip/dagordering"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
)

var factor int

func init() {

	factorString := os.Getenv("SIZE_FACTOR")
	// parse factor into int
	var err error
	factor, err = strconv.Atoi(factorString)
	if err != nil {
		panic(fmt.Sprintf("invalid SIZE_FACTOR: %s", factorString))
	}

	if factor == 0 {
		panic("factor is zero")
	}

	config := DefaultConfig(cachescale.Identity)
	fmt.Println("buffer limits:", config.Protocol.DagProcessor.EventsBufferLimit.Num*idx.Event(factor), config.Protocol.DagProcessor.EventsBufferLimit.Size*uint64(factor*2))
	fmt.Println("factor", factor)

}

func BenchmarkDagProcessorQueue_fluid(b *testing.B) {
	genericBenchmark(b, happyCase, 3250)
}
func BenchmarkDagProcessorQueue_hf_hiccups(b *testing.B) {
	genericBenchmark(b, hiccupsGenerator(10), 3250)
}

func BenchmarkDagProcessorQueue_lf_hiccups(b *testing.B) {
	genericBenchmark(b, hiccupsGenerator(1000), 3250)
}

func BenchmarkDagProcessorQueue_jam(b *testing.B) {
	genericBenchmark(b, linearChainGenerator, 3250)
}

func BenchmarkDagProcessorQueue_fluid_full(b *testing.B) {
	genericBenchmark(b, happyCase, 3250*factor)
}
func BenchmarkDagProcessorQueue_hf_hiccups_full(b *testing.B) {
	genericBenchmark(b, hiccupsGenerator(10), 3250*factor)
}

func BenchmarkDagProcessorQueue_lf_hiccups_full(b *testing.B) {
	genericBenchmark(b, hiccupsGenerator(1000), 3250*factor)
}

func BenchmarkDagProcessorQueue_jam_full(b *testing.B) {
	genericBenchmark(b, linearChainGenerator, 3250*factor)
}

// linearChainGenerator generates events in reverse order
// so that each event depends on the previous one,
// creating a linear chain, which can only be processed once the last event is
// received.
func linearChainGenerator(size int) []*testEvent {
	res := make([]*testEvent, 0, size)
	for i := range size {
		e := NewtestEvent(idx.Event(i))
		res = append(res, e)
	}
	slices.Reverse(res)
	return res
}

func hiccupsGenerator(chunkSize int) generator {

	return func(size int) []*testEvent {
		res := make([]*testEvent, 0, size)
		// first generate in order of processing
		for i := range size {
			res = append(res, NewtestEvent(idx.Event(i)))
		}

		// reverse chunks
		for i := 0; i < size; i += chunkSize {
			end := i + chunkSize
			if end > size {
				end = size
			}
			slices.Reverse(res[i:end])
		}

		return res
	}
}

// happyCase generates events in the order they can be processed
// as they are received.
func happyCase(size int) []*testEvent {
	res := make([]*testEvent, 0, size)
	for i := range size {
		e := NewtestEvent(idx.Event(i))
		res = append(res, e)
	}
	return res
}

func genericBenchmark(b *testing.B, inputGen generator, numEvents int) {

	config := DefaultConfig(cachescale.Identity)

	eventsBufferLimit := dag.Metric{
		Num:  config.Protocol.DagProcessor.EventsBufferLimit.Num * idx.Event(factor),
		Size: config.Protocol.DagProcessor.EventsBufferLimit.Size * uint64(factor*2),
	}

	inputEvents := inputGen(int(numEvents))

	var completed map[hash.Event]dag.Event
	noParent := errors.New("no parent processed yet")
	queue := dagordering.New(eventsBufferLimit, dagordering.Callback{
		Process: func(e dag.Event) error {
			completed[e.ID()] = e
			return nil
		},
		Released: func(e dag.Event, peer string, err error) {},
		Get: func(id hash.Event) dag.Event {
			return completed[id]
		},
		Exists: func(id hash.Event) bool {
			_, ok := completed[id]
			return ok
		},
		Check: func(e dag.Event, parents dag.Events) error {
			for _, p := range parents {
				if _, ok := completed[p.ID()]; !ok {
					return noParent
				}
			}
			return nil
		},
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		completed = make(map[hash.Event]dag.Event, numEvents)

		b.StartTimer()
		for _, e := range inputEvents {
			queue.PushEvent(e, "peer")
		}
	}
	b.StopTimer()

	b.ReportMetric(float64(numEvents), "events_per_run")
	b.ReportMetric(float64(factor), "factor")

	// Check if all events were processed
	if len(completed) != len(inputEvents) {
		b.Fatalf("not all events were processed: got %d, want %d", len(completed), len(inputEvents))
	}
}

type generator func(int) []*testEvent

//================= Test Event Implementation =================//

type testEvent struct {
	i       idx.Event
	parents hash.Events
}

func NewtestEvent(i idx.Event) *testEvent {
	var parents hash.Events
	if i != 0 {
		parents = hash.Events{{
			0: byte(i - 1),
			1: byte((i - 1) >> 8),
			2: byte((i - 1) >> 16),
			3: byte((i - 1) >> 24),
		}}
	}

	return &testEvent{
		i:       i,
		parents: parents,
	}
}

func (e *testEvent) Epoch() idx.Epoch {
	return 0
}

func (e *testEvent) Seq() idx.Event {
	return e.i
}

func (e *testEvent) Frame() idx.Frame {
	return 0
}

func (e *testEvent) Creator() idx.ValidatorID {
	return 0
}

func (e *testEvent) Lamport() idx.Lamport {
	return 0
}
func (e *testEvent) Parents() hash.Events {
	return e.parents
}

func (e *testEvent) SelfParent() *hash.Event {
	return &e.parents[0]
}

func (e *testEvent) IsSelfParent(hash hash.Event) bool {
	return e.parents[0] == hash
}

func (e *testEvent) ID() hash.Event {
	return hash.Event{
		0: byte(e.i),
		1: byte(e.i >> 8),
		2: byte(e.i >> 16),
		3: byte(e.i >> 24),
	}
}

func (e *testEvent) String() string {
	return "testEvent"
}

func (e *testEvent) Size() int {
	return 1
}
