package dagordering

import (
	"math"
	"sync"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/cacheutils/wlru"
	eventcheck "github.com/0xsoniclabs/sonic/eventcheck/base"
)

type (
	// event is a inter.Event and data for ordering purpose.
	event struct {
		event consensus.Event

		peer     string
		err      error
		released bool
	}

	// Callback is a set of EventsBuffer()'s args.
	Callback struct {
		Process  func(e consensus.Event) error
		Released func(e consensus.Event, peer string, err error)
		Get      func(consensus.EventHash) consensus.Event
		Exists   func(consensus.EventHash) bool
		Check    func(e consensus.Event, parents consensus.Events) error
	}
)

type EventsBuffer struct {
	incompletes *wlru.Cache // event hash -> event
	callback    Callback
	mu          sync.Mutex

	limit consensus.Metric
}

func New(limit consensus.Metric, callback Callback) *EventsBuffer {
	buf := &EventsBuffer{
		callback: callback,
		limit:    limit,
	}
	buf.incompletes, _ = wlru.New(math.MaxInt32, math.MaxInt32)
	return buf
}

func (buf *EventsBuffer) PushEvent(de consensus.Event, peer string) (complete bool) {
	e := &event{
		event: de,
		peer:  peer,
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	if _, ok := buf.incompletes.Peek(e.event.ID()); ok {
		// duplicate
		buf.dropEvent(e, eventcheck.ErrDuplicateEvent)
		buf.releaseEvent(e)
		return false
	}
	complete = buf.pushEvent(e, nil, false)
	buf.spillIncompletes(buf.limit)
	return complete
}

func (buf *EventsBuffer) pushEvent(e *event, incompleteEventsList []*event, recheck bool) bool {
	if buf.callback.Exists(e.event.ID()) {
		buf.incompletes.Remove(e.event.ID())
		if !recheck {
			buf.dropEvent(e, eventcheck.ErrAlreadyConnectedEvent)
		}
		buf.releaseEvent(e)
		return false
	}
	parents := buf.completeEventParents(e)
	if parents == nil {
		if !recheck {
			buf.incompletes.Add(e.event.ID(), e, uint(e.event.Size()))
		}
		return false
	}

	ok := buf.processCompleteEvent(e, parents)
	buf.releaseEvent(e)

	if ok {
		// now child events may become complete, check it again
		eHash := e.event.ID()
		if incompleteEventsList == nil {
			incompleteEventsList = buf.getIncompleteEventsList()
		}
		for _, child := range incompleteEventsList {
			for _, parent := range child.event.Parents() {
				if parent == eHash {
					buf.pushEvent(child, incompleteEventsList, true)
					break
				}
			}
		}
	}
	buf.incompletes.Remove(e.event.ID())
	return ok
}

func (buf *EventsBuffer) getIncompleteEventsList() []*event {
	res := make([]*event, 0, buf.incompletes.Len())
	for _, childID := range buf.incompletes.Keys() {
		child, _ := buf.incompletes.Peek(childID)
		if child == nil {
			continue
		}
		res = append(res, child.(*event))
	}
	return res
}

func (buf *EventsBuffer) completeEventParents(e *event) consensus.Events {
	parents := make(consensus.Events, len(e.event.Parents()))
	for i, p := range e.event.Parents() {
		parent := buf.callback.Get(p)
		if parent == nil {
			return nil
		}
		parents[i] = parent
	}
	return parents
}

func (buf *EventsBuffer) processCompleteEvent(e *event, parents consensus.Events) bool {
	// validate
	if buf.callback.Check != nil {
		err := buf.callback.Check(e.event, parents)
		if err != nil {
			buf.dropEvent(e, err)
			return false
		}
	}

	// process
	err := buf.callback.Process(e.event)
	if err != nil {
		e.err = err
		buf.dropEvent(e, err)
		return false
	}
	return true
}

func (buf *EventsBuffer) spillIncompletes(limit consensus.Metric) {
	for consensus.Seq(buf.incompletes.Len()) > limit.Num || uint64(buf.incompletes.Weight()) > limit.Size {
		_, val, ok := buf.incompletes.RemoveOldest()
		if !ok {
			break
		}
		e := val.(*event)
		buf.dropEvent(e, eventcheck.ErrSpilledEvent)
		buf.releaseEvent(e)
	}
}

func (buf *EventsBuffer) dropEvent(e *event, err error) {
	if e.err == nil {
		e.err = err
	}
}

func (buf *EventsBuffer) releaseEvent(e *event) {
	if buf.callback.Released != nil && !e.released {
		buf.callback.Released(e.event, e.peer, e.err)
	}
	e.released = true
}

func (buf *EventsBuffer) IsBuffered(id consensus.EventHash) bool {
	// wlru is thread-safe, no need for a mutex here
	return buf.incompletes.Contains(id)
}

func (buf *EventsBuffer) Clear() {
	buf.mu.Lock()
	defer buf.mu.Unlock()
	buf.spillIncompletes(consensus.Metric{})
}

// Total returns the total weight and number of items in the cache.
func (buf *EventsBuffer) Total() consensus.Metric {
	// wlru is thread-safe, no need for a mutex here
	weight, num := buf.incompletes.Total()
	return consensus.Metric{
		Num:  consensus.Seq(num),
		Size: uint64(weight),
	}
}
