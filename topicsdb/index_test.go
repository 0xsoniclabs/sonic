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

package topicsdb

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// captureLog redirects the default logger to a buffer for the test duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	orig := log.Root()
	log.SetDefault(log.NewLogger(log.NewTerminalHandler(buf, false)))
	t.Cleanup(func() { log.SetDefault(orig) })
	return buf
}

// errOnCloseStore wraps a kvdb.Store whose Close always returns a fixed error.
type errOnCloseStore struct {
	kvdb.Store
	closeErr error
}

func (e *errOnCloseStore) Close() error { return e.closeErr }

func TestIndex_Close_LogsTopicAndLogrecErrors(t *testing.T) {
	buf := captureLog(t)

	idx := newIndex(memorydb.New())
	idx.table.Topic = &errOnCloseStore{Store: memorydb.New(), closeErr: errors.New("boom-topic")}
	idx.table.Logrec = &errOnCloseStore{Store: memorydb.New(), closeErr: errors.New("boom-logrec")}

	idx.Close()

	out := buf.String()
	require.Contains(t, out, "Failed to close topic table")
	require.Contains(t, out, "boom-topic")
	require.Contains(t, out, "Failed to close logrec table")
	require.Contains(t, out, "boom-logrec")
}

// errOnBatchWriteStore wraps a kvdb.Store whose batches always fail on Write.
type errOnBatchWriteStore struct {
	kvdb.Store
	writeErr error
}

type errBatch struct {
	kvdb.Batch
	writeErr error
}

func (b *errBatch) Write() error { return b.writeErr }

func (s *errOnBatchWriteStore) NewBatch() kvdb.Batch {
	return &errBatch{Batch: s.Store.NewBatch(), writeErr: s.writeErr}
}

func TestIndex_WrapTablesAsBatched_UnwrapLogsFlushErrors(t *testing.T) {
	buf := captureLog(t)

	idx := newIndex(memorydb.New())
	// Replace the tables so batches produced by them fail on Write. The
	// batched wrapper's Flush calls batch.Write(), which surfaces the error.
	idx.table.Topic = &errOnBatchWriteStore{Store: memorydb.New(), writeErr: errors.New("boom-topic")}
	idx.table.Logrec = &errOnBatchWriteStore{Store: memorydb.New(), writeErr: errors.New("boom-logrec")}

	unwrap := idx.WrapTablesAsBatched()

	// Push a record so both batches are non-empty (Flush short-circuits on
	// empty batches in some implementations, but here always calls Write).
	require.NoError(t, idx.Push(&types.Log{
		BlockNumber: 1,
		Address:     randAddress(),
		Topics:      []common.Hash{{}},
	}))

	unwrap()

	out := buf.String()
	require.Contains(t, out, "Failed to flush topic batch")
	require.Contains(t, out, "boom-topic")
	require.Contains(t, out, "Failed to flush logrec batch")
	require.Contains(t, out, "boom-logrec")
}
