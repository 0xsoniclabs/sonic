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

/*
	In LRU cache data stored like pointer
*/

import (
	"bytes"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/inter"
)

// DelEvent deletes event.
func (s *Store) DelEvent(id consensus.EventHash) {
	key := id.Bytes()

	err := s.table.Events.Delete(key)
	if err != nil {
		s.Log.Crit("Failed to delete key", "err", err)
	}

	// Remove from LRU cache.
	s.cache.Events.Remove(id)
	s.cache.EventsHeaders.Remove(id)
	s.cache.EventIDs.Remove(id)
}

// SetEvent stores event.
func (s *Store) SetEvent(e *inter.EventPayload) {
	key := e.ID().Bytes()

	s.rlp.Set(s.table.Events, key, e)

	// Add to LRU cache.
	s.cache.Events.Add(e.ID(), e, uint(e.Size()))
	eh := e.Event
	s.cache.EventsHeaders.Add(e.ID(), &eh, nominalSize)
	s.cache.EventIDs.Add(e.ID())
}

// GetEventPayload returns stored event.
func (s *Store) GetEventPayload(id consensus.EventHash) *inter.EventPayload {
	// Get event from LRU cache first.
	if ev, ok := s.cache.Events.Get(id); ok {
		return ev.(*inter.EventPayload)
	}

	key := id.Bytes()
	w, _ := s.rlp.Get(s.table.Events, key, &inter.EventPayload{}).(*inter.EventPayload)

	// Put event to LRU cache.
	if w != nil {
		s.cache.Events.Add(id, w, uint(w.Size()))
		eh := w.Event
		s.cache.EventsHeaders.Add(id, &eh, nominalSize)
	}

	return w
}

// GetEvent returns stored event.
func (s *Store) GetEvent(id consensus.EventHash) *inter.Event {
	// Get event from LRU cache first.
	if ev, ok := s.cache.EventsHeaders.Get(id); ok {
		return ev.(*inter.Event)
	}

	key := id.Bytes()
	w, _ := s.rlp.Get(s.table.Events, key, &inter.EventPayload{}).(*inter.EventPayload)
	if w == nil {
		return nil
	}

	eh := w.Event

	// Put event to LRU cache.
	s.cache.Events.Add(id, w, uint(w.Size()))
	s.cache.EventsHeaders.Add(id, &eh, nominalSize)

	return &eh
}

func (s *Store) forEachEvent(it ethdb.Iterator, onEvent func(event *inter.EventPayload) bool) {
	for it.Next() {
		event := &inter.EventPayload{}
		err := rlp.DecodeBytes(it.Value(), event)
		if err != nil {
			s.Log.Crit("Failed to decode event", "err", err)
		}

		if !onEvent(event) {
			return
		}
	}
}

func (s *Store) ForEachEpochEvent(epoch consensus.Epoch, onEvent func(event *inter.EventPayload) bool) {
	it := s.table.Events.NewIterator(epoch.Bytes(), nil)
	defer it.Release()
	s.forEachEvent(it, onEvent)
}

func (s *Store) ForEachEvent(start consensus.Epoch, onEvent func(event *inter.EventPayload) bool) {
	it := s.table.Events.NewIterator(nil, start.Bytes())
	defer it.Release()
	s.forEachEvent(it, onEvent)
}

func (s *Store) ForEachEventRLP(start []byte, onEvent func(key consensus.EventHash, event rlp.RawValue) bool) {
	it := s.table.Events.NewIterator(nil, start)
	defer it.Release()
	for it.Next() {
		if !onEvent(consensus.BytesToEvent(it.Key()), it.Value()) {
			return
		}
	}
}

func (s *Store) FindEventHashes(epoch consensus.Epoch, lamport consensus.Lamport, hashPrefix []byte) consensus.EventHashes {
	prefix := bytes.NewBuffer(epoch.Bytes())
	prefix.Write(lamport.Bytes())
	prefix.Write(hashPrefix)
	res := make(consensus.EventHashes, 0, 10)

	it := s.table.Events.NewIterator(prefix.Bytes(), nil)
	defer it.Release()
	for it.Next() {
		res = append(res, consensus.BytesToEvent(it.Key()))
	}

	return res
}

// GetEventPayloadRLP returns stored event. Serialized.
func (s *Store) GetEventPayloadRLP(id consensus.EventHash) rlp.RawValue {
	key := id.Bytes()

	data, err := s.table.Events.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	return data
}

// HasEvent returns true if event exists.
func (s *Store) HasEvent(h consensus.EventHash) bool {
	if has, ok := s.cache.EventIDs.Has(h); ok {
		return has
	}
	has, _ := s.table.Events.Has(h.Bytes())
	return has
}

func (s *Store) loadHighestLamport() consensus.Lamport {
	lamportBytes, err := s.table.HighestLamport.Get([]byte("k"))
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if lamportBytes == nil {
		return 0
	}
	return consensus.BytesToLamport(lamportBytes)
}

func (s *Store) getCachedHighestLamport() (consensus.Lamport, bool) {
	cache := s.cache.HighestLamport.Load()
	if cache != nil {
		return cache.(consensus.Lamport), true
	}
	return 0, false
}

func (s *Store) GetHighestLamport() consensus.Lamport {
	cached, ok := s.getCachedHighestLamport()
	if ok {
		return cached
	}
	lamport := s.loadHighestLamport()
	s.cache.HighestLamport.Store(lamport)
	return lamport
}

func (s *Store) SetHighestLamport(lamport consensus.Lamport) {
	s.cache.HighestLamport.Store(lamport)
}

func (s *Store) FlushHighestLamport() {
	cached, ok := s.getCachedHighestLamport()
	if !ok {
		return
	}
	err := s.table.HighestLamport.Put([]byte("k"), cached.Bytes())
	if err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}
