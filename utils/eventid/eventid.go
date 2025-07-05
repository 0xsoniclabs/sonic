// Copyright 2025 Sonic Operations Ltd
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

package eventid

import (
	"sync"

	"github.com/0xsoniclabs/consensus/consensus"
)

type Cache struct {
	ids     map[consensus.EventHash]bool
	mu      sync.RWMutex
	maxSize int
	epoch   consensus.Epoch
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
	}
}

func (c *Cache) Reset(epoch consensus.Epoch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = make(map[consensus.EventHash]bool)
	c.epoch = epoch
}

func (c *Cache) Has(id consensus.EventHash) (has bool, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ids == nil {
		return false, false
	}
	if c.epoch != id.Epoch() {
		return false, false
	}
	return c.ids[id], true
}

func (c *Cache) Add(id consensus.EventHash) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ids == nil {
		return false
	}
	if c.epoch != id.Epoch() {
		return false
	}
	if len(c.ids) >= c.maxSize {
		c.ids = nil
		return false
	}
	c.ids[id] = true
	return true
}

func (c *Cache) Remove(id consensus.EventHash) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ids == nil {
		return
	}
	delete(c.ids, id)
}
