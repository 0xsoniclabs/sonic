package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/0xsoniclabs/carmen/go/database/mpt"
	mptio "github.com/0xsoniclabs/carmen/go/database/mpt/io"
	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

func heal(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}
	cacheRatio, err := cacheScaler(ctx)
	if err != nil {
		return err
	}
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenArchiveDir := filepath.Join(dataDir, "carmen", "archive")
	carmenLiveDir := filepath.Join(dataDir, "carmen", "live")

	archiveInfo, err := os.Stat(carmenArchiveDir)
	if err != nil || !archiveInfo.IsDir() {
		return fmt.Errorf("archive database not found in datadir - only databases with archive can be healed")
	}

	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}

	info, err := mptio.CheckMptDirectoryAndGetInfo(carmenArchiveDir)
	if err != nil {
		return fmt.Errorf("failed to read carmen archive: %w", err)
	}

	if info.Mode != mpt.Immutable {
		return fmt.Errorf("the database in the archive directory is not an archive")
	}

	// Check whether the directory is locked.
	if lock, err := mpt.LockDirectory(carmenArchiveDir); err != nil {
		log.Info("Forcing unlock of directory", "dir", carmenArchiveDir)
		if err := mpt.ForceUnlockDirectory(carmenArchiveDir); err != nil {
			return fmt.Errorf("failed to unlock directory: %w", err)
		}
	} else {
		if err := lock.Release(); err != nil {
			return fmt.Errorf("failed to unlock directory: %w", err)
		}
	}

	archiveCheckpointBlock, err := mpt.GetCheckpointBlock(carmenArchiveDir)
	if err != nil {
		return fmt.Errorf("failed to get checkpoint - probably none has been created in this database yet, healing not possible: %w", err)
	}

	cancelCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	recoveredBlock, err := db.HealChaindata(chaindataDir, cacheRatio, cfg, idx.Block(archiveCheckpointBlock))
	if err != nil {
		return err
	}

	if err = mpt.RestoreBlockHeight(carmenArchiveDir, info.Config, uint64(recoveredBlock)); err != nil {
		return fmt.Errorf("failed to revert archive state to block %d: %w", recoveredBlock, err)
	}
	log.Info("Archive state database reverted", "block", recoveredBlock)

	log.Info("Re-creating live state from the archive...")
	if err := healLiveFromArchive(cancelCtx, carmenLiveDir, carmenArchiveDir, recoveredBlock); err != nil {
		return fmt.Errorf("failed to re-create carmen live state from archive; %w", err)
	}

	log.Info("Healing finished")
	return nil
}

func healLiveFromArchive(ctx context.Context, carmenLiveDir, carmenArchiveDir string, recoveredBlock idx.Block) (err error) {
	if err := os.RemoveAll(carmenLiveDir); err != nil {
		return fmt.Errorf("failed to remove broken live state: %w", err)
	}
	if err := os.MkdirAll(carmenLiveDir, 0700); err != nil {
		return fmt.Errorf("failed to create carmen live dir; %w", err)
	}

	reader, writer := io.Pipe()
	defer caution.CloseAndReportError(&err, reader, "failed to close reader")
	bufReader := bufio.NewReaderSize(reader, 100*1024*1024) // 100 MiB
	bufWriter := bufio.NewWriterSize(writer, 100*1024*1024) // 100 MiB

	var exportErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer caution.CloseAndReportError(&exportErr, writer, "failed to close writer")
		exportErr = mptio.ExportBlockFromArchive(ctx, mptio.NewLog(), carmenArchiveDir, bufWriter, uint64(recoveredBlock))
		if exportErr == nil {
			exportErr = bufWriter.Flush()
		}
	}()

	err = mptio.ImportLiveDb(mptio.NewLog(), carmenLiveDir, bufReader)

	wg.Wait()
	return errors.Join(err, exportErr)
}
