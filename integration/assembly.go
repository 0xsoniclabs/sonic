package integration

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/0xsoniclabs/consensus/dagindexer"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"

	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/keycard-go/hexutils"
)

var (
	MetadataPrefix = hexutils.HexToBytes("0068c2927bf842c3e9e2f1364494a33a752db334b9a819534bc9f17d2c3b4e5970008ff519d35a86f29fcaa5aae706b75dee871f65f174fcea1747f2915fc92158f6bfbf5eb79f65d16225738594bffb")
	FlushIDKey     = append(common.CopyBytes(MetadataPrefix), 0x0c)
	TablesKey      = append(common.CopyBytes(MetadataPrefix), 0x0d)
)

type Configs struct {
	Opera         gossip.Config
	OperaStore    gossip.StoreConfig
	Lachesis      consensusengine.Config
	LachesisStore consensusstore.StoreConfig
	VectorClock   dagindexer.IndexConfig
	DBs           DBsConfig
}

func panics(name string) func(error) {
	return func(err error) {
		log.Crit(fmt.Sprintf("%s error", name), "err", err)
	}
}

func getStores(producer kvdb.FlushableDBProducer, cfg Configs) (*gossip.Store, *consensusstore.Store, error) {
	gdb, err := gossip.NewStore(producer, cfg.OperaStore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open gossip store: %w", err)
	}

	cMainDb, err := producer.OpenDB("lachesis")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open lachesis database: %w", err)
	}
	cGetEpochDB := func(epoch consensus.Epoch) kvdb.Store {
		cEpochDb, err := producer.OpenDB(fmt.Sprintf("lachesis-%d", epoch))
		if err != nil {
			panic(fmt.Errorf("failed to open lachesis-%d database: %w", epoch, err))
		}
		return cEpochDb
	}
	cdb := consensusstore.NewStore(cMainDb, cGetEpochDB, panics("Lachesis store"), cfg.LachesisStore)
	return gdb, cdb, nil
}

func rawMakeEngine(gdb *gossip.Store, cdb *consensusstore.Store, cfg Configs) (*consensusengine.Lachesis, *dagindexer.Index, gossip.BlockProc, error) {
	blockProc := gossip.DefaultBlockProc()
	// create consensus
	vecClock := dagindexer.NewIndex(panics("Vector clock"), cfg.VectorClock)
	engine := consensusengine.NewLachesis(cdb, &GossipStoreAdapter{gdb}, vecClock, panics("Lachesis"), cfg.Lachesis)
	return engine, vecClock, blockProc, nil
}

func makeEngine(chaindataDir string, cfg Configs) (engine *consensusengine.Lachesis, vecClock *dagindexer.Index,
	gdb *gossip.Store, cdb *consensusstore.Store, blockProc gossip.BlockProc, dbsClose func() error, err error) {
	dbs, err := GetDbProducer(chaindataDir, cfg.DBs.RuntimeCache)
	if err != nil {
		return nil, nil, nil, nil, gossip.BlockProc{}, nil, err
	}

	gdb, cdb, err = getStores(dbs, cfg)
	if err != nil {
		err = fmt.Errorf("failed to get stores: %w", err)
		return nil, nil, nil, nil, gossip.BlockProc{}, nil, err
	}
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, cdb, "failed to close lachesis store")
		}
	}()
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, gdb, "failed to close gossip store")
		}
	}()
	defer func() {
		if err != nil {
			caution.CloseAndReportError(&err, dbs, "failed to close db producer")
		}
	}()

	err = gdb.EvmStore().Open()
	dbsClose = dbs.Close
	if err != nil {
		err = fmt.Errorf("failed to open EvmStore: %v", err)
		return nil, nil, nil, nil, gossip.BlockProc{}, dbsClose, err
	}

	engine, vecClock, blockProc, err = rawMakeEngine(gdb, cdb, cfg)
	if err != nil {
		err = fmt.Errorf("failed to make engine: %v", err)
		return nil, nil, nil, nil, gossip.BlockProc{}, dbsClose, err
	}

	return engine, vecClock, gdb, cdb, blockProc, dbsClose, nil
}

// MakeEngine makes consensus engine from config.
func MakeEngine(chaindataDir string, cfg Configs) (*consensusengine.Lachesis, *dagindexer.Index, *gossip.Store, *consensusstore.Store, gossip.BlockProc, func() error, error) {
	if isEmpty(chaindataDir) || isInterrupted(chaindataDir) {
		return nil, nil, nil, nil, gossip.BlockProc{}, nil, fmt.Errorf("database is empty or the genesis import interrupted")
	}

	engine, vecClock, gdb, cdb, blockProc, closeDBs, err := makeEngine(chaindataDir, cfg)
	if err != nil {
		return nil, nil, nil, nil, gossip.BlockProc{}, nil, err
	}

	rules := gdb.GetRules()
	genesisID := gdb.GetGenesisID()
	log.Info("Genesis is written", "name", rules.Name, "id", rules.NetworkID, "genesis", genesisID.String())

	return engine, vecClock, gdb, cdb, blockProc, closeDBs, nil
}

// SetAccountKey sets key into accounts manager and unlocks it with pswd.
func SetAccountKey(
	am *accounts.Manager, key *ecdsa.PrivateKey, pswd string,
) (
	acc accounts.Account,
) {
	kss := am.Backends(keystore.KeyStoreType)
	if len(kss) < 1 {
		log.Crit("Keystore is not found")
		return
	}
	ks := kss[0].(*keystore.KeyStore)

	acc = accounts.Account{
		Address: crypto.PubkeyToAddress(key.PublicKey),
	}

	imported, err := ks.ImportECDSA(key, pswd)
	if err == nil {
		acc = imported
	} else if err.Error() != "account already exists" {
		log.Crit("Failed to import key", "err", err)
	}

	err = ks.Unlock(acc, pswd)
	if err != nil {
		log.Crit("failed to unlock key", "err", err)
	}

	return
}
