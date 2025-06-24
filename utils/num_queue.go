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

package utils

import (
	"sync"
)

type NumQueue struct {
	mu       sync.Mutex
	lastDone uint64
	waiters  []chan struct{}
}

func NewNumQueue(init uint64) *NumQueue {
	return &NumQueue{
		lastDone: init,
	}
}

func (q *NumQueue) Done(n uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if n <= q.lastDone {
		panic("Already done!")
	}

	pos := int(n - q.lastDone - 1)
	for i := 0; i < len(q.waiters) && i <= pos; i++ {
		close(q.waiters[i])
	}
	if pos < len(q.waiters) {
		q.waiters = q.waiters[pos+1:]
	} else {
		q.waiters = make([]chan struct{}, 0, 1000)
	}

	q.lastDone = n
}

func (q *NumQueue) WaitFor(n uint64) {
	q.mu.Lock()

	if n <= q.lastDone {
		q.mu.Unlock()
		return
	}

	count := int(n - q.lastDone)
	for i := len(q.waiters); i < count; i++ {
		q.waiters = append(q.waiters, make(chan struct{}))
	}
	ch := q.waiters[count-1]
	q.mu.Unlock()
	<-ch
}
