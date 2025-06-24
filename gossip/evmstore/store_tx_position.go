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

/*
	In LRU cache data stored like pointer
*/

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

type TxPosition struct {
	Block       idx.Block
	Event       hash.Event
	EventOffset uint32
	BlockOffset uint32
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
