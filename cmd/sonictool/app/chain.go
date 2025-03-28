package app

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/utils/caution"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/chain"
	"github.com/0xsoniclabs/sonic/config/flags"
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

	from := consensus.Epoch(1)
	if len(ctx.Args()) > 1 {
		n, err := strconv.ParseUint(ctx.Args().Get(1), 10, 32)
		if err != nil {
			return err
		}
		from = consensus.Epoch(n)
	}
	to := consensus.Epoch(0)
	if len(ctx.Args()) > 2 {
		n, err := strconv.ParseUint(ctx.Args().Get(2), 10, 32)
		if err != nil {
			return err
		}
		to = consensus.Epoch(n)
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
