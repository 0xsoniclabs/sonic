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
	"fmt"
	"math/big"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/eventid"
	"github.com/0xsoniclabs/sonic/utils/rlpstore"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/Fantom-foundation/lachesis-base/utils/wlru"
)

// Store is a node persistent storage working over physical key-value database.
type Store struct {
	dbs kvdb.FlushableDBProducer
	cfg StoreConfig

	mainDB kvdb.Store
	evm    *evmstore.Store
	table  struct {
		Version kvdb.Store `table:"_"`

		// Main DAG tables
		BlockEpochState        kvdb.Store `table:"D"`
		BlockEpochStateHistory kvdb.Store `table:"h"`
		Events                 kvdb.Store `table:"e"`
		Blocks                 kvdb.Store `table:"b"`
		EpochBlocks            kvdb.Store `table:"P"`
		Genesis                kvdb.Store `table:"g"`
		UpgradeHeights         kvdb.Store `table:"U"`

		// Sonic Certification Chain tables
		CommitteeCertificates kvdb.Store `table:"C"`
		BlockCertificates     kvdb.Store `table:"c"`

		// P2P-only
		HighestLamport kvdb.Store `table:"l"`

		// Network version
		NetworkVersion kvdb.Store `table:"V"`

		// API-only
		BlockHashes kvdb.Store `table:"B"`
	}

	prevFlushTime atomic.Value

	epochStore atomic.Value

	cache struct {
		Events                 *wlru.Cache `cache:"-"` // store by pointer
		EventIDs               *eventid.Cache
		EventsHeaders          *wlru.Cache  `cache:"-"` // store by pointer
		Blocks                 *wlru.Cache  `cache:"-"` // store by pointer
		BlockHashes            *wlru.Cache  `cache:"-"` // store by value
		BRHashes               *wlru.Cache  `cache:"-"` // store by value
		BlockEpochStateHistory *wlru.Cache  `cache:"-"` // store by pointer
		BlockEpochState        atomic.Value // store by value
		HighestLamport         atomic.Value // store by value
		UpgradeHeights         atomic.Value // store by pointer
		Genesis                atomic.Value // store by value
	}

	rlp rlpstore.Helper

	logger.Instance

	// values needed for flush randomizationAdd comment
	randomOffsetEpoch idx.Epoch // epoch when random offset was selected
	randomOffset      uint64    // random number re-selected in each epoch between 0 and 99
}

// NewMemStore creates temporary gossip store for testing purposes.
func NewMemStore(tb testing.TB) (*Store, error) {
	mems := memorydb.NewProducer("")
	dbs := flushable.NewSyncedPool(mems, []byte{0})

	tmpDir := tb.TempDir()
	cfg := MemTestStoreConfig(tmpDir)
	return NewStore(dbs, cfg)
}

// NewStore creates store over key-value db.
func NewStore(dbs kvdb.FlushableDBProducer, cfg StoreConfig) (*Store, error) {
	mainDB, err := dbs.OpenDB("gossip")
	if err != nil {
		return nil, fmt.Errorf("failed to open gossip db: %w", err)
	}
	s := &Store{
		dbs:           dbs,
		cfg:           cfg,
		mainDB:        mainDB,
		Instance:      logger.New("gossip-store"),
		prevFlushTime: atomic.Value{},
		rlp:           rlpstore.Helper{Instance: logger.New("rlp")},
	}
	s.prevFlushTime.Store(time.Now())

	table.MigrateTables(&s.table, s.mainDB)

	s.initCache()
	s.evm = evmstore.NewStore(s.mainDB, cfg.EVM)

	if err := s.migrateData(); err != nil {
		return nil, fmt.Errorf("failed to migrate gossip db: %w", err)
	}

	return s, nil
}

func (s *Store) initCache() {
	s.cache.Events = s.makeCache(s.cfg.Cache.EventsSize, s.cfg.Cache.EventsNum)
	s.cache.Blocks = s.makeCache(s.cfg.Cache.BlocksSize, s.cfg.Cache.BlocksNum)

	blockHashesNum := s.cfg.Cache.BlocksNum
	blockHashesCacheSize := nominalSize * uint(blockHashesNum)
	s.cache.BlockHashes = s.makeCache(blockHashesCacheSize, blockHashesNum)
	s.cache.BRHashes = s.makeCache(blockHashesCacheSize, blockHashesNum)

	eventsHeadersNum := s.cfg.Cache.EventsNum
	eventsHeadersCacheSize := nominalSize * uint(eventsHeadersNum)
	s.cache.EventsHeaders = s.makeCache(eventsHeadersCacheSize, eventsHeadersNum)

	s.cache.EventIDs = eventid.NewCache(s.cfg.Cache.EventsIDsNum)

	blockEpochStatesNum := s.cfg.Cache.BlockEpochStateNum
	blockEpochStatesSize := nominalSize * uint(blockEpochStatesNum)
	s.cache.BlockEpochStateHistory = s.makeCache(blockEpochStatesSize, blockEpochStatesNum)
}

// Close closes underlying database.
func (s *Store) Close() error {
	// set all tables/caches fields to nil
	table.MigrateTables(&s.table, nil)
	table.MigrateCaches(&s.cache, func() interface{} {
		return nil
	})

	if err := s.mainDB.Close(); err != nil {
		return err
	}
	if err := s.closeEpochStore(); err != nil {
		return err
	}
	if err := s.evm.Close(); err != nil {
		return err
	}
	return nil
}

func (s *Store) IsCommitNeeded() bool {
	// randomize flushing criteria for each epoch so that nodes would desynchronize flushes
	if cur := s.GetEpoch(); s.randomOffsetEpoch != cur {
		s.randomOffset = uint64(rand.Int32N(100))
		s.randomOffsetEpoch = cur
	}
	ratio := 900 + s.randomOffset
	return s.isCommitNeeded(ratio, ratio)
}

func (s *Store) isCommitNeeded(sc, tc uint64) bool {
	period := s.cfg.MaxNonFlushedPeriod * time.Duration(sc) / 1000
	size := (uint64(s.cfg.MaxNonFlushedSize) / 2) * tc / 1000
	return time.Since(s.prevFlushTime.Load().(time.Time)) > period ||
		uint64(s.dbs.NotFlushedSizeEst()) > size
}

// Commit changes.
func (s *Store) Commit() error {
	s.FlushBlockEpochState()
	s.FlushHighestLamport()
	es := s.getAnyEpochStore()
	if es != nil {
		es.FlushHeads()
		es.FlushLastEvents()
	}
	return s.flushDBs()
}

func (s *Store) flushDBs() error {
	now := time.Now()
	s.prevFlushTime.Store(now)
	flushID := bigendian.Uint64ToBytes(uint64(now.UnixNano()))
	return s.dbs.Flush(flushID)
}

func (s *Store) EvmStore() *evmstore.Store {
	return s.evm
}

func (s *Store) AsBaseFeeSource() emitter.BaseFeeSource {
	return &baseFeeSource{store: s}
}

type baseFeeSource struct {
	store *Store
}

func (s *baseFeeSource) GetCurrentBaseFee() *big.Int {
	return s.store.GetBlock(s.store.GetLatestBlockIndex()).BaseFee
}

/*
 * Utils:
 */

func (s *Store) makeCache(weight uint, size int) *wlru.Cache {
	cache, err := wlru.New(weight, size)
	if err != nil {
		s.Log.Crit("Failed to create LRU cache", "err", err)
		return nil
	}
	return cache
}
