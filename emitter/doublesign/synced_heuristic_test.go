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

package doublesign

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func makeBadConnections(s SyncStatus) SyncStatus {
	s.PeersNum = 0
	return s
}

func makeNoP2PSynced(s SyncStatus) SyncStatus {
	s.P2PSynced = time.Time{}
	return s
}

func makeBadExternalSelfEventDetected(s SyncStatus, period time.Duration) SyncStatus {
	s.ExternalSelfEventDetected = s.Now.Add(-period)
	return s
}

func makeBadExternalSelfEventCreated(s SyncStatus, period time.Duration) SyncStatus {
	s.ExternalSelfEventCreated = s.Now.Add(-period)
	return s
}

func makeBadBecameValidator(s SyncStatus, period time.Duration) SyncStatus {
	s.BecameValidator = s.Now.Add(-period)
	return s
}

func makeBadLastConnected(s SyncStatus, period time.Duration) SyncStatus {
	s.LastConnected = s.Now.Add(-period)
	return s
}

func makeBadP2PSynced(s SyncStatus, period time.Duration) SyncStatus {
	s.P2PSynced = s.Now.Add(-period)
	return s
}

func TestSyncedToEmit(t *testing.T) {
	{
		now := time.Time{}.Add(10)
		s := SyncStatus{
			PeersNum:                  1,
			Now:                       now,
			P2PSynced:                 now.Add(-9),
			Startup:                   now.Add(-9),
			LastConnected:             now.Add(-9),
			BecameValidator:           now.Add(-9),
			ExternalSelfEventCreated:  now.Add(-9),
			ExternalSelfEventDetected: now.Add(-9),
		}
		wait, err := SyncedToEmit(s, 9)
		require.Zero(t, wait)
		require.NoError(t, err)

		// test no connections
		wait, err = SyncedToEmit(makeBadConnections(s), 10)
		require.Zero(t, wait)
		require.EqualError(t, err, ErrNoConnections.Error())

		// test not synced
		wait, err = SyncedToEmit(makeNoP2PSynced(s), 10)
		require.Zero(t, wait)
		require.EqualError(t, err, ErrP2PSyncOngoing.Error())

		// test ErrSelfEventsOngoing
		wait, err = SyncedToEmit(makeBadExternalSelfEventCreated(s, 0), 2)
		require.Equal(t, time.Duration(2), wait)
		require.EqualError(t, err, ErrSelfEventsOngoing.Error())

		wait, err = SyncedToEmit(makeBadExternalSelfEventCreated(s, 1), 2)
		require.Equal(t, time.Duration(1), wait)
		require.EqualError(t, err, ErrSelfEventsOngoing.Error())

		wait, err = SyncedToEmit(makeBadExternalSelfEventCreated(s, 2), 2)
		require.Zero(t, wait)
		require.NoError(t, err)

		wait, err = SyncedToEmit(makeBadExternalSelfEventDetected(s, 0), 2)
		require.Equal(t, time.Duration(2), wait)
		require.EqualError(t, err, ErrSelfEventsOngoing.Error())

		// test ErrJustBecameValidator
		wait, err = SyncedToEmit(makeBadBecameValidator(s, 1), 2)
		require.Equal(t, time.Duration(1), wait)
		require.EqualError(t, err, ErrJustBecameValidator.Error())

		// test ErrJustConnected
		wait, err = SyncedToEmit(makeBadLastConnected(s, 1), 2)
		require.Equal(t, time.Duration(1), wait)
		require.EqualError(t, err, ErrJustConnected.Error())

		// test ErrJustP2PSynced
		wait, err = SyncedToEmit(makeBadP2PSynced(s, 1), 2)
		require.Equal(t, time.Duration(1), wait)
		require.EqualError(t, err, ErrJustP2PSynced.Error())

		// test ErrJustP2PSynced and ErrNoConnections
		wait, err = SyncedToEmit(makeBadConnections(makeBadP2PSynced(s, 1)), 2)
		require.Zero(t, wait)
		require.EqualError(t, err, ErrNoConnections.Error())

		// test ErrJustP2PSynced and ErrJustBecameValidator
		wait, err = SyncedToEmit(makeBadBecameValidator(makeBadP2PSynced(s, 1), 0), 2)
		require.Equal(t, time.Duration(2), wait)
		require.EqualError(t, err, ErrJustBecameValidator.Error())

		// test ErrJustP2PSynced and ErrJustBecameValidator
		wait, err = SyncedToEmit(makeBadBecameValidator(makeBadP2PSynced(s, 0), 1), 2)
		require.Equal(t, time.Duration(2), wait)
		require.EqualError(t, err, ErrJustP2PSynced.Error())
	}
}
