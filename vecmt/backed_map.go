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

package vecmt

import (
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
)

type backedMap struct {
	cache      map[string][]byte
	backup     kvdb.Store
	memSize    int
	maxMemSize int
	batchSize  int
}

func newBackedMap(backup kvdb.Store, maxMemSize, batchSize int) *backedMap {
	return &backedMap{
		cache:      make(map[string][]byte),
		backup:     backup,
		maxMemSize: maxMemSize,
		batchSize:  batchSize,
	}
}

func (w *backedMap) has(key []byte) (bool, error) {
	if _, ok := w.cache[string(key)]; ok {
		return true, nil
	}
	val, err := w.backup.Get(key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

func (w *backedMap) get(key []byte) ([]byte, error) {
	if val, ok := w.cache[string(key)]; ok {
		return common.CopyBytes(val), nil
	}
	return w.backup.Get(key)
}

func (w *backedMap) close() error {
	batch := w.backup.NewBatch()
	defer batch.Reset()

	for key, val := range w.cache {
		err := batch.Put([]byte(key), val)
		if err != nil {
			return err
		}
	}
	err := batch.Write()
	if err != nil {
		return err
	}

	w.cache = nil
	// backing kvdb.Store not closed here intentionally (is a table)
	return nil
}

func (w *backedMap) add(key string, val []byte) {
	lenBefore := len(w.cache)
	w.cache[key] = val
	// TODO it works correctly only if new key/value have the same size (which is practically true currently)
	if len(w.cache) > lenBefore {
		w.memSize += mapMemEst(len(key), len(val))
	}
}

// mayUnload evicts and flushes one batch of data
func (w *backedMap) mayUnload() error {
	for w.memSize > w.maxMemSize {
		err := w.unload(w.batchSize)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *backedMap) unload(toUnload int) error {
	batch := w.backup.NewBatch()
	defer batch.Reset()

	for key, val := range w.cache {
		err := batch.Put([]byte(key), val)
		if err != nil {
			return err
		}

		delete(w.cache, key)
		rmS := mapMemEst(len(key), len(val))
		if rmS <= w.memSize {
			w.memSize -= rmS
		} else {
			w.memSize = 0
		}

		if batch.ValueSize() >= toUnload {
			break
		}
	}

	err := batch.Write()
	if err != nil {
		return err
	}

	return nil
}
