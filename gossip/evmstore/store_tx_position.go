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
	In LRU cache data stored like pointer
*/

import (
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type TxPosition struct {
	Block       idx.Block
	BlockOffset uint32
}

// DecodeRLP decodes a TxPosition, tolerating the historical 4-element layout
// [Block, Event, EventOffset, BlockOffset] in addition to the current
// [Block, BlockOffset]. In both layouts Block is the first element and
// BlockOffset is the last, so older records remain readable without a reindex.
func (p *TxPosition) DecodeRLP(s *rlp.Stream) error {
	var raw []rlp.RawValue
	if err := s.Decode(&raw); err != nil {
		return err
	}
	if len(raw) != 2 && len(raw) != 4 {
		return fmt.Errorf("unexpected TxPosition arity %d", len(raw))
	}
	if err := rlp.DecodeBytes(raw[0], &p.Block); err != nil {
		return err
	}
	return rlp.DecodeBytes(raw[len(raw)-1], &p.BlockOffset)
}

// SetTxPosition stores transaction block and position.
func (s *Store) SetTxPosition(txid common.Hash, position TxPosition) {
	if s.cfg.DisableTxHashesIndexing {
		return
	}

	s.rlp.Set(s.table.TxPositions, txid.Bytes(), &position)

	// Add to LRU cache.
	s.cache.TxPositions.Add(txid.String(), &position, nominalSize)
}

// GetTxPosition returns stored transaction block and position.
func (s *Store) GetTxPosition(txid common.Hash) *TxPosition {
	if s.cfg.DisableTxHashesIndexing {
		return nil
	}

	// Get data from LRU cache first.
	if c, ok := s.cache.TxPositions.Get(txid.String()); ok {
		if b, ok := c.(*TxPosition); ok {
			return b
		}
	}

	txPosition, _ := s.rlp.Get(s.table.TxPositions, txid.Bytes(), &TxPosition{}).(*TxPosition)

	// Add to LRU cache.
	if txPosition != nil {
		s.cache.TxPositions.Add(txid.String(), txPosition, nominalSize)
	}

	return txPosition
}
