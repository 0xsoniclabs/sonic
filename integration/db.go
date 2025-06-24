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

package integration

import (
	"fmt"
	"io"
	"os"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/dbutil/dbcounter"
	"github.com/0xsoniclabs/sonic/utils/dbutil/threads"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/cachedproducer"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flaggedproducer"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/skipkeys"
	"github.com/ethereum/go-ethereum/metrics"
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
	return threads.CountedFullDBProducer(skippingProducer), nil
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
