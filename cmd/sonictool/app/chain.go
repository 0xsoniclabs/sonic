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
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/utils/caution"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/chain"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

func exportEvents(ctx *cli.Context) (err error) {
	if len(ctx.Args()) < 1 {
		return fmt.Errorf("this command requires an argument - the output file")
	}

	filename := ctx.Args().First()

	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}

	// Open the file handle and potentially wrap with a gzip stream
	fileHandler, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, fileHandler, fmt.Sprintf("failed to close file %v", filename))

	var writer io.Writer = fileHandler
	if strings.HasSuffix(filename, ".gz") {
		writer = gzip.NewWriter(writer)
		defer caution.CloseAndReportError(&err,
			writer.(*gzip.Writer),
			fmt.Sprintf("failed to close gzip writer for file %v", filename))
	}

	from := idx.Epoch(1)
	if len(ctx.Args()) > 1 {
		n, err := strconv.ParseUint(ctx.Args().Get(1), 10, 32)
		if err != nil {
			return err
		}
		from = idx.Epoch(n)
	}
	to := idx.Epoch(0)
	if len(ctx.Args()) > 2 {
		n, err := strconv.ParseUint(ctx.Args().Get(2), 10, 32)
		if err != nil {
			return err
		}
		to = idx.Epoch(n)
	}

	gdbParams := db.GossipDbParameters{
		DataDir:      dataDir,
		LiveDbCache:  ctx.GlobalInt64(flags.LiveDbCacheFlag.Name),
		ArchiveCache: ctx.GlobalInt64(flags.ArchiveCacheFlag.Name),
	}

	log.Info("Exporting events to file", "file", filename)
	err = chain.ExportEvents(gdbParams, writer, from, to)
	if err != nil {
		return fmt.Errorf("export error: %w", err)
	}

	return nil
}

func importEvents(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		return fmt.Errorf("this command requires an argument - the input file")
	}

	err := chain.EventsImport(ctx, ctx.Args()...)
	if err != nil {
		return err
	}

	return nil
}
