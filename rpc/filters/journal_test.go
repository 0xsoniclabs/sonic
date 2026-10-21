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

	"github.com/stretchr/testify/require"
)

func TestJournal_SinceReturnsTheEventsFromAPosition(t *testing.T) {
	j := newJournal[int](100)
	require.Zero(t, j.end())
	j.append(1, 2, 3)
	require.EqualValues(t, 3, j.end())

	events, kept := j.since(1)
	require.True(t, kept)
	require.Equal(t, []int{2, 3}, events)
	events, kept = j.since(3)
	require.True(t, kept)
	require.Empty(t, events)
}

func TestJournal_BeyondTheLimitTheOldestQuarterIsDropped(t *testing.T) {
	j := newJournal[int](8)
	for i := range 9 {
		j.append(i)
	}
	// 9 events over a limit of 8: 6 are kept.
	require.EqualValues(t, 3, j.first)
	require.Equal(t, []int{3, 4, 5, 6, 7, 8}, j.items)
	_, kept := j.since(2)
	require.False(t, kept)
	events, kept := j.since(3)
	require.True(t, kept)
	require.Len(t, events, 6)
}

func TestJournal_TrimDropsTheEventsBeforeAPosition(t *testing.T) {
	j := newJournal[int](100)
	j.append(0, 1, 2, 3)
	j.trim(2)
	require.EqualValues(t, 2, j.first)
	require.Equal(t, []int{2, 3}, j.items)
	j.trim(1) // already dropped
	require.EqualValues(t, 2, j.first)
	j.trim(10) // past the end
	require.EqualValues(t, 4, j.first)
	require.Empty(t, j.items)
	require.EqualValues(t, 4, j.end())
}

func TestJournal_AResultIsNeverWrittenAgain(t *testing.T) {
	j := newJournal[int](8)
	j.append(0, 1, 2)
	events, _ := j.since(0)
	j.append(3, 4, 5, 6, 7, 8, 9) // over the limit: drops and copies
	j.trim(8)
	j.append(10)
	require.Equal(t, []int{0, 1, 2}, events)
	// Appending to a result does not write into the journal.
	events, _ = j.since(8)
	_ = append(events, 99)
	more, _ := j.since(8)
	require.Equal(t, []int{8, 9, 10}, more)
}
