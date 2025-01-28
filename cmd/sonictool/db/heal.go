package db

import (
	"fmt"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"strings"
	"time"
)

func HealChaindata(chaindataDir string, cacheRatio cachescale.Func, cfg *config.Config, lastCarmenBlock idx.Block) (idx.Block, error) {
	producer := &DummyScopedProducer{integration.GetRawDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   cacheRatio.U64(480 * opt.MiB),
		Fdlimit: makeDatabaseHandles(),
	})}
	defer producer.Close()

	log.Info("Healing gossip db...")
	epochState, lastBlock, err := healGossipDb(producer, cfg.OperaStore, lastCarmenBlock)
	if err != nil {
		return 0, err
	}

	log.Info("Removing epoch DBs - will be recreated on next start")
	if err = dropAllEpochDbs(producer); err != nil {
		return 0, err
	}

	log.Info("Recreating consensus database")
	cMainDb, err := producer.OpenDB("lachesis")
	if err != nil {
		return 0, fmt.Errorf("failed to open 'lachesis' database: %w", err)
	}
	cGetEpochDB := func(epoch idx.Epoch) kvdb.Store {
		name := fmt.Sprintf("lachesis-%d", epoch)
		cEpochDB, err := producer.OpenDB(name)
		if err != nil {
			panic(fmt.Errorf("failed to open '%s' database: %w", name, err))
		}
		return cEpochDB
	}
	cdb := abft.NewStore(cMainDb, cGetEpochDB, panics("Lachesis store"), cfg.LachesisStore)
	if err = cdb.ApplyGenesis(&abft.Genesis{
		Epoch:      epochState.Epoch,
		Validators: epochState.Validators,
	}); err != nil {
		return 0, fmt.Errorf("failed to init consensus database: %w", err)
	}
	if err = cdb.Close(); err != nil {
		return 0, fmt.Errorf("failed to close consensus database: %w", err)
	}

	log.Info("Clearing DBs dirty flags")
	if err := clearDirtyFlags(producer); err != nil {
		return 0, fmt.Errorf("failed to write clean FlushID: %w", err)
	}

	return lastBlock, nil
}

// healGossipDb reverts the gossip database into state, into which can be reverted carmen
func healGossipDb(producer kvdb.FlushableDBProducer, cfg gossip.StoreConfig, lastCarmenBlock idx.Block) (
	epochState *iblockproc.EpochState, lastBlock idx.Block, err error) {

	gdb, err := gossip.NewStore(producer, cfg) // requires FlushIDKey present (not clean) in all dbs
	if err != nil {
		return nil, 0, err
	}
	defer gdb.Close()

	// find the last closed epoch with the state available
	epochIdx, blockState, epochState := getLastEpochWithState(gdb, lastCarmenBlock)
	if blockState == nil || epochState == nil {
		return nil, 0, fmt.Errorf("no epoch with available state found")
	}

	// set the historic state to be the current
	log.Info("Reverting to epoch state", "epoch", epochIdx, "block", blockState.LastBlock.Idx)
	gdb.SetBlockEpochState(*blockState, *epochState)
	gdb.FlushBlockEpochState()

	// Service.switchEpochTo
	gdb.SetHighestLamport(0)
	gdb.FlushHighestLamport()

	// removing excessive events (event epoch >= closed epoch)
	log.Info("Removing excessive events")
	gdb.ForEachEventRLP(epochIdx.Bytes(), func(id hash.Event, _ rlp.RawValue) bool {
		gdb.DelEvent(id)
		return true
	})

	return epochState, blockState.LastBlock.Idx, nil
}

// getLastEpochWithState finds the last closed epoch with the state available
func getLastEpochWithState(gdb *gossip.Store, lastCarmenBlock idx.Block) (epochIdx idx.Epoch, blockState *iblockproc.BlockState, epochState *iblockproc.EpochState) {
	currentEpoch := gdb.GetEpoch()
	epochsToTry := idx.Epoch(10000)
	endEpoch := idx.Epoch(1)
	if currentEpoch > epochsToTry {
		endEpoch = currentEpoch - epochsToTry
	}

	for epochIdx = currentEpoch; epochIdx > endEpoch; epochIdx-- {
		blockState, epochState = gdb.GetHistoryBlockEpochState(epochIdx)
		if blockState == nil || epochState == nil {
			log.Info("Last closed epoch is not available", "epoch", epochIdx)
			continue
		}
		firstBlockOfEpoch := blockState.LastBlock.Idx
		if firstBlockOfEpoch > lastCarmenBlock {
			log.Info("State for the last closed epoch is not available", "epoch", epochIdx)
			continue
		}
		log.Info("Last closed epoch with available state found", "epoch", epochIdx)
		return epochIdx, blockState, epochState
	}

	return 0, nil, nil
}

func dropAllEpochDbs(producer kvdb.IterableDBProducer) error {
	for _, name := range producer.Names() {
		if strings.HasPrefix(name, "gossip-") || strings.HasPrefix(name, "lachesis-") || name == "lachesis" {
			log.Info("Removing epoch db", "name", name)
			db, err := producer.OpenDB(name)
			if err != nil {
				return fmt.Errorf("unable to open db %s; %s", name, err)
			}
			_ = db.Close()
			db.Drop()
		}
	}
	return nil
}

// clearDirtyFlags - writes the CleanPrefix into all databases
func clearDirtyFlags(rawProducer kvdb.IterableDBProducer) error {
	id := bigendian.Uint64ToBytes(uint64(time.Now().UnixNano()))
	names := rawProducer.Names()
	for _, name := range names {
		db, err := rawProducer.OpenDB(name)
		if err != nil {
			return err
		}

		err = db.Put(integration.FlushIDKey, append([]byte{flushable.CleanPrefix}, id...))
		if err != nil {
			return fmt.Errorf("failed to write CleanPrefix into %s: %w", name, err)
		}
		log.Info("Database set clean", "name", name)
		if err = db.Close(); err != nil {
			return err
		}
	}
	return nil
}

func panics(name string) func(error) {
	return func(err error) {
		panic(fmt.Errorf("%s failed: %w", name, err))
	}
}

type DummyScopedProducer struct {
	kvdb.IterableDBProducer
}

func (d DummyScopedProducer) NotFlushedSizeEst() int {
	return 0
}

func (d DummyScopedProducer) Flush(_ []byte) error {
	return nil
}

func (d DummyScopedProducer) Initialize(_ []string, flushID []byte) ([]byte, error) {
	return flushID, nil
}

func (d DummyScopedProducer) Close() error {
	return nil
}
