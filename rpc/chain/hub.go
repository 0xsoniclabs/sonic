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

// The reasons a consumer of a hub ends.
var (
	// ErrFellBehind ends a consumer whose queue was full when an event came.
	ErrFellBehind = errors.New("the consumer fell behind the events")
	// ErrFeedLost ends the consumers of a hub whose feed was lost, so that
	// none of them goes on with a gap.
	ErrFeedLost = errors.New("the feed of the events was lost")
)

// Hub hands events to its consumers through bounded queues, so that what
// publishes never waits for a consumer. A consumer whose queue is full when
// an event comes is dropped: it ends and learns why, so that it never goes on
// with a gap.
type Hub[Event any] struct {
	mutex     sync.Mutex
	consumers map[*Consumer[Event]]struct{}
}

// NewHub creates a hub without consumers.
func NewHub[Event any]() *Hub[Event] {
	return &Hub[Event]{consumers: map[*Consumer[Event]]struct{}{}}
}

// Consumer takes the events a hub publishes after it subscribed, in order,
// until it ends.
type Consumer[Event any] struct {
	hub    *Hub[Event]
	events chan Event
	ended  chan struct{}
	err    error
}

// Subscribe adds a consumer with a queue of buffer events.
func (h *Hub[Event]) Subscribe(buffer int) *Consumer[Event] {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	consumer := &Consumer[Event]{hub: h, events: make(chan Event, buffer), ended: make(chan struct{})}
	h.consumers[consumer] = struct{}{}
	return consumer
}

// Consumers is how many consumers the hub has.
func (h *Hub[Event]) Consumers() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return len(h.consumers)
}

// Publish hands an event to every consumer and drops those whose queue is
// full.
func (h *Hub[Event]) Publish(event Event) {
	h.publish(event, func() {})
}

// publish hands an event to every consumer, then runs after, under the lock
// that Subscribe takes, so that a consumer subscribes either before the
// event or after what after changes.
func (h *Hub[Event]) publish(event Event, after func()) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for consumer := range h.consumers {
		select {
		case consumer.events <- event:
		default:
			h.endLocked(consumer, ErrFellBehind)
		}
	}
	after()
}

// DropAll ends every consumer with the reason.
func (h *Hub[Event]) DropAll(err error) {
	h.dropAll(err, func() {})
}

// dropAll ends every consumer with the reason, then runs after under the
// lock that Subscribe takes.
func (h *Hub[Event]) dropAll(err error, after func()) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for consumer := range h.consumers {
		h.endLocked(consumer, err)
	}
	after()
}

// endLocked ends a consumer; the caller holds the lock.
func (h *Hub[Event]) endLocked(consumer *Consumer[Event], err error) {
	if _, live := h.consumers[consumer]; !live {
		return
	}
	delete(h.consumers, consumer)
	consumer.err = err
	close(consumer.ended)
}

// Events delivers the consumer's events. Once Ended is closed, what it
// still holds may be followed by a gap and is not to be taken.
func (c *Consumer[Event]) Events() <-chan Event {
	return c.events
}

// Ended is closed when the consumer ends.
func (c *Consumer[Event]) Ended() <-chan struct{} {
	return c.ended
}

// Err is why the consumer ended, nil while it has not, or when it ended by
// Unsubscribe.
func (c *Consumer[Event]) Err() error {
	c.hub.mutex.Lock()
	defer c.hub.mutex.Unlock()
	return c.err
}

// Unsubscribe ends the consumer.
func (c *Consumer[Event]) Unsubscribe() {
	c.hub.mutex.Lock()
	defer c.hub.mutex.Unlock()
	c.hub.endLocked(c, nil)
}
