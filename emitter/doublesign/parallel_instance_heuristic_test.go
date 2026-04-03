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

func TestDetectParallelInstance(t *testing.T) {
	{
		now := time.Now()
		s := SyncStatus{
			Now:                      now,
			Startup:                  now.Add(-2 * time.Hour),
			ExternalSelfEventCreated: now.Add(-1 * time.Hour),
		}
		require.False(t, DetectParallelInstance(s, 0*time.Hour))
		require.False(t, DetectParallelInstance(s, 1*time.Hour))
		require.True(t, DetectParallelInstance(s, 1*time.Hour+1))
		require.True(t, DetectParallelInstance(s, 2*time.Hour))
		s.Startup = now.Add(-1 * time.Hour)
		require.True(t, DetectParallelInstance(s, 1*time.Hour+1))
		require.True(t, DetectParallelInstance(s, 2*time.Hour))
		s.Startup = now.Add(-1*time.Hour + 1)
		require.False(t, DetectParallelInstance(s, 1*time.Hour+1))
		require.False(t, DetectParallelInstance(s, 2*time.Hour))
	}
	{
		now := time.Now()
		s := SyncStatus{
			Now:                       now,
			Startup:                   now.Add(-2 * time.Hour),
			ExternalSelfEventDetected: now.Add(-1 * time.Hour),
		}
		require.False(t, DetectParallelInstance(s, 0*time.Hour))
		require.False(t, DetectParallelInstance(s, 1*time.Hour))
		require.False(t, DetectParallelInstance(s, 1*time.Hour+1))
		require.False(t, DetectParallelInstance(s, 2*time.Hour))
	}
}
