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

package ethdb2kvdb

import (
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Adapter struct {
	ethdb.KeyValueStore
}

var _ kvdb.Store = (*Adapter)(nil)

func Wrap(v ethdb.KeyValueStore) *Adapter {
	return &Adapter{v}
}

func (db *Adapter) Drop() {
	panic("called Drop on ethdb")
}

func (db *Adapter) AncientDatadir() (string, error) {
	panic("called AncientDatadir on ethdb")
}

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	ethdb.Batch
}

// Replay replays the batch contents.
func (b *batch) Replay(w kvdb.Writer) error {
	return b.Batch.Replay(w)
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *Adapter) NewBatch() kvdb.Batch {
	return &batch{db.KeyValueStore.NewBatch()}
}

func (db *Adapter) GetSnapshot() (kvdb.Snapshot, error) {
	panic("called GetSnapshot on ethdb")
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (db *Adapter) NewIterator(prefix []byte, start []byte) kvdb.Iterator {
	return db.KeyValueStore.NewIterator(prefix, start)
}
