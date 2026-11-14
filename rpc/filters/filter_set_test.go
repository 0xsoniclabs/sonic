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

package filters

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func installed(t *testing.T, set *filterSet, kind kind) dispatch.ID {
	t.Helper()
	id, err := set.install(&filter{kind: kind})
	require.NoError(t, err)
	return id
}

// An event lives once in its journal for every filter; a filter polls the
// events from its install on, and the journal keeps only what a live filter
// has not polled.
func TestFilterSet_SharesOneJournalAmongTheFilters(t *testing.T) {
	set := newFilterSet(time.Minute)
	set.addBlock(common.Hash{0x00}, nil) // before any filter
	ids := make([]dispatch.ID, 1000)
	for i := range ids {
		ids[i] = installed(t, set, blocksKind)
	}
	set.addBlock(common.Hash{0x01}, []*types.Log{{}})
	require.Len(t, set.blocks.events, 1)
	require.Empty(t, set.logs.events, "no filter of logs keeps them")

	for _, id := range ids {
		found, events, err := set.poll(id)
		require.NoError(t, err)
		require.Equal(t, blocksKind, found.kind)
		require.Equal(t, []common.Hash{{0x01}}, events.blocks)
	}
	require.Empty(t, set.blocks.events, "every filter polled the event")
	_, events, err := set.poll(ids[0])
	require.NoError(t, err)
	require.Empty(t, events.blocks)
}

func TestFilterSet_BoundsTheFilters(t *testing.T) {
	set := newFilterSet(time.Minute)
	for range maxFilters {
		installed(t, set, transactionsKind)
	}
	_, err := set.install(&filter{kind: blocksKind})
	require.EqualError(t, err, "too many filters, the limit is 100000")
}

// A filter whose events the journal drops at its limit is removed; one that
// polled gets every event.
func TestFilterSet_RemovesAFilterThatFellBehind(t *testing.T) {
	set := newFilterSet(time.Minute)
	behind, polling := installed(t, set, transactionsKind), installed(t, set, transactionsKind)
	batch := make([]*types.Transaction, journalLimit/2)
	set.addTransactions(batch)
	_, events, err := set.poll(polling)
	require.NoError(t, err)
	require.Len(t, events.transactions, journalLimit/2)
	set.addTransactions(batch)
	set.addTransactions(batch[:1])
	_, _, err = set.poll(behind)
	require.ErrorIs(t, err, errFilterNotFound)
	_, events, err = set.poll(polling)
	require.NoError(t, err)
	require.Len(t, events.transactions, journalLimit/2+1)
	require.Empty(t, set.transactions.events)
}

func TestFilterSet_ExpiresFiltersNotPolledInTime(t *testing.T) {
	set := newFilterSet(time.Minute)
	now := time.Unix(1000, 0)
	set.now = func() time.Time { return now }
	idle, polled := installed(t, set, blocksKind), installed(t, set, blocksKind)
	set.addBlock(common.Hash{0x01}, nil)
	now = now.Add(50 * time.Second)
	_, _, err := set.poll(polled)
	require.NoError(t, err)
	now = now.Add(20 * time.Second)
	set.expire()
	_, err = set.get(idle)
	require.ErrorIs(t, err, errFilterNotFound)
	_, err = set.get(polled)
	require.NoError(t, err)
	require.Empty(t, set.blocks.events, "the expired filter's events are dropped")
}

func TestFilterSet_DropsTheFiltersOfAKind(t *testing.T) {
	set := newFilterSet(time.Minute)
	blocks, logs, transactions := installed(t, set, blocksKind), installed(t, set, logsKind), installed(t, set, transactionsKind)
	set.addBlock(common.Hash{0x01}, []*types.Log{{}})
	set.drop(blocksKind, logsKind)
	for _, id := range []dispatch.ID{blocks, logs} {
		_, err := set.get(id)
		require.ErrorIs(t, err, errFilterNotFound)
	}
	require.Empty(t, set.blocks.events)
	require.Empty(t, set.logs.events)
	_, err := set.get(transactions)
	require.NoError(t, err)
	require.True(t, set.uninstall(transactions))
	require.False(t, set.uninstall(transactions))
}
