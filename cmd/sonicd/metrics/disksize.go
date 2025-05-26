package metrics

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var once sync.Once

func SetDataDir(datadir string) {
	once.Do(func() {
		go measureDbDir(context.Background(), "db_size", datadir, time.Minute)
		go measureDbDir(context.Background(), "statedb/disksize",
			filepath.Join(datadir, "carmen"), time.Minute)
	})
}

func measureDbDir(ctx context.Context, name, datadir string, delay time.Duration) {
	var (
		gauge  = metrics.GetOrRegisterGauge(name, nil)
		rescan = len(datadir) > 0 && datadir != "inmemory"
	)
	for rescan {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			size := sizeOfDir(datadir)
			gauge.Update(size)
		}
	}
}

func sizeOfDir(dir string) (size int64) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Debug("datadir walk", "path", path, "err", err)
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		dst, err := filepath.EvalSymlinks(path)
		if err == nil && dst != path {
			size += sizeOfDir(dst)
		} else {
			size += info.Size()
		}

		return nil
	})

	if err != nil {
		log.Debug("datadir walk", "path", dir, "err", err)
	}

	return
}
