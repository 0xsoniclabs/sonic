package vecmt

import (
	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/cacheutils/wlru"
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/vecengine"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/table"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// IndexCacheConfig - config for cache sizes of Engine
type IndexCacheConfig struct {
	HighestBeforeTimeSize uint
	DBCache               int
}

// IndexConfig - Engine config (cache sizes)
type IndexConfig struct {
	Fc     vecengine.IndexConfig
	Caches IndexCacheConfig
}

// Index is a data to detect forkless-cause condition, calculate median timestamp, detect forks.
type Index struct {
	*vecengine.Engine
	baseCallbacks vecengine.Callbacks

	crit          func(error)
	validators    *consensus.Validators
	validatorIdxs map[consensus.ValidatorID]consensus.ValidatorIndex

	getEvent func(consensus.EventHash) consensus.Event

	vecDb kvdb.Store
	table struct {
		HighestBeforeTime kvdb.Store `table:"T"`
	}

	cache struct {
		HighestBeforeTime *wlru.Cache
	}

	cfg IndexConfig
}

// DefaultConfig returns default index config
func DefaultConfig(scale cachescale.Func) IndexConfig {
	return IndexConfig{
		Fc: vecengine.DefaultConfig(scale),
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: scale.U(160 * 1024),
			DBCache:               scale.I(10 * opt.MiB),
		},
	}
}

// LiteConfig returns default index config for tests
func LiteConfig() IndexConfig {
	return IndexConfig{
		Fc: vecengine.LiteConfig(),
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: 4 * 1024,
		},
	}
}

// NewIndex creates Index instance.
func NewIndex(crit func(error), config IndexConfig) *Index {
	vi := &Index{
		cfg:  config,
		crit: crit,
	}
	engine := vecengine.NewIndex(crit, config.Fc, func(e *vecengine.Engine) vecengine.Callbacks { return vi.GetEngineCallbacks() })

	vi.Engine = engine
	vi.baseCallbacks = vecengine.GetEngineCallbacks(vi.Engine)
	vi.initCaches()

	return vi
}

func (vi *Index) initCaches() {
	vi.cache.HighestBeforeTime, _ = wlru.New(vi.cfg.Caches.HighestBeforeTimeSize, int(vi.cfg.Caches.HighestBeforeTimeSize))
}

// Reset resets buffers.
func (vi *Index) Reset(validators *consensus.Validators, db kvdb.Store, getEvent func(consensus.EventHash) consensus.Event) {
	fdb := WrapByVecFlushable(db, vi.cfg.Caches.DBCache)
	vi.vecDb = fdb
	vi.Engine.Reset(validators, fdb, getEvent)
	vi.getEvent = getEvent
	vi.validators = validators
	vi.validatorIdxs = validators.Idxs()
	vi.onDropNotFlushed()

	table.MigrateTables(&vi.table, vi.vecDb)
}

func (vi *Index) Close() error {
	return vi.vecDb.Close()
}

func (vi *Index) onDropNotFlushed() {
	vi.cache.HighestBeforeTime.Purge()
}

func (vi *Index) GetEngineCallbacks() vecengine.Callbacks {
	return vecengine.Callbacks{
		GetHighestBefore: func(event consensus.EventHash) vecengine.HighestBeforeI {
			return vi.GetHighestBefore(event)
		},
		GetLowestAfter: func(event consensus.EventHash) vecengine.LowestAfterI {
			return vi.baseCallbacks.GetLowestAfter(event)
		},
		SetHighestBefore: func(event consensus.EventHash, b vecengine.HighestBeforeI) {
			vi.SetHighestBefore(event, b.(*HighestBefore))
		},
		SetLowestAfter: func(event consensus.EventHash, i vecengine.LowestAfterI) {
			vi.baseCallbacks.SetLowestAfter(event, i)
		},
		NewHighestBefore: func(size consensus.ValidatorIndex) vecengine.HighestBeforeI {
			return NewHighestBefore(size)
		},
		NewLowestAfter: func(size consensus.ValidatorIndex) vecengine.LowestAfterI {
			return vi.baseCallbacks.NewLowestAfter(size)
		},
		OnDropNotFlushed: func() {
			vi.baseCallbacks.OnDropNotFlushed()
			vi.onDropNotFlushed()
		},
	}
}

// GetMergedHighestBefore returns HighestBefore vector clock without branches, where branches are merged into one
func (vi *Index) GetMergedHighestBefore(id consensus.EventHash) *HighestBefore {
	return vi.Engine.GetMergedHighestBefore(id).(*HighestBefore)
}
