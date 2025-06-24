// Copyright 2025 Sonic Operations Ltd
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

package evmstore

import (
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type (
	// StoreCacheConfig is a config for the db.
	StoreCacheConfig struct {
		// Cache size for Receipts (size in bytes).
		ReceiptsSize uint
		// Cache size for Receipts (number of blocks).
		ReceiptsBlocks int
		// Cache size for TxPositions.
		TxPositions int
		// Cache size for EvmBlock (number of blocks).
		EvmBlocksNum int
		// Cache size for EvmBlock (size in bytes).
		EvmBlocksSize uint
		// Cache size for StateDb instances (0 for DB-selected default)
		StateDbCapacity int
	}
	// StoreConfig is a config for store db.
	StoreConfig struct {
		Cache StoreCacheConfig
		// Carmen StateDB config
		StateDb carmen.Parameters
		// Skip running with a different archive mode prevention
		SkipArchiveCheck bool
		// Disables EVM logs indexing
		DisableLogsIndexing bool
		// Disables storing of txs positions
		DisableTxHashesIndexing bool
	}
)

// DefaultStoreConfig for product.
func DefaultStoreConfig(scale cachescale.Func) StoreConfig {
	return StoreConfig{
		Cache: StoreCacheConfig{
			ReceiptsSize:   scale.U(4 * opt.MiB),
			ReceiptsBlocks: scale.I(4000),
			TxPositions:    scale.I(20000),
			EvmBlocksNum:   scale.I(5000),
			EvmBlocksSize:  scale.U(6 * opt.MiB),
		},
		StateDb: carmen.Parameters{
			Variant:      "go-file",
			Schema:       carmen.Schema(5),
			Archive:      carmen.S5Archive,
			LiveCache:    scale.I64(1940 * opt.MiB),
			ArchiveCache: scale.I64(1940 * opt.MiB),
		},
	}
}

// LiteStoreConfig is for tests or inmemory.
func LiteStoreConfig() StoreConfig {
	return DefaultStoreConfig(cachescale.Ratio{Base: 10, Target: 1})
}
