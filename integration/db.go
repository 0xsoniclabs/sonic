package integration

import (
	"fmt"
	"io"
	"os"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/cachedproducer"
	"github.com/0xsoniclabs/kvdb/flaggedproducer"
	"github.com/0xsoniclabs/kvdb/pebble"
	"github.com/0xsoniclabs/kvdb/skipkeys"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/dbutil/dbcounter"
	"github.com/0xsoniclabs/sonic/utils/dbutil/threads"
	"github.com/ethereum/go-ethereum/metrics"
)

type DBsConfig struct {
	RuntimeCache DBCacheConfig
}

type DBCacheConfig struct {
	Cache   uint64
	Fdlimit uint64
}

func GetRawDbProducer(chaindataDir string, cfg DBCacheConfig) kvdb.IterableDBProducer {
	if chaindataDir == "inmemory" || chaindataDir == "" {
		chaindataDir, _ = os.MkdirTemp("", "opera-tmp")
	}
	cacher := func(name string) (int, int) {
		return int(cfg.Cache), int(cfg.Fdlimit)
	}

	rawProducer := dbcounter.Wrap(pebble.NewProducer(chaindataDir, cacher), true)

	if metrics.Enabled() {
		rawProducer = WrapDatabaseWithMetrics(rawProducer)
	}
	return rawProducer
}

func GetDbProducer(chaindataDir string, cfg DBCacheConfig) (kvdb.FullDBProducer, error) {
	rawProducer := GetRawDbProducer(chaindataDir, cfg)
	scopedProducer := flaggedproducer.Wrap(rawProducer, FlushIDKey) // pebble-flg
	_, err := scopedProducer.Initialize(rawProducer.Names(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open existing databases: %v", err)
	}
	cachedProducer := cachedproducer.WrapAll(scopedProducer)
	skippingProducer := skipkeys.WrapAllProducer(cachedProducer, MetadataPrefix)
	return threads.CountedFullDBProducer(skippingProducer), nil
}

func isEmpty(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return true
	}
	defer caution.CloseAndReportError(&err, f, "failed to close dir")
	_, err = f.Readdirnames(1)
	return err == io.EOF
}

type GossipStoreAdapter struct {
	*gossip.Store
}

func (g *GossipStoreAdapter) GetEvent(id consensus.EventHash) consensus.Event {
	e := g.Store.GetEvent(id)
	if e == nil {
		return nil
	}
	return e
}
