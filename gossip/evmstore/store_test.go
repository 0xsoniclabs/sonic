package evmstore

import (
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
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
