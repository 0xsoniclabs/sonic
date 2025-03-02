package dagprocessor

import (
	"errors"
	"sync"

	"github.com/0xsoniclabs/consensus/eventcheck"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/utils/datasemaphore"
	"github.com/0xsoniclabs/sonic/gossip/dagordering"
	"github.com/0xsoniclabs/sonic/utils/workers"
)

var (
	ErrBusy = errors.New("failed to acquire events semaphore")
)

// Processor is responsible for processing incoming events
type Processor struct {
	cfg Config

	quit chan struct{}
	wg   sync.WaitGroup

	callback Callback

	checker         *workers.Workers
	orderedInserter *workers.Workers

	buffer *dagordering.EventsBuffer

	eventsSemaphore *datasemaphore.DataSemaphore
}

type EventCallback struct {
	Process         func(e dag.Event) error
	Released        func(e dag.Event, peer string, err error)
	Get             func(hash.Event) dag.Event
	Exists          func(hash.Event) bool
	CheckParents    func(e dag.Event, parents dag.Events) error
	CheckParentless func(e dag.Event, checked func(error))
}

type Callback struct {
	Event          EventCallback
	HighestLamport func() idx.Lamport
}

// New creates an event processor
func New(eventsSemaphore *datasemaphore.DataSemaphore, cfg Config, callback Callback) *Processor {
	f := &Processor{
		cfg:             cfg,
		quit:            make(chan struct{}),
		eventsSemaphore: eventsSemaphore,
	}
	released := callback.Event.Released
	callback.Event.Released = func(e dag.Event, peer string, err error) {
		f.eventsSemaphore.Release(dag.Metric{Num: 1, Size: uint64(e.Size())})
		if released != nil {
			released(e, peer, err)
		}
	}
	f.callback = callback
	f.buffer = dagordering.New(cfg.EventsBufferLimit, dagordering.Callback{
		Process:  callback.Event.Process,
		Released: callback.Event.Released,
		Get:      callback.Event.Get,
		Exists:   callback.Event.Exists,
		Check:    callback.Event.CheckParents,
	})
	f.orderedInserter = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	f.checker = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	return f
}

// Start boots up the events processor.
func (f *Processor) Start() {
	f.orderedInserter.Start(1)
	f.checker.Start(1)
}

// Stop interrupts the processor, canceling all the pending operations.
// Stop waits until all the internal goroutines have finished.
func (f *Processor) Stop() {
	close(f.quit)
	f.eventsSemaphore.Terminate()
	f.wg.Wait()
	f.buffer.Clear()
}

// Overloaded returns true if too much events are being processed or requested
func (f *Processor) Overloaded() bool {
	return f.checker.TasksCount() > f.cfg.MaxTasks*3/4 ||
		f.orderedInserter.TasksCount() > f.cfg.MaxTasks*3/4
}

type checkRes struct {
	e   dag.Event
	err error
	pos idx.Event
}

func (f *Processor) Enqueue(peer string, events dag.Events, ordered bool, notifyAnnounces func(hash.Events), done func()) error {
	if !f.eventsSemaphore.Acquire(events.Metric(), f.cfg.EventsSemaphoreTimeout) {
		return ErrBusy
	}

	checkedC := make(chan *checkRes, len(events))
	err := f.checker.Enqueue(func() {
		for i, e := range events {
			pos := idx.Event(i)
			event := e
			f.callback.Event.CheckParentless(event, func(err error) {
				checkedC <- &checkRes{
					e:   event,
					err: err,
					pos: pos,
				}
			})
		}
	})
	if err != nil {
		return err
	}
	eventsLen := len(events)
	return f.orderedInserter.Enqueue(func() {
		if done != nil {
			defer done()
		}

		var orderedResults []*checkRes
		if ordered {
			orderedResults = make([]*checkRes, eventsLen)
		}
		var processed int
		var toRequest hash.Events
		for processed < eventsLen {
			select {
			case res := <-checkedC:
				if ordered {
					orderedResults[res.pos] = res

					for i := processed; processed < len(orderedResults) && orderedResults[i] != nil; i++ {
						toRequest = append(toRequest, f.process(peer, orderedResults[i].e, orderedResults[i].err)...)
						orderedResults[i] = nil // free the memory
						processed++
					}
				} else {
					toRequest = append(toRequest, f.process(peer, res.e, res.err)...)
					processed++
				}

			case <-f.quit:
				return
			}
		}

		// request unknown event parents
		if notifyAnnounces != nil && len(toRequest) != 0 {
			notifyAnnounces(toRequest)
		}
	})
}

func (f *Processor) process(peer string, event dag.Event, resErr error) (toRequest hash.Events) {
	// release event if failed validation
	if resErr != nil {
		f.callback.Event.Released(event, peer, resErr)
		return hash.Events{}
	}
	// release event if it's too far in future
	highestLamport := f.callback.HighestLamport()
	maxLamportDiff := 1 + idx.Lamport(f.cfg.EventsBufferLimit.Num)
	if event.Lamport() > highestLamport+maxLamportDiff {
		f.callback.Event.Released(event, peer, eventcheck.ErrSpilledEvent)
		return hash.Events{}
	}
	// push event to the ordering buffer
	complete := f.buffer.PushEvent(event, peer)
	if !complete && event.Lamport() <= highestLamport+maxLamportDiff/10 {
		return event.Parents()
	}
	return hash.Events{}
}

func (f *Processor) IsBuffered(id hash.Event) bool {
	return f.buffer.IsBuffered(id)
}

func (f *Processor) Clear() {
	f.buffer.Clear()
}

func (f *Processor) TotalBuffered() dag.Metric {
	return f.buffer.Total()
}

func (f *Processor) TasksCount() int {
	return f.orderedInserter.TasksCount() + f.checker.TasksCount()
}
