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

package tests

import (
	"sync"

	"github.com/0xsoniclabs/sonic/rpc/core"
)

// feed sends events to its subscribers as the node's feeds do: a send waits
// until every subscriber has taken the event or unsubscribed, and a stopped
// feed answers a subscription that has ended.
type feed[Event any] struct {
	mutex       sync.Mutex
	stopped     bool
	subscribers map[*subscription[Event]]struct{}
}

type subscription[Event any] struct {
	events chan<- Event
	done   chan struct{}
	errors chan error
	once   sync.Once
}

func (s *subscription[Event]) Err() <-chan error { return s.errors }

func (s *subscription[Event]) Unsubscribe() {
	s.once.Do(func() {
		close(s.done)
		close(s.errors)
	})
}

func (f *feed[Event]) subscribe(events chan<- Event) core.Subscription {
	added := &subscription[Event]{events: events, done: make(chan struct{}), errors: make(chan error)}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.stopped {
		added.Unsubscribe()
		return added
	}
	if f.subscribers == nil {
		f.subscribers = map[*subscription[Event]]struct{}{}
	}
	f.subscribers[added] = struct{}{}
	return added
}

func (f *feed[Event]) send(event Event) {
	f.mutex.Lock()
	var receivers []*subscription[Event]
	for receiver := range f.subscribers {
		receivers = append(receivers, receiver)
	}
	f.mutex.Unlock()
	for _, receiver := range receivers {
		select {
		case receiver.events <- event:
		case <-receiver.done:
		}
	}
}

func (f *feed[Event]) subscribed() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.subscribers)
}

func (f *feed[Event]) stop() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.stopped = true
	for receiver := range f.subscribers {
		receiver.Unsubscribe()
	}
	f.subscribers = nil
}
