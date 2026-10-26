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

package transport

import (
	"context"
	"sync"
)

// connections are the stream connections of a server: Stop ends them and
// waits for them, as net/http does not for the connections it handed over.
type connections struct {
	ctx    context.Context
	cancel context.CancelFunc

	mutex   sync.Mutex
	stopped bool
	running sync.WaitGroup
}

func newConnections() *connections {
	ctx, cancel := context.WithCancel(context.Background())
	return &connections{ctx: ctx, cancel: cancel}
}

// add counts a new connection, whose context ends at Stop; done uncounts
// it. Once Stop began, no connection is added.
func (served *connections) add() (ctx context.Context, done func(), open bool) {
	served.mutex.Lock()
	defer served.mutex.Unlock()
	if served.stopped {
		return nil, nil, false
	}
	served.running.Add(1)
	return served.ctx, served.running.Done, true
}

// stop ends every connection and waits for them.
func (served *connections) stop() {
	served.mutex.Lock()
	served.stopped = true
	served.mutex.Unlock()
	served.cancel()
	served.running.Wait()
}
