// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

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
