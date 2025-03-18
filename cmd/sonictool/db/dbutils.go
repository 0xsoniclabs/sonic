package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	// DefaultCacheSize is calculated as memory consumption in a worst case scenario with default configuration
	// Average memory consumption might be 3-5 times lower than the maximum
	DefaultCacheSize  = 3600
	ConstantCacheSize = 400
)

// makeDatabaseHandles raises out the number of allowed file handles per process and returns allowance for db.
func makeDatabaseHandles() uint64 {
	limit, err := fdlimit.Maximum()
	if err != nil {
		panic(fmt.Errorf("failed to retrieve file descriptor allowance: %v", err))
	}
	raised, err := fdlimit.Raise(uint64(limit))
	if err != nil {
		panic(fmt.Errorf("failed to raise file descriptor allowance: %v", err))
	}
	return raised/6 + 1
}

func AssertDatabaseNotInitialized(dataDir string) error {
	_, err1 := os.Stat(filepath.Join(dataDir, "chaindata"))
	_, err2 := os.Stat(filepath.Join(dataDir, "carmen"))
	if !errors.Is(err1, os.ErrNotExist) && !errors.Is(err2, os.ErrNotExist) {
		return fmt.Errorf("database directories 'chaindata' and 'carmen' already exists")
	}
	return nil
}

func RemoveDatabase(dataDir string) error {
	err1 := os.RemoveAll(filepath.Join(dataDir, "chaindata"))
	err2 := os.RemoveAll(filepath.Join(dataDir, "carmen"))
	err3 := os.RemoveAll(filepath.Join(dataDir, "errlock"))
	return errors.Join(err1, err2, err3)
}

func MakeDbProducer(chaindataDir string, cacheRatio cachescale.Func) (kvdb.FullDBProducer, error) {
	err := os.MkdirAll(chaindataDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create datadir directory: %w", err)
	}
	return integration.GetDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: makeDatabaseHandles(),
	})
}

// GossipDbParameters are parameters for GossipDb factory func.
type GossipDbParameters struct {
	Dbs                       kvdb.FullDBProducer
	DataDir                   string
	ValidatorMode             bool
	CacheRatio                cachescale.Func
	LiveDbCache, ArchiveCache int64 // in bytes
}

func MakeGossipDb(params GossipDbParameters) (*gossip.Store, error) {
	gdbConfig := gossip.DefaultStoreConfig(params.CacheRatio)
	if params.ValidatorMode {
		gdbConfig.EVM.StateDb.Archive = carmen.NoArchive
		gdbConfig.EVM.DisableLogsIndexing = true
		gdbConfig.EVM.DisableTxHashesIndexing = true
	}

	if params.LiveDbCache > 0 {
		gdbConfig.EVM.StateDb.LiveCache = params.LiveDbCache
	}
	if params.ArchiveCache > 0 {
		gdbConfig.EVM.StateDb.ArchiveCache = params.ArchiveCache
	}
	gdbConfig.EVM.StateDb.Directory = filepath.Join(params.DataDir, "carmen")

	gdb, err := gossip.NewStore(params.Dbs, gdbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossip store: %w", err)
	}
	return gdb, nil
}
