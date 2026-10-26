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

package chain

import (
	"errors"
	"sync"
)

// ErrFellBehind is the reason a consumer was dropped because its queue was
// full.
var ErrFellBehind = errors.New("consumer fell behind the feed")

// ErrFeedRestarted is the reason every consumer of a feed is dropped when its
// loop restarts: the events between the failure and the new subscription
// were never published.
var ErrFeedRestarted = errors.New("feed restarted, events were lost")

// ErrUnsubscribed is the reason of a consumer that left by itself.
var ErrUnsubscribed = errors.New("unsubscribed")

// Hub fans the events of one feed out to bounded queues, one per consumer.
// Publish never waits: a consumer whose queue is full is dropped and told
// why, so it never goes on with a gap.
type Hub[Event any] struct {
	capacity int

	mutex     sync.Mutex
	consumers map[*Consumer[Event]]struct{}
}

// NewHub returns a hub whose consumers queue up to capacity events.
func NewHub[Event any](capacity int) *Hub[Event] {
	return &Hub[Event]{capacity: capacity, consumers: map[*Consumer[Event]]struct{}{}}
}

// Consumer receives the events of a hub in order. Its channel closes when it
// is dropped or leaves; Err then tells why.
type Consumer[Event any] struct {
	hub    *Hub[Event]
	events chan Event
	reason error
}

// Subscribe adds a consumer that receives every event published from now on.
func (hub *Hub[Event]) Subscribe() *Consumer[Event] {
	consumer := &Consumer[Event]{hub: hub, events: make(chan Event, hub.capacity)}
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	hub.consumers[consumer] = struct{}{}
	return consumer
}

// Publish queues an event for every consumer and drops each one whose queue
// is full.
func (hub *Hub[Event]) Publish(event Event) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	for consumer := range hub.consumers {
		select {
		case consumer.events <- event:
		default:
			hub.drop(consumer, ErrFellBehind)
		}
	}
}

// DropAll drops every consumer with the given reason.
func (hub *Hub[Event]) DropAll(reason error) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	for consumer := range hub.consumers {
		hub.drop(consumer, reason)
	}
}

// Len is the number of consumers.
func (hub *Hub[Event]) Len() int {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	return len(hub.consumers)
}

// drop removes a consumer and closes its queue; the caller holds the lock.
func (hub *Hub[Event]) drop(consumer *Consumer[Event], reason error) {
	if _, found := hub.consumers[consumer]; !found {
		return
	}
	delete(hub.consumers, consumer)
	consumer.reason = reason
	close(consumer.events)
}

// Events is the queue of the consumer. It closes when the consumer is
// dropped or leaves.
func (consumer *Consumer[Event]) Events() <-chan Event {
	return consumer.events
}

// Err tells why the queue closed. It is nil while the consumer is live.
func (consumer *Consumer[Event]) Err() error {
	consumer.hub.mutex.Lock()
	defer consumer.hub.mutex.Unlock()
	return consumer.reason
}

// Close removes the consumer from its hub.
func (consumer *Consumer[Event]) Close() {
	consumer.hub.mutex.Lock()
	defer consumer.hub.mutex.Unlock()
	consumer.hub.drop(consumer, ErrUnsubscribed)
}
