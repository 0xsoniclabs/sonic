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

// journal keeps the events of one kind for the polled filters, once for all of
// them. A position counts every event ever appended; a filter remembers the
// position up to which its client has the events. The journal is not safe for
// concurrent use; the API guards it with its lock.
type journal[T any] struct {
	first uint64 // position of items[0]
	items []T
	limit int // the number of events kept at most
}

func newJournal[T any](limit int) journal[T] { return journal[T]{limit: max(limit, 1)} }

// end returns the position after the newest event.
func (j *journal[T]) end() uint64 { return j.first + uint64(len(j.items)) }

// append adds events. Beyond the limit the oldest quarter of the events is
// dropped at once, so the copy that drops them is paid once per many events.
func (j *journal[T]) append(events ...T) {
	j.items = append(j.items, events...)
	if len(j.items) > j.limit {
		j.trim(j.end() - uint64(j.limit-j.limit/4))
	}
}

// since returns the events from the position on, and false if some of them
// were dropped. The events of the result are never written again: the caller
// may read them after it released the lock.
func (j *journal[T]) since(pos uint64) ([]T, bool) {
	if pos < j.first {
		return nil, false
	}
	if pos >= j.end() {
		return nil, true
	}
	view := j.items[pos-j.first:]
	return view[:len(view):len(view)], true
}

// trim drops the events before the position. The rest is copied: a slice the
// caller of since still reads stays as it was.
func (j *journal[T]) trim(pos uint64) {
	if pos <= j.first {
		return
	}
	if pos >= j.end() {
		j.first, j.items = j.end(), nil
		return
	}
	j.items = append([]T(nil), j.items[pos-j.first:]...)
	j.first = pos
}
