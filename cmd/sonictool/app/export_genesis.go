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
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/cmd/sonictool/genesis"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gopkg.in/urfave/cli.v1"
)

func exportGenesis(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}
	fileName := ctx.Args().First()
	if fileName == "" {
		return fmt.Errorf("the output file name must be provided as an argument")
	}
	forValidatorMode, err := isValidatorModeSet(ctx)
	if err != nil {
		return err
	}

	cancelCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cacheRatio, err := cacheScaler(ctx)
	if err != nil {
		return err
	}
	chaindataDir := filepath.Join(dataDir, "chaindata")
	dbs, err := integration.GetDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to make DB producer: %v", err)
	}
	defer caution.CloseAndReportError(&err, dbs, "failed to close DB producer")

	gdb, err := db.MakeGossipDb(db.GossipDbParameters{
		Dbs:           dbs,
		DataDir:       dataDir,
		ValidatorMode: false,
		CacheRatio:    cacheRatio,
		LiveDbCache:   ctx.GlobalInt64(flags.LiveDbCacheFlag.Name),
		ArchiveCache:  ctx.GlobalInt64(flags.ArchiveCacheFlag.Name),
	})
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close Gossip DB")

	fileHandler, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, fileHandler, fmt.Sprintf("failed to close file %v", fileName))

	tmpPath := path.Join(dataDir, "tmp-genesis-export")
	_ = os.RemoveAll(tmpPath)
	defer caution.ExecuteAndReportError(&err, func() error { return os.RemoveAll(tmpPath) },
		"failed to remove tmp genesis export dir")

	return genesis.ExportGenesis(cancelCtx, gdb, !forValidatorMode, fileHandler, tmpPath)
}
