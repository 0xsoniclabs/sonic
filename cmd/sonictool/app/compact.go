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

package app

import (
	"fmt"
	"path/filepath"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/dbutil"
	"github.com/0xsoniclabs/sonic/utils/dbutil/compactdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gopkg.in/urfave/cli.v1"
)

func compactDbs(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}
	cacheRatio, err := cacheScaler(ctx)
	if err != nil {
		return err
	}
	chaindataDir := filepath.Join(dataDir, "chaindata")
	dbs := integration.GetRawDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: 100,
	})

	for _, name := range dbs.Names() {
		if err := compactDB(name, dbs); err != nil {
			return err
		}
	}
	return nil
}

func compactDB(name string, producer kvdb.DBProducer) (err error) {
	db, err := producer.OpenDB(name)
	if err != nil {
		log.Error("Cannot open db or db does not exists", "db", name)
		return err
	}
	defer caution.CloseAndReportError(&err, db, "failed to close db")

	log.Info("Stats before compaction", "db", name)
	showDbStats(db)

	err = compactdb.Compact(db, name, 64*opt.GiB)
	if err != nil {
		log.Error("Database compaction failed", "err", err)
		return err
	}

	log.Info("Stats after compaction", "db", name)
	showDbStats(db)

	return nil
}

func showDbStats(db ethdb.KeyValueStater) {
	if stats, err := db.Stat(); err != nil {
		log.Warn("Failed to read database stats", "error", err)
	} else {
		fmt.Println(stats)
	}
	measurableStore, isMeasurable := db.(dbutil.MeasurableStore)
	if !isMeasurable {
		log.Warn("Failed to read database iostats - not a MeasurableStore")
		return
	}
	if ioStats, err := measurableStore.IoStats(); err != nil {
		log.Warn("Failed to read database iostats", "error", err)
	} else {
		fmt.Println(ioStats)
	}
}
