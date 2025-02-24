package evmstore

import (
	"github.com/0xsoniclabs/consensus/kvdb/memorydb"
)

func cachedStore() *Store {
	cfg := LiteStoreConfig()

	store := NewStore(memorydb.New(), cfg)
	return store
}

func nonCachedStore() *Store {
	cfg := StoreConfig{}

	store := NewStore(memorydb.New(), cfg)
	return store
}
