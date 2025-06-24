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

package originatedtxs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/golang-lru/simplelru"
)

type Buffer struct {
	senderCount *simplelru.LRU // sender address -> number of transactions
}

func New(maxAddresses int) *Buffer {
	ring := &Buffer{}
	ring.senderCount, _ = simplelru.NewLRU(maxAddresses, nil)
	return ring
}

// Inc is not safe for concurrent use
func (ring *Buffer) Inc(sender common.Address) {
	cur, ok := ring.senderCount.Peek(sender)
	if ok {
		ring.senderCount.Add(sender, cur.(int)+1)
	} else {
		ring.senderCount.Add(sender, int(1))
	}
}

// Dec is not safe for concurrent use
func (ring *Buffer) Dec(sender common.Address) {
	cur, ok := ring.senderCount.Peek(sender)
	if !ok {
		return
	}
	if cur.(int) <= 1 {
		ring.senderCount.Remove(sender)
	} else {
		ring.senderCount.Add(sender, cur.(int)-1)
	}
}

// Clear is not safe for concurrent use
func (ring *Buffer) Clear() {
	ring.senderCount.Purge()
}

// TotalOf is not safe for concurrent use
func (ring *Buffer) TotalOf(sender common.Address) int {
	cur, ok := ring.senderCount.Get(sender)
	if !ok {
		return 0
	}
	return cur.(int)
}

// Empty is not safe for concurrent use
func (ring *Buffer) Empty() bool {
	return ring.senderCount.Len() == 0
}
