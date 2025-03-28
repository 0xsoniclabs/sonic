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
