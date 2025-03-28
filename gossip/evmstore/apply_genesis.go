package evmstore

import (
	"github.com/0xsoniclabs/kvdb/batched"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/0xsoniclabs/sonic/utils/dbutil/autocompact"
)

func (s *Store) WrapTablesAsBatched() (unwrap func()) {
	origTables := s.table

	batchedTxs := batched.Wrap(s.table.Txs)
	s.table.Txs = batchedTxs

	batchedTxPositions := batched.Wrap(s.table.TxPositions)
	s.table.TxPositions = batchedTxPositions

	unwrapLogs := s.EvmLogs.WrapTablesAsBatched()

	batchedReceipts := batched.Wrap(autocompact.Wrap2M(s.table.Receipts, opt.GiB, 16*opt.GiB, false, "receipts"))
	s.table.Receipts = batchedReceipts
	return func() {
		_ = batchedTxs.Flush()
		_ = batchedTxPositions.Flush()
		_ = batchedReceipts.Flush()
		unwrapLogs()
		s.table = origTables
	}
}
