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

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// This file implements the storage of receipt gas amendments. Before Canto, a
// failing bundle with a bare execution plan root leaves its gas in the block's
// gas counter, which inflates the CumulativeGasUsed of the next receipt. Since
// storage keeps only the cumulative gas, the GasUsed derived for that receipt
// when it is read back includes the leaked gas. The amendments record, per
// block, the difference between the live GasUsed of each receipt and the one
// derived from storage, so that it can be subtracted after deriving.
//
// Consensus data, including the receipts root, is not affected.
//
// In the underlying table, the following keys are used:
//  - key: <blockNum> -> rlp([]ReceiptGasAmendment)  // only for blocks with amendments

// ReceiptGasAmendment corrects the GasUsed derived from storage for a receipt.
type ReceiptGasAmendment struct {
	// Index is the position of the amended receipt within its block.
	Index uint32
	// GasExcess is the gas wrongly attributed to the receipt, to be subtracted
	// from its derived GasUsed.
	GasExcess uint64
}

// setReceiptGasAmendments stores the gas amendments needed to restore the
// GasUsed of the given receipts of block n, if there are any.
func (s *Store) setReceiptGasAmendments(n idx.Block, receipts types.Receipts) {
	amendments := findReceiptGasAmendments(receipts)
	if len(amendments) == 0 {
		return
	}
	s.Log.Warn("Receipt gas differs from cumulative gas", "block", n, "amendments", amendments)
	s.putReceiptGasAmendments(n, amendments)
}

// putReceiptGasAmendments stores the given gas amendments for block n.
func (s *Store) putReceiptGasAmendments(n idx.Block, amendments []ReceiptGasAmendment) {
	buf, _ := rlp.EncodeToBytes(amendments)
	if err := s.table.ReceiptGasAmendments.Put(n.Bytes(), buf); err != nil {
		s.Log.Crit("Failed to store receipt gas amendments", "err", err)
	}
}

// getReceiptGasAmendments returns the gas amendments stored for block n, or
// nil if there are none.
func (s *Store) getReceiptGasAmendments(n idx.Block) []ReceiptGasAmendment {
	buf, err := s.table.ReceiptGasAmendments.Get(n.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get receipt gas amendments", "err", err)
	}
	if buf == nil {
		return nil
	}
	var amendments []ReceiptGasAmendment
	if err := rlp.DecodeBytes(buf, &amendments); err != nil {
		s.Log.Crit("Failed to decode receipt gas amendments", "err", err)
	}
	return amendments
}

// applyReceiptGasAmendments subtracts the gas amendments stored for block n
// from the GasUsed of the given receipts derived from storage. Amendments
// beyond the receipts are skipped.
func (s *Store) applyReceiptGasAmendments(n idx.Block, receipts types.Receipts) {
	for _, a := range s.getReceiptGasAmendments(n) {
		if int(a.Index) >= len(receipts) {
			continue
		}
		receipts[a.Index].GasUsed -= a.GasExcess
	}
}

// findReceiptGasAmendments returns the amendments for all receipts whose
// GasUsed differs from the one derived from their CumulativeGasUsed. The
// arithmetic wraps, so applying an amendment restores GasUsed exactly.
func findReceiptGasAmendments(receipts types.Receipts) []ReceiptGasAmendment {
	var amendments []ReceiptGasAmendment
	var prev uint64
	for i, r := range receipts {
		if leaked := r.CumulativeGasUsed - prev - r.GasUsed; leaked != 0 {
			amendments = append(amendments, ReceiptGasAmendment{Index: uint32(i), GasExcess: leaked})
		}
		prev = r.CumulativeGasUsed
	}
	return amendments
}
