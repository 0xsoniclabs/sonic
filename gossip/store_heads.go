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
	"bytes"
	"sort"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/0xsoniclabs/sonic/utils/concurrent"
)

type sortedHead []byte

func (es *epochStore) getCachedHeads() (*concurrent.EventsSet, bool) {
	cache := es.cache.Heads.Load()
	if cache != nil {
		return cache.(*concurrent.EventsSet), true
	}
	return nil, false
}

func (es *epochStore) loadHeads() *concurrent.EventsSet {
	res := make(hash.EventsSet, 100)

	b, err := es.table.Heads.Get([]byte{})
	if err != nil {
		es.Log.Crit("Failed to get key-value", "err", err)
	}
	if b == nil {
		return concurrent.WrapEventsSet(res)
	}
	for i := 0; i < len(b); i += 32 {
		res.Add(hash.BytesToEvent(b[i : i+32]))
	}

	return concurrent.WrapEventsSet(res)
}

func (es *epochStore) GetHeads() *concurrent.EventsSet {
	cached, ok := es.getCachedHeads()
	if ok {
		return cached
	}
	heads := es.loadHeads()
	if heads == nil {
		heads = &concurrent.EventsSet{}
	}
	es.cache.Heads.Store(heads)
	return heads
}

func (es *epochStore) SetHeads(ids *concurrent.EventsSet) {
	es.cache.Heads.Store(ids)
}

func (es *epochStore) FlushHeads() {
	ids, ok := es.getCachedHeads()
	if !ok {
		return
	}

	// sort values for determinism
	sortedHeads := make([]sortedHead, 0, len(ids.Val))
	for id := range ids.Val {
		sortedHeads = append(sortedHeads, id.Bytes())
	}
	sort.Slice(sortedHeads, func(i, j int) bool {
		a, b := sortedHeads[i], sortedHeads[j]
		return bytes.Compare(a, b) < 0
	})

	b := make([]byte, 0, len(sortedHeads)*32)
	for _, head := range sortedHeads {
		b = append(b, head...)
	}

	if err := es.table.Heads.Put([]byte{}, b); err != nil {
		es.Log.Crit("Failed to put key-value", "err", err)
	}
}

// GetHeadsSlice returns IDs of all the epoch events with no descendants
func (s *Store) GetHeadsSlice(epoch idx.Epoch) hash.Events {
	heads := s.GetHeads(epoch)
	heads.RLock()
	defer heads.RUnlock()
	return heads.Val.Slice()
}

// GetHeads returns set of all the epoch event IDs with no descendants
func (s *Store) GetHeads(epoch idx.Epoch) *concurrent.EventsSet {
	es := s.getEpochStore(epoch)
	if es == nil {
		return nil
	}

	return es.GetHeads()
}

func (s *Store) SetHeads(epoch idx.Epoch, ids *concurrent.EventsSet) {
	es := s.getEpochStore(epoch)
	if es == nil {
		return
	}

	es.SetHeads(ids)
}
