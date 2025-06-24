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

package kvdb2ethdb

import (
	"bytes"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Adapter struct {
	kvdb.Store
}

var _ ethdb.KeyValueStore = (*Adapter)(nil)

func Wrap(v kvdb.Store) *Adapter {
	return &Adapter{v}
}

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	kvdb.Batch
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	return b.Batch.Replay(w)
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *Adapter) NewBatch() ethdb.Batch {
	return &batch{db.Store.NewBatch()}
}

func (db *Adapter) NewBatchWithSize(int) ethdb.Batch {
	return db.NewBatch()
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (db *Adapter) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return db.Store.NewIterator(prefix, start)
}

// DeleteRange deletes all of the keys (and values) in the range [start,end).
func (db *Adapter) DeleteRange(start, end []byte) error {
	iter := db.Store.NewIterator(nil, start)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		if bytes.Compare(key, end) >= 0 {
			break
		}
		if err := db.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
