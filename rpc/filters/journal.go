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

// journal keeps the events of one kind once for all polled filters of that
// kind. An event has a position; a filter keeps the position up to which its
// client polled. The journal holds at most limit events, and no event every
// live filter already polled.
type journal[T any] struct {
	limit int

	mu     sync.Mutex
	base   uint64 // position of events[0]
	events []T
}

func newJournal[T any](limit int) *journal[T] {
	return &journal[T]{limit: limit}
}

// end is the position the next event gets: a new filter starts there.
func (j *journal[T]) end() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.base + uint64(len(j.events))
}

// add appends events, dropping the oldest beyond the limit.
func (j *journal[T]) add(events ...T) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.events = append(j.events, events...)
	if over := len(j.events) - j.limit; over > 0 {
		clear(j.events[:over])
		j.events = j.events[over:]
		j.base += uint64(over)
	}
}

// since returns the events from a position on and the position after them.
// It reports false if events after the position were dropped already: the
// filter would see a gap.
func (j *journal[T]) since(pos uint64) ([]T, uint64, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if pos < j.base {
		return nil, 0, false
	}
	end := j.base + uint64(len(j.events))
	if pos >= end {
		return nil, end, true
	}
	return append([]T(nil), j.events[pos-j.base:]...), end, true
}

// trim drops the events before a position, the oldest one a live filter has
// not polled.
func (j *journal[T]) trim(pos uint64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if pos <= j.base {
		return
	}
	n := min(pos-j.base, uint64(len(j.events)))
	clear(j.events[:n])
	j.events = j.events[n:]
	j.base += n
	if len(j.events) == 0 {
		j.events = nil
	}
}

// len is the number of events held.
func (j *journal[T]) len() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.events)
}
