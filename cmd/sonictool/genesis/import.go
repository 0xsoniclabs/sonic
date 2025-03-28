package genesis

import (
	"fmt"
	"path/filepath"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"

	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/ethereum/go-ethereum/log"
)

// ImportParams are parameters for ImportGenesisStore func.
type ImportParams struct {
	GenesisStore              *genesisstore.Store
	DataDir                   string
	ValidatorMode             bool
	CacheRatio                cachescale.Func
	LiveDbCache, ArchiveCache int64 // in bytes
}

func ImportGenesisStore(params ImportParams) (err error) {
	if err := db.AssertDatabaseNotInitialized(params.DataDir); err != nil {
		return fmt.Errorf("database in datadir is already initialized: %w", err)
	}
	if err := db.RemoveDatabase(params.DataDir); err != nil {
		return fmt.Errorf("failed to remove existing data from the datadir: %w", err)
	}

	chaindataDir := filepath.Join(params.DataDir, "chaindata")
	dbs, err := db.MakeDbProducer(chaindataDir, params.CacheRatio)
	if err != nil {
		return fmt.Errorf("failed to create db producer: %w", err)
	}
	defer caution.CloseAndReportError(&err, dbs, "failed to close db producer")
	setGenesisProcessing(chaindataDir)

	gdb, err := db.MakeGossipDb(db.GossipDbParameters{
		Dbs:           dbs,
		DataDir:       params.DataDir,
		ValidatorMode: params.ValidatorMode,
		CacheRatio:    params.CacheRatio,
		LiveDbCache:   params.LiveDbCache,
		ArchiveCache:  params.ArchiveCache,
	})
	if err != nil {
		return fmt.Errorf("failed to create gossip db: %w", err)
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close gossip db")

	err = gdb.ApplyGenesis(params.GenesisStore.Genesis())
	if err != nil {
		return fmt.Errorf("failed to write Gossip genesis state: %w", err)
	}

	cMainDb, err := dbs.OpenDB("lachesis")
	if err != nil {
		return fmt.Errorf("failed to open lachesis db: %w", err)
	}
	cGetEpochDB := func(epoch consensus.Epoch) kvdb.Store {
		db, err := dbs.OpenDB(fmt.Sprintf("lachesis-%d", epoch))
		if err != nil {
			panic(fmt.Errorf("failed to open epoch db: %w", err))
		}
		return db
	}
	abftCrit := func(err error) {
		panic(fmt.Errorf("lachesis store error: %w", err))
	}
	cdb := consensusstore.NewStore(cMainDb, cGetEpochDB, abftCrit, consensusstore.DefaultStoreConfig(params.CacheRatio))
	defer caution.CloseAndReportError(&err, cdb, "failed to close consensus db")

	err = cdb.ApplyGenesis(&consensusstore.Genesis{
		Epoch:      gdb.GetEpoch(),
		Validators: gdb.GetValidators(),
	})
	if err != nil {
		return fmt.Errorf("failed to write lachesis genesis state: %w", err)
	}

	err = gdb.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit gossip db: %w", err)
	}
	setGenesisComplete(chaindataDir)
	log.Info("Successfully imported genesis file")
	return nil
}

func IsGenesisTrusted(genesisStore *genesisstore.Store, genesisHashes genesis.Hashes) error {
	g := genesisStore.Genesis()

	// try trusted hashes first
	for _, allowed := range allowedGenesis {
		if allowed.Hashes.Equal(genesisHashes) && allowed.Header.Equal(g.Header) {
			return nil
		}
	}

	// try using SignedMetadata section
	hash, _, err := GetGenesisMetadata(g.Header, genesisHashes)
	if err != nil {
		return fmt.Errorf("failed to calculate hash of genesis: %w", err)
	}
	signature, err := g.SignatureSection.GetSignature()
	if err != nil {
		return fmt.Errorf("genesis file doesn't refer to any trusted preset, signature not found: %w", err)
	}
	if err := CheckGenesisSignature(hash, signature); err != nil {
		return fmt.Errorf("genesis file doesn't refer to any trusted preset: %w", err)
	}
	return nil
}
