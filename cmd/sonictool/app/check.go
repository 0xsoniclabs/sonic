package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/check"
	"github.com/0xsoniclabs/sonic/config/flags"
	"gopkg.in/urfave/cli.v1"
)

func checkLive(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}
	cacheRatio, err := cacheScaler(ctx)
	if err != nil {
		return err
	}

	cancelCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return check.CheckLiveStateDb(cancelCtx, dataDir, cacheRatio)
}

func checkArchive(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s need to be set", flags.DataDirFlag.Name)
	}
	cacheRatio, err := cacheScaler(ctx)
	if err != nil {
		return err
	}

	cancelCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return check.CheckArchiveStateDb(cancelCtx, dataDir, cacheRatio)
}
