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

package autocompact

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// errStore wraps a kvdb.Store so that Compact always returns a fixed error.
type errStore struct {
	kvdb.Store
	compactErr error
}

func (e *errStore) Compact(start []byte, limit []byte) error { return e.compactErr }

func TestMayCompact_LogsCompactError(t *testing.T) {
	buf := &bytes.Buffer{}
	orig := log.Root()
	log.SetDefault(log.NewLogger(log.NewTerminalHandler(buf, false)))
	t.Cleanup(func() { log.SetDefault(orig) })

	inner := &errStore{Store: memorydb.New(), compactErr: errors.New("boom-compact")}
	// limit=0 → any write triggers compaction immediately.
	s := Wrap(inner, 0, NewBackwardsCont, "test-db")

	require.NoError(t, s.Put([]byte("key"), []byte("value")))

	out := buf.String()
	require.Contains(t, out, "Autocompact range failed")
	require.Contains(t, out, "test-db")
	require.Contains(t, out, "boom-compact")
}
