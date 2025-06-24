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

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type Cache struct {
	ids     map[hash.Event]bool
	mu      sync.RWMutex
	maxSize int
	epoch   idx.Epoch
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
	}
}

func (c *Cache) Reset(epoch idx.Epoch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = make(map[hash.Event]bool)
	c.epoch = epoch
}

func (c *Cache) Has(id hash.Event) (has bool, ok bool) {
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

func (c *Cache) Add(id hash.Event) bool {
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

func (c *Cache) Remove(id hash.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ids == nil {
		return
	}
	delete(c.ids, id)
}
