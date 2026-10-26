// Copyright 2026 Sonic Operations Ltd
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

package filters

import "sync"

// journal keeps the events of one kind once for all the polled filters of
// the kind: a filter keeps only the position up to which its client polled
// (I17). It keeps at most limit events, dropping the oldest.
type journal[Event any] struct {
	mutex  sync.Mutex
	events []Event
	first  uint64
	limit  int
}

func newJournal[Event any](limit int) *journal[Event] {
	return &journal[Event]{limit: limit}
}

// end is the position after the last event.
func (events *journal[Event]) end() uint64 {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	return events.first + uint64(len(events.events))
}

// append adds events, dropping the oldest above the limit.
func (events *journal[Event]) append(added ...Event) {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	events.events = append(events.events, added...)
	if excess := len(events.events) - events.limit; excess > 0 {
		events.drop(excess)
	}
}

// since returns the events from a position on and the end; lost tells that
// events after the position were dropped.
func (events *journal[Event]) since(position uint64) (found []Event, end uint64, lost bool) {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	end = events.first + uint64(len(events.events))
	if position < events.first {
		return nil, end, true
	}
	return append([]Event(nil), events.events[position-events.first:]...), end, false
}

// trim drops the events before a position.
func (events *journal[Event]) trim(position uint64) {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	if position > events.first {
		events.drop(int(min(position-events.first, uint64(len(events.events)))))
	}
}

func (events *journal[Event]) drop(count int) {
	clear(events.events[:count])
	events.events = events.events[count:]
	events.first += uint64(count)
}
