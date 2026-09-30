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
	"io"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/dbutil/dbcounter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/cachedproducer"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flaggedproducer"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/skipkeys"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type DBsConfig struct {
	RuntimeCache DBCacheConfig
}

type DBCacheConfig struct {
	Cache   uint64
	Fdlimit uint64
}

func GetRawDbProducer(chaindataDir string, cfg DBCacheConfig) kvdb.IterableDBProducer {
	if chaindataDir == "inmemory" || chaindataDir == "" {
		chaindataDir, _ = os.MkdirTemp("", "opera-tmp")
	}
	cacher := func(name string) (int, int) {
		return int(cfg.Cache), int(cfg.Fdlimit)
	}

	rawProducer := dbcounter.Wrap(pebble.NewProducer(chaindataDir, cacher), true)

	if metrics.Enabled() {
		rawProducer = WrapDatabaseWithMetrics(rawProducer)
	}
	return rawProducer
}

func GetDbProducer(chaindataDir string, cfg DBCacheConfig) (kvdb.FullDBProducer, error) {
	rawProducer := GetRawDbProducer(chaindataDir, cfg)
	scopedProducer := flaggedproducer.Wrap(rawProducer, FlushIDKey) // pebble-flg
	_, err := scopedProducer.Initialize(rawProducer.Names(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open existing databases: %v", err)
	}
	cachedProducer := cachedproducer.WrapAll(scopedProducer)
	skippingProducer := skipkeys.WrapAllProducer(cachedProducer, MetadataPrefix)
	return skippingProducer, nil
}

// OpenStore opens the gossip store of the Sonic data directory dataDir, which
// must contain a "chaindata" and a "carmen" subdirectory, including its EVM store.
// configure, if not nil, adjusts the store config before the store is opened. The
// caller must close the returned store and then the database producer.
func OpenStore(
	dataDir string,
	cacheRatio cachescale.Func,
	configure func(*gossip.StoreConfig),
) (_ *gossip.Store, _ kvdb.FullDBProducer, err error) {
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenDir := filepath.Join(dataDir, "carmen")

	if stat, statErr := os.Stat(chaindataDir); statErr != nil || !stat.IsDir() {
		return nil, nil, fmt.Errorf("data directory %q does not contain a chaindata directory", dataDir)
	}
	if stat, statErr := os.Stat(carmenDir); statErr != nil || !stat.IsDir() {
		return nil, nil, fmt.Errorf("data directory %q does not contain a carmen directory", dataDir)
	}

	dbs, err := GetDbProducer(chaindataDir, DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: 100,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database producer: %w", err)
	}
	// On any failure past this point, release what has been opened so far. The
	// store is closed before the producer, mirroring assembly.go.
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, dbs, "failed to close database producer")
		}
	}()

	storeConfig := gossip.DefaultStoreConfig(cacheRatio)
	storeConfig.EVM.StateDb.Directory = carmenDir
	if configure != nil {
		configure(&storeConfig)
	}

	store, err := gossip.NewStore(dbs, storeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open gossip store: %w", err)
	}
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, store, "failed to close gossip store")
		}
	}()

	if err = store.EvmStore().Open(); err != nil {
		return nil, nil, fmt.Errorf("failed to open EVM store: %w", err)
	}
	return store, dbs, nil
}

func isEmpty(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return true
	}
	defer caution.CloseAndReportError(&err, f, "failed to close dir")
	_, err = f.Readdirnames(1)
	return err == io.EOF
}

type GossipStoreAdapter struct {
	*gossip.Store
}

func (g *GossipStoreAdapter) GetEvent(id hash.Event) dag.Event {
	e := g.Store.GetEvent(id)
	if e == nil {
		return nil
	}
	return e
}
