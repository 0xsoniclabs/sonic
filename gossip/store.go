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
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/eventid"
	"github.com/0xsoniclabs/sonic/utils/rlpstore"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/Fantom-foundation/lachesis-base/utils/wlru"
	"github.com/cespare/xxhash/v2"
	"github.com/elastic/go-freelru"
	"github.com/ethereum/go-ethereum/common"
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
		EventIDs *eventid.Cache

		Events        *freelru.SyncedLRU[hash.Event, *inter.EventPayload]
		EventsHeaders *freelru.SyncedLRU[hash.Event, *inter.Event]
		Blocks        *freelru.SyncedLRU[idx.Block, *inter.Block]

		BlockHashes            *freelru.SyncedLRU[common.Hash, idx.Block]
		BlockEpochStateHistory *freelru.SyncedLRU[idx.Epoch, *BlockEpochState]
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

	s.cache.EventIDs = eventid.NewCache(s.cfg.Cache.EventsIDsNum)
	var err error

	s.cache.Events, err = freelru.NewSynced[hash.Event, *inter.EventPayload](uint32(s.cfg.Cache.EventsSize), eventHashToInt)
	if err != nil {
		s.Log.Crit("Failed to create freelru cache for events", "err", err)
	}

	s.cache.Blocks, err = freelru.NewSynced[idx.Block, *inter.Block](uint32(s.cfg.Cache.BlocksSize), func(b idx.Block) uint32 {
		return uint32(b)
	})
	if err != nil {
		s.Log.Crit("Failed to create freelru cache for blocks", "err", err)
	}

	s.cache.BlockHashes, err = freelru.NewSynced[common.Hash, idx.Block](uint32(s.cfg.Cache.BlocksNum), func(h common.Hash) uint32 {
		return uint32(xxhash.Sum64(h[:]))
	})
	if err != nil {
		s.Log.Crit("Failed to create freelru cache for block hashes", "err", err)
	}

	s.cache.EventsHeaders, err = freelru.NewSynced[hash.Event, *inter.Event](uint32(s.cfg.Cache.EventsIDsNum), eventHashToInt)
	if err != nil {
		s.Log.Crit("Failed to create freelru cache for events headers", "err", err)
	}

	s.cache.BlockEpochStateHistory, err = freelru.NewSynced[idx.Epoch, *BlockEpochState](uint32(s.cfg.Cache.BlockEpochStateNum), func(epoch idx.Epoch) uint32 {
		return uint32(xxhash.Sum64(epoch.Bytes()))
	})
	if err != nil {
		s.Log.Crit("Failed to create freelru cache for block epoch states", "err", err)
	}
}

// Close closes underlying database.
func (s *Store) Close() error {
	// set all tables/caches fields to nil
	table.MigrateTables(&s.table, nil)

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

func eventHashToInt(event hash.Event) uint32 {
	return uint32(xxhash.Sum64(event[:]))
}
