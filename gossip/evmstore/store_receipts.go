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

package evmstore

/*
	In LRU cache data stored like value
*/

import (
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// SetReceipts stores transaction receipts.
func (s *Store) SetReceipts(n idx.Block, receipts types.Receipts) {
	receiptsStorage := make([]*types.ReceiptForStorage, receipts.Len())
	for i, r := range receipts {
		receiptsStorage[i] = (*types.ReceiptForStorage)(r)
	}

	size := s.SetRawReceipts(n, receiptsStorage)
	s.setGasUsedOverrides(n, findGasUsedOverrides(receipts))

	// Add to LRU cache.
	s.cache.Receipts.Add(n, receipts, uint(size))
}

// SetRawReceipts stores raw transaction receipts.
func (s *Store) SetRawReceipts(n idx.Block, receipts []*types.ReceiptForStorage) (size int) {
	buf, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		s.Log.Crit("Failed to encode rlp", "err", err)
	}

	if err := s.table.Receipts.Put(n.Bytes(), buf); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}

	// Remove from LRU cache.
	s.cache.Receipts.Remove(n)

	return len(buf)
}

func (s *Store) GetRawReceiptsRLP(n idx.Block) rlp.RawValue {
	buf, err := s.table.Receipts.Get(n.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	return buf
}

func (s *Store) GetRawReceipts(n idx.Block) ([]*types.ReceiptForStorage, int) {
	buf := s.GetRawReceiptsRLP(n)
	if buf == nil {
		return nil, 0
	}

	var receiptsStorage []*types.ReceiptForStorage
	err := rlp.DecodeBytes(buf, &receiptsStorage)
	if err != nil {
		s.Log.Crit("Failed to decode rlp", "err", err, "size", len(buf))
	}
	return receiptsStorage, len(buf)
}

func UnwrapStorageReceipts(receiptsStorage []*types.ReceiptForStorage, n idx.Block, config *params.ChainConfig, hash common.Hash, time uint64, baseFee *big.Int, blobGasPrice *big.Int, txs types.Transactions) (types.Receipts, error) {
	receipts := make(types.Receipts, len(receiptsStorage))
	for i, r := range receiptsStorage {
		receipts[i] = (*types.Receipt)(r)
	}
	err := receipts.DeriveFields(config, hash, uint64(n), time, baseFee, blobGasPrice, txs)
	return receipts, err
}

// GetReceipts returns stored transaction receipts.
func (s *Store) GetReceipts(n idx.Block, config *params.ChainConfig, hash common.Hash, time uint64, baseFee *big.Int, blobGasPrice *big.Int, txs types.Transactions) types.Receipts {
	// Get data from LRU cache first.
	if s.cache.Receipts != nil {
		if c, ok := s.cache.Receipts.Get(n); ok {
			return c.(types.Receipts)
		}
	}

	receiptsStorage, size := s.GetRawReceipts(n)

	receipts, err := UnwrapStorageReceipts(receiptsStorage, n, config, hash, time, baseFee, blobGasPrice, txs)
	if err != nil {
		s.Log.Crit("Failed to derive receipts", "err", err)
	}
	s.applyGasUsedOverrides(n, receipts)

	// Add to LRU cache.
	s.cache.Receipts.Add(n, receipts, uint(size))

	return receipts
}

// gasUsedOverride is the gas used of the receipt at the given index in a block,
// for receipts whose gas used differs from the delta of their cumulative gas
// used and the one of their predecessor.
type gasUsedOverride struct {
	Index   uint64
	GasUsed uint64
}

// findGasUsedOverrides identifies receipts directly following a gas surplus,
// being gas charged to the block that is not reported by any receipt.
//
// Before the Canto upgrade, a failed transaction bundle whose execution plan
// root is a single transaction was not rolled back. Its gas was added to the
// block's used gas, and thus to the cumulative gas used of all following
// receipts, but neither the transaction nor its receipt became part of the
// block. Those values are part of the block hash and must not be altered.
//
// Only the cumulative gas used is stored with receipts. When loading them, the
// gas used of a receipt is derived as the difference to its predecessor, which
// for the first receipt after a gas surplus includes that surplus. To
// restore the gas used reported by the execution, the actual gas used is
// recorded for each receipt for which it is smaller than the derived value.
func findGasUsedOverrides(receipts types.Receipts) []gasUsedOverride {
	var res []gasUsedOverride
	previous := uint64(0)
	for i, r := range receipts {
		if r.CumulativeGasUsed >= previous && r.GasUsed < r.CumulativeGasUsed-previous {
			res = append(res, gasUsedOverride{Index: uint64(i), GasUsed: r.GasUsed})
		}
		previous = r.CumulativeGasUsed
	}
	return res
}

// setGasUsedOverrides replaces the gas used overrides recorded for a block.
func (s *Store) setGasUsedOverrides(n idx.Block, overrides []gasUsedOverride) {
	if len(overrides) == 0 {
		// Drop overrides left behind by a previous execution of this block.
		// Checking first avoids writing a deletion marker for every block.
		has, err := s.table.ReceiptGasUsedOverrides.Has(n.Bytes())
		if err != nil {
			s.Log.Crit("Failed to check key", "err", err)
		}
		if has {
			if err := s.table.ReceiptGasUsedOverrides.Delete(n.Bytes()); err != nil {
				s.Log.Crit("Failed to delete key", "err", err)
			}
		}
		return
	}
	buf, err := rlp.EncodeToBytes(overrides)
	if err != nil {
		s.Log.Crit("Failed to encode rlp", "err", err)
	}
	if err := s.table.ReceiptGasUsedOverrides.Put(n.Bytes(), buf); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// applyGasUsedOverrides restores the gas used of receipts of a block for which
// it can not be derived from the stored cumulative gas used.
func (s *Store) applyGasUsedOverrides(n idx.Block, receipts types.Receipts) {
	buf, err := s.table.ReceiptGasUsedOverrides.Get(n.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if len(buf) == 0 {
		return
	}
	var overrides []gasUsedOverride
	if err := rlp.DecodeBytes(buf, &overrides); err != nil {
		s.Log.Crit("Failed to decode rlp", "err", err, "size", len(buf))
	}
	for _, o := range overrides {
		if o.Index >= uint64(len(receipts)) {
			s.Log.Error("Gas used override out of range", "block", n, "index", o.Index, "receipts", len(receipts))
			continue
		}
		receipts[o.Index].GasUsed = o.GasUsed
	}
}
