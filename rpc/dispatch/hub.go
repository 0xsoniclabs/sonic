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

package dispatch

import (
	"sync"
	"sync/atomic"
)

// Hub hands each value it publishes to every consumer through a bounded queue.
// Publishing never waits for a consumer: one that falls behind is dropped, so a
// slow consumer cannot hold back the feed it follows (I15).
type Hub[T any] struct {
	mu        sync.Mutex
	consumers map[*Consumer[T]]struct{}
}

// NewHub creates a hub without consumers.
func NewHub[T any]() *Hub[T] {
	return &Hub[T]{consumers: map[*Consumer[T]]struct{}{}}
}

// Consumer receives the values of a hub. Its queue is closed when the consumer
// is dropped or closed.
type Consumer[T any] struct {
	ch      chan T
	hub     *Hub[T]
	dropped atomic.Bool
	once    sync.Once
}

// Subscribe registers a consumer with a queue of the given size.
func (h *Hub[T]) Subscribe(size int) *Consumer[T] {
	c := &Consumer[T]{ch: make(chan T, max(size, 1)), hub: h}
	h.mu.Lock()
	h.consumers[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Publish offers the value to every consumer and drops those whose queue is
// full.
func (h *Hub[T]) Publish(v T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.consumers {
		select {
		case c.ch <- v:
		default:
			c.drop()
			delete(h.consumers, c)
		}
	}
}

// Len returns the number of consumers.
func (h *Hub[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.consumers)
}

// Events is the queue of the consumer.
func (c *Consumer[T]) Events() <-chan T { return c.ch }

// Dropped reports whether the consumer was dropped because it fell behind.
func (c *Consumer[T]) Dropped() bool { return c.dropped.Load() }

func (c *Consumer[T]) drop() {
	c.dropped.Store(true)
	c.once.Do(func() { close(c.ch) })
}

// Close unregisters the consumer.
func (c *Consumer[T]) Close() {
	c.hub.mu.Lock()
	delete(c.hub.consumers, c)
	c.hub.mu.Unlock()
	c.once.Do(func() { close(c.ch) })
}
