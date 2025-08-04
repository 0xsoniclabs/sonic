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

package evmstore

import (
	"fmt"
	"os"
	"path/filepath"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/topicsdb"
	"github.com/0xsoniclabs/sonic/utils/rlpstore"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/cespare/xxhash/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/elastic/go-freelru"
)

// Store is a node persistent storage working over physical key-value database.
type Store struct {
	cfg StoreConfig

	mainDB kvdb.Store
	table  struct {
		// API-only tables
		Receipts    kvdb.Store `table:"r"`
		TxPositions kvdb.Store `table:"x"`
		Txs         kvdb.Store `table:"X"`
	}

	EvmLogs topicsdb.Index

	cache struct {
		TxPositions *freelru.SyncedLRU[common.Hash, TxPosition]
		Receipts    *freelru.SyncedLRU[idx.Block, types.Receipts]
		EvmBlocks   *freelru.SyncedLRU[idx.Block, *evmcore.EvmBlock]
	}

	rlp rlpstore.Helper

	logger.Instance

	parameters  carmen.Parameters
	carmenState carmen.State
	liveStateDb carmen.StateDB
}

// NewStore creates store over key-value db.
func NewStore(mainDB kvdb.Store, cfg StoreConfig) *Store {
	s := &Store{
		cfg:        cfg,
		mainDB:     mainDB,
		Instance:   logger.New("evm-store"),
		rlp:        rlpstore.Helper{Instance: logger.New("rlp")},
		parameters: cfg.StateDb,
	}

	table.MigrateTables(&s.table, s.mainDB)

	if cfg.DisableLogsIndexing {
		s.EvmLogs = topicsdb.NewDummy()
	} else {
		s.EvmLogs = topicsdb.NewWithThreadPool(mainDB)
	}
	s.initCache()

	return s
}

// Open the StateDB database (after the genesis import)
func (s *Store) Open() error {
	err := s.initCarmen()
	if err != nil {
		return err
	}
	s.carmenState, err = carmen.NewState(s.parameters)
	if err != nil {
		return fmt.Errorf("failed to create carmen state; %s", err)
	}
	s.liveStateDb = carmen.CreateCustomStateDBUsing(s.carmenState, s.cfg.Cache.StateDbCapacity)
	return nil
}

// Close closes underlying database.
func (s *Store) Close() error {
	// set all table/cache fields to nil
	table.MigrateTables(&s.table, nil)
	s.EvmLogs.Close()

	if s.liveStateDb != nil {
		s.Log.Info("Closing State DB...")
		err := s.liveStateDb.Close()
		if err != nil {
			return fmt.Errorf("failed to close State DB: %w", err)
		}
		s.Log.Info("State DB closed")
		s.carmenState = nil
		s.liveStateDb = nil
	}
	return nil
}

func (s *Store) initCache() {
	var err error
	s.cache.Receipts, err = freelru.NewSynced[idx.Block, types.Receipts](uint32(s.cfg.Cache.ReceiptsSize), blockIdToInt)
	if err != nil {
		log.Crit("Failed to create receipts cache", "err", err)
	}
	s.cache.TxPositions, err = freelru.NewSynced[common.Hash, TxPosition](uint32(s.cfg.Cache.TxPositions), hashToInt)
	if err != nil {
		log.Crit("Failed to create tx positions cache", "err", err)
	}
	s.cache.EvmBlocks, err = freelru.NewSynced[idx.Block, *evmcore.EvmBlock](uint32(s.cfg.Cache.EvmBlocksNum), blockIdToInt)
	if err != nil {
		log.Crit("Failed to create EVM blocks cache", "err", err)
	}
}

// IndexLogs indexes EVM logs
func (s *Store) IndexLogs(recs ...*types.Log) {
	err := s.EvmLogs.Push(recs...)
	if err != nil {
		s.Log.Crit("DB logs index error", "err", err)
	}
}

/*
 * Utils:
 */

func (s *Store) initCarmen() error {
	params := s.parameters
	err := os.MkdirAll(params.Directory, 0700)
	if err != nil {
		return fmt.Errorf("failed to create carmen dir \"%s\"; %v", params.Directory, err)
	}
	if s.cfg.SkipArchiveCheck {
		return nil // skip the following check (like for verification)
	}
	liveDir := filepath.Join(params.Directory, "live")
	liveInfo, err := os.Stat(liveDir)
	liveExists := err == nil && liveInfo.IsDir()
	archiveDir := filepath.Join(params.Directory, "archive")
	archiveInfo, err := os.Stat(archiveDir)
	archiveExists := err == nil && archiveInfo.IsDir()

	if liveExists { // not checked if the datadir is empty
		if archiveExists && params.Archive == carmen.NoArchive {
			return fmt.Errorf("starting node with disabled archive (validator mode), but the archive database exists - terminated to avoid archive-live states inconsistencies (remove the datadir/carmen/archive to enforce starting as a validator)")
		}
		if !archiveExists && params.Archive != carmen.NoArchive {
			return fmt.Errorf("starting node with enabled archive (rpc mode), but the archive database does exists - terminated to avoid creating an inconsistent archive database (re-apply genesis and resync the node to switch to archive configuration)")
		}
	}
	return nil
}

func hashToInt(hash common.Hash) uint32 {
	return uint32(xxhash.Sum64(hash[:]))
}
func blockIdToInt(block idx.Block) uint32 {
	return uint32(block)
}
