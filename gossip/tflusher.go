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

package gossip

import (
	"sync"
	"time"
)

type PeriodicFlusherCallaback struct {
	busy         func() bool
	commitNeeded func() bool
	commit       func()
}

// PeriodicFlusher periodically commits the Store if isCommitNeeded returns true
type PeriodicFlusher struct {
	period   time.Duration
	callback PeriodicFlusherCallaback

	wg   sync.WaitGroup
	quit chan struct{}
}

func (c *PeriodicFlusher) loop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !c.callback.busy() && c.callback.commitNeeded() {
				c.callback.commit()
			}
		case <-c.quit:
			return
		}
	}
}

func (c *PeriodicFlusher) Start() {
	c.wg.Add(1)
	go c.loop()
}

func (c *PeriodicFlusher) Stop() {
	close(c.quit)
	c.wg.Wait()
}
