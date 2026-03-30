package gossip

import (
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

//go:generate mockgen -source=table_test.go -destination=table_mock.go -package=gossip

// storeTable is an interface needed to generate a mock for a kvdb.Store.
type storeTable interface {
	kvdb.Store
}

var _ storeTable // to avoid storeTable unused warning.

// storeBatch is an interface needed to generate a mock for a kvdb.Batch.
type storeBatch interface {
	kvdb.Batch
}

var _ storeBatch // to avoid storeBatch unused warning.

// dbIterator is an interface needed to generate a mock for a ethdb.Iterator.
type dbIterator interface {
	ethdb.Iterator
}

var _ dbIterator // to avoid dbIterator unused warning.
