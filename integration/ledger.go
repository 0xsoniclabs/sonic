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

package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Ledger is a standalone, store-owning block ledger opened from a data
// directory. It embeds the gossip.Ledger interface, so it can replay and process
// blocks (BeginBlock -> Run -> Finalize -> Commit -> Publish), and it owns the
// underlying database resources, which Close releases. It is intended for tools
// and tests that need to drive a ledger over an on-disk store without bringing up
// a full node.
type Ledger struct {
	gossip.Ledger
	store *gossip.Store
	dbs   kvdb.FullDBProducer
}

// OpenLedger opens the on-disk store under dataDir and returns a Ledger over it.
// dataDir is the standard Sonic node data directory, expected to contain a
// "chaindata" (key-value databases) and a "carmen" (EVM state) subdirectory. The
// returned Ledger owns the opened resources and must be closed with Close.
func OpenLedger(dataDir string, config gossip.LedgerConfig) (_ *Ledger, err error) {
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenDir := filepath.Join(dataDir, "carmen")

	if stat, statErr := os.Stat(chaindataDir); statErr != nil || !stat.IsDir() {
		return nil, fmt.Errorf("data directory %q does not contain a chaindata directory", dataDir)
	}
	if stat, statErr := os.Stat(carmenDir); statErr != nil || !stat.IsDir() {
		return nil, fmt.Errorf("data directory %q does not contain a carmen directory", dataDir)
	}

	cacheRatio := cachescale.Identity
	dbs, err := GetDbProducer(chaindataDir, DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database producer: %w", err)
	}
	// On any failure past this point, release what has been opened so far. The
	// store is closed before the producer, mirroring Close and assembly.go.
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, dbs, "failed to close database producer")
		}
	}()

	storeConfig := gossip.DefaultStoreConfig(cacheRatio)
	storeConfig.EVM.StateDb.Directory = carmenDir

	store, err := gossip.NewStore(dbs, storeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open gossip store: %w", err)
	}
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, store, "failed to close gossip store")
		}
	}()

	if err = store.EvmStore().Open(); err != nil {
		return nil, fmt.Errorf("failed to open EVM store: %w", err)
	}

	return &Ledger{
		Ledger: gossip.NewLedger(store, config),
		store:  store,
		dbs:    dbs,
	}, nil
}

// Close releases the resources owned by the ledger: the gossip store and then the
// database producer. Errors from both are reported.
func (l *Ledger) Close() (err error) {
	caution.CloseAndReportError(&err, l.store, "failed to close gossip store")
	caution.CloseAndReportError(&err, l.dbs, "failed to close database producer")
	return err
}
