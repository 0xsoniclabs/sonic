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

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestPolledFilters_KeepAnEventOnceForAllFilters holds I17.
func TestPolledFilters_KeepAnEventOnceForAllFilters(t *testing.T) {
	polled := newPolledFilters(time.Minute, 16, uncounted{})
	identifiers := make([]string, 1000)
	for index := range identifiers {
		identifier, err := polled.install(&filter{kind: blocksKind})
		require.NoError(t, err)
		identifiers[index] = identifier
	}
	polled.hashes.append(common.Hash{1})
	require.Len(t, polled.hashes.events, 1, "one copy for a thousand filters")
	for _, identifier := range identifiers {
		taken, err := polled.poll(identifier)
		require.NoError(t, err)
		require.Equal(t, []common.Hash{{1}}, taken.hashes)
	}
	polled.sweep(time.Now())
	require.Empty(t, polled.hashes.events, "the journal keeps nothing every filter polled")
}

// TestPolledFilters_RemoveAFilterThatFellBehind holds I17: a filter whose
// events left the journal is gone, one that polled gets every event.
func TestPolledFilters_RemoveAFilterThatFellBehind(t *testing.T) {
	polled := newPolledFilters(time.Minute, 4, uncounted{})
	behind, err := polled.install(&filter{kind: blocksKind})
	require.NoError(t, err)
	current, err := polled.install(&filter{kind: blocksKind})
	require.NoError(t, err)
	var seen []common.Hash
	for number := range 10 {
		polled.hashes.append(common.Hash{byte(number)})
		taken, err := polled.poll(current)
		require.NoError(t, err)
		seen = append(seen, taken.hashes...)
	}
	require.Len(t, seen, 10)
	_, err = polled.poll(behind)
	require.ErrorIs(t, err, errFilterNotFound)
	_, found := polled.find(behind)
	require.False(t, found)
}

func TestPolledFilters_TrimToWhatALiveFilterHasNotPolled(t *testing.T) {
	polled := newPolledFilters(time.Minute, 100, uncounted{})
	identifier, err := polled.install(&filter{kind: blocksKind})
	require.NoError(t, err)
	polled.hashes.append(common.Hash{1}, common.Hash{2})
	_, err = polled.poll(identifier)
	require.NoError(t, err)
	polled.hashes.append(common.Hash{3})
	polled.sweep(time.Now())
	require.Equal(t, []common.Hash{{3}}, polled.hashes.events)

	polled.sweep(time.Now().Add(2 * time.Minute))
	_, found := polled.find(identifier)
	require.False(t, found, "an expired filter is removed")
	require.Empty(t, polled.hashes.events, "and with it what it had not polled")
	require.False(t, polled.has(blocksKind))
}

func TestPolledFilters_AreBounded(t *testing.T) {
	polled := newPolledFilters(time.Minute, 1, uncounted{})
	for range maxFilters {
		_, err := polled.install(&filter{kind: logsKind})
		require.NoError(t, err)
	}
	_, err := polled.install(&filter{kind: logsKind})
	require.EqualError(t, err, "too many filters, the limit is 100000")
	polled.dropKinds(logsKind)
	_, err = polled.install(&filter{kind: logsKind})
	require.NoError(t, err)
}

func TestJournal_DropsTheOldestAboveItsLimit(t *testing.T) {
	events := newJournal[int](3)
	events.append(1, 2, 3, 4, 5)
	found, end, lost := events.since(2)
	require.Equal(t, []int{3, 4, 5}, found)
	require.Equal(t, uint64(5), end)
	require.False(t, lost)
	_, _, lost = events.since(1)
	require.True(t, lost)
	events.trim(10)
	require.Equal(t, uint64(5), events.end(), "trimming keeps the positions")
	require.Empty(t, events.events)
}

// uncounted is a counter that counts nothing.
type uncounted struct{}

func (uncounted) CountFilters(string, int)        {}
func (uncounted) CountEvents(string, string, int) {}
