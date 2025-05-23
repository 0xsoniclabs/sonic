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
	// Deprecated: this used to be a reference to the event that introduced the
	// transaction which's position this object is representing. It was used to
	// look up the event in RPC calls, when trying to obtain a transaction based
	// on its hash. However, all transactions are now stored explicitly and
	// events are supposed to be pruned eventually. Thus, look-ups should no
	// longer be done based on events. However, the field is kept for backwards
	// compatibility with Store data created by older versions of the code.
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
