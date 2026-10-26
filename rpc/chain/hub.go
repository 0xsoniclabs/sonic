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
	"sync"
)

// Hub fans one feed out to its consumers through bounded queues. Publishing
// never waits for a consumer: one whose queue is full is dropped and told so,
// and so are all of them when the feed restarts. A consumer never goes on
// with a gap in what it received.
type Hub[T any] struct {
	size      int
	mu        sync.Mutex
	consumers map[*Consumer[T]]struct{}
}

// NewHub creates a hub whose consumers queue up to size events.
func NewHub[T any](size int) *Hub[T] {
	return &Hub[T]{size: size, consumers: make(map[*Consumer[T]]struct{})}
}

// Consumer receives the events of a hub until it is closed or dropped.
type Consumer[T any] struct {
	hub      *Hub[T]
	events   chan T
	dropped  chan struct{}
	dropOnce sync.Once
}

// Subscribe adds a consumer; it receives every event published after it.
func (h *Hub[T]) Subscribe() *Consumer[T] {
	c := &Consumer[T]{hub: h, events: make(chan T, h.size), dropped: make(chan struct{})}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.consumers[c] = struct{}{}
	return c
}

// Events delivers the events in the order they were published.
func (c *Consumer[T]) Events() <-chan T {
	return c.events
}

// Dropped is closed when the hub dropped the consumer: it fell behind, or the
// feed restarted. Events still queued are not to be used after it.
func (c *Consumer[T]) Dropped() <-chan struct{} {
	return c.dropped
}

// Close removes the consumer from its hub.
func (c *Consumer[T]) Close() {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	delete(c.hub.consumers, c)
}

func (c *Consumer[T]) drop() {
	c.dropOnce.Do(func() { close(c.dropped) })
}

// Publish queues an event for every consumer without waiting; a consumer
// whose queue is full is dropped.
func (h *Hub[T]) Publish(event T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.consumers {
		select {
		case c.events <- event:
		default:
			delete(h.consumers, c)
			c.drop()
		}
	}
}

// DropAll drops every consumer: the feed lost events they will never see.
func (h *Hub[T]) DropAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.consumers {
		delete(h.consumers, c)
		c.drop()
	}
}

// Len is the number of consumers.
func (h *Hub[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.consumers)
}
