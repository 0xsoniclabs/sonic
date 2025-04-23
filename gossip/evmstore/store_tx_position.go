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
	Block idx.Block
	// Deprecated: in the past, transactions have been stored as part of the
	// events that have introduced them to the DAG. When performing a lookup
	// operation for a transaction on the RPC, this index was queried to locate
	// the event containing the actual transaction.
	//
	// Nowadays, all transactions are stored independently in a separate table
	// within the gossip database. Thus, this index is no longer needed.
	// Additionally, with the introduction of the Single-Proposer protocol,
	// the event payload has been altered, leading to a change in the location
	// of transactions within the events. Finally, long-term, events shall be
	// pruned, while transactions are expected to be kept indefinitely.
	//
	// Thus, this field should no longer be used. However, the field is kept
	// for backward compatibility of the RLP encoding of this type.
	Event hash.Event
	// Deprecated: for the same reason as the Event field above.
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
