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
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/skiperrors"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"

	"github.com/0xsoniclabs/sonic/logger"
)

var (
	errDBClosed = errors.New("database closed")
)

type (
	epochStore struct {
		epoch idx.Epoch
		db    kvdb.Store
		table struct {
			LastEvents kvdb.Store `table:"t"`
			Heads      kvdb.Store `table:"H"`
			DagIndex   kvdb.Store `table:"v"`
		}
		cache struct {
			Heads      atomic.Value
			LastEvents atomic.Value
		}

		logger.Instance
	}
)

func newEpochStore(epoch idx.Epoch, db kvdb.Store) *epochStore {
	es := &epochStore{
		epoch:    epoch,
		db:       db,
		Instance: logger.New("epoch-store"),
	}
	table.MigrateTables(&es.table, db)

	// wrap with skiperrors to skip errors on reading from a dropped DB
	es.table.LastEvents = skiperrors.Wrap(es.table.LastEvents, errDBClosed)
	es.table.Heads = skiperrors.Wrap(es.table.Heads, errDBClosed)

	// load the cache to avoid a race condition
	es.GetHeads()
	es.GetLastEvents()

	return es
}

func (s *Store) getAnyEpochStore() *epochStore {
	_es := s.epochStore.Load()
	if _es == nil {
		return nil
	}
	return _es.(*epochStore)
}

// getEpochStore is safe for concurrent use.
func (s *Store) getEpochStore(epoch idx.Epoch) *epochStore {
	es := s.getAnyEpochStore()
	if es.epoch != epoch {
		return nil
	}
	return es
}

func (s *Store) resetEpochStore(newEpoch idx.Epoch) {
	oldEs := s.epochStore.Load()
	// create new DB
	s.createEpochStore(newEpoch)
	// drop previous DB
	// there may be race condition with threads which hold this DB, so wrap tables with skiperrors
	if oldEs != nil {
		err := oldEs.(*epochStore).db.Close()
		if err != nil {
			s.Log.Error("Failed to close epoch DB", "err", err)
			return
		}
		oldEs.(*epochStore).db.Drop()
	}
}

func (s *Store) loadEpochStore(epoch idx.Epoch) {
	if s.epochStore.Load() != nil {
		return
	}
	s.createEpochStore(epoch)
}

func (s *Store) closeEpochStore() error {
	es := s.getAnyEpochStore()
	if es == nil {
		return nil
	}
	return es.db.Close()
}

func (s *Store) createEpochStore(epoch idx.Epoch) {
	// create new DB
	name := fmt.Sprintf("gossip-%d", epoch)
	db, err := s.dbs.OpenDB(name)
	if err != nil {
		s.Log.Crit("Filed to open DB", "name", name, "err", err)
	}
	s.epochStore.Store(newEpochStore(epoch, db))
}
