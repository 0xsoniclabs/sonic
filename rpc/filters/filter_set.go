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

package filters

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// maxFilters bounds the polled filters of the node.
	maxFilters = 100_000
	// journalLimit bounds the events a journal keeps.
	journalLimit = 1 << 20
)

var errFilterNotFound = errors.New("filter not found")

// kind is what a polled filter receives.
type kind int

const (
	blocksKind kind = iota
	logsKind
	transactionsKind
	kinds
)

// filter is a polled filter: what it selects, where in the journal of its
// kind it polls next and when it expires.
type filter struct {
	kind     kind
	next     uint64
	deadline time.Time
	// criteria select the logs of a logs filter; mined tells whether it
	// receives any.
	criteria criteria
	mined    bool
	// full serves whole transactions rather than their hashes.
	full bool
}

// journal holds the events of a kind in order, from the oldest one a live
// filter of the kind has not polled; each has a sequence number.
type journal[Event any] struct {
	first  uint64
	events []Event
}

// sequence is what the filter set asks of a journal of any kind.
type sequence interface {
	start() uint64
	end() uint64
	trim(oldest uint64)
}

// start is the sequence number of the oldest event.
func (j *journal[Event]) start() uint64 {
	return j.first
}

// end is the sequence number of the next event.
func (j *journal[Event]) end() uint64 {
	return j.first + uint64(len(j.events))
}

// since is the events from a sequence number on. A poller takes them outside
// the lock: the slice is never written, only appended to beyond its length.
func (j *journal[Event]) since(next uint64) []Event {
	return j.events[next-j.first:]
}

// trim drops the events before a sequence number.
func (j *journal[Event]) trim(oldest uint64) {
	if oldest > j.first {
		j.events = j.events[oldest-j.first:]
		j.first = oldest
	}
}

// filterSet holds the polled filters of the node and a journal per kind.
// Its methods lock and never call out.
type filterSet struct {
	timeout time.Duration
	now     func() time.Time

	mutex        sync.Mutex
	filters      map[dispatch.ID]*filter
	live         [kinds]int
	blocks       journal[common.Hash]
	logs         journal[*types.Log]
	transactions journal[*types.Transaction]
}

func newFilterSet(timeout time.Duration) *filterSet {
	return &filterSet{timeout: timeout, now: time.Now, filters: map[dispatch.ID]*filter{}}
}

// install adds a filter that receives the events from now on.
func (s *filterSet) install(added *filter) (dispatch.ID, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.filters) >= maxFilters {
		return "", fmt.Errorf("too many filters, the limit is %d", maxFilters)
	}
	added.next = s.journal(added.kind).end()
	added.deadline = s.now().Add(s.timeout)
	id := dispatch.NewID()
	s.filters[id] = added
	s.live[added.kind]++
	return id, nil
}

// poll answers a copy of a filter as it was and the events it has not
// polled, and moves it past them.
func (s *filterSet) poll(id dispatch.ID) (filter, polled, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	found, exists := s.filters[id]
	if !exists {
		return filter{}, polled{}, errFilterNotFound
	}
	found.deadline = s.now().Add(s.timeout)
	var events polled
	switch found.kind {
	case blocksKind:
		events.blocks = s.blocks.since(found.next)
	case logsKind:
		events.logs = s.logs.since(found.next)
	case transactionsKind:
		events.transactions = s.transactions.since(found.next)
	}
	was := *found
	journal := s.journal(found.kind)
	found.next = journal.end()
	if was.next == journal.start() {
		s.trimLocked(found.kind)
	}
	return was, events, nil
}

// polled are the events of one poll; only those of the filter's kind.
type polled struct {
	blocks       []common.Hash
	logs         []*types.Log
	transactions []*types.Transaction
}

// get answers a copy of a filter.
func (s *filterSet) get(id dispatch.ID) (filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	found, exists := s.filters[id]
	if !exists {
		return filter{}, errFilterNotFound
	}
	return *found, nil
}

// uninstall removes a filter and tells whether there was one.
func (s *filterSet) uninstall(id dispatch.ID) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	found, exists := s.filters[id]
	if exists {
		s.removeLocked(id, found)
		s.trimLocked(found.kind)
	}
	return exists
}

// expire removes the filters whose deadline has passed.
func (s *filterSet) expire() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := s.now()
	for id, found := range s.filters {
		if now.After(found.deadline) {
			s.removeLocked(id, found)
		}
	}
	for kind := range kinds {
		s.trimLocked(kind)
	}
}

// addBlock journals a block's hash and logs for the filters of their kinds.
func (s *filterSet) addBlock(hash common.Hash, logs []*types.Log) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.blocks.events = append(s.blocks.events, hash)
	s.logs.events = append(s.logs.events, logs...)
	s.boundLocked(blocksKind)
	s.boundLocked(logsKind)
}

// addTransactions journals transactions that entered the pool.
func (s *filterSet) addTransactions(transactions []*types.Transaction) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.transactions.events = append(s.transactions.events, transactions...)
	s.boundLocked(transactionsKind)
}

// drop removes every filter of the kinds and empties their journals, after
// their feed lost events.
func (s *filterSet) drop(dropped ...kind) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for id, found := range s.filters {
		for _, kind := range dropped {
			if found.kind == kind {
				s.removeLocked(id, found)
			}
		}
	}
	for _, kind := range dropped {
		s.trimLocked(kind)
	}
}

func (s *filterSet) removeLocked(id dispatch.ID, found *filter) {
	delete(s.filters, id)
	s.live[found.kind]--
}

// boundLocked keeps a journal empty without filters, and within its limit
// with them, removing the filters whose events it drops. The filters trim
// the journal as they poll.
func (s *filterSet) boundLocked(kind kind) {
	journal := s.journal(kind)
	if s.live[kind] == 0 {
		journal.trim(journal.end())
		return
	}
	if journal.end()-journal.start() <= journalLimit {
		return
	}
	oldest := journal.end() - journalLimit
	for id, found := range s.filters {
		if found.kind == kind && found.next < oldest {
			s.removeLocked(id, found)
		}
	}
	s.trimLocked(kind)
}

// trimLocked drops the events every live filter of the kind has polled; all
// of them without one.
func (s *filterSet) trimLocked(kind kind) {
	journal := s.journal(kind)
	oldest := journal.end()
	if s.live[kind] > 0 {
		for _, found := range s.filters {
			if found.kind == kind {
				oldest = min(oldest, found.next)
			}
		}
	}
	journal.trim(oldest)
}

func (s *filterSet) journal(kind kind) sequence {
	switch kind {
	case blocksKind:
		return &s.blocks
	case logsKind:
		return &s.logs
	}
	return &s.transactions
}
