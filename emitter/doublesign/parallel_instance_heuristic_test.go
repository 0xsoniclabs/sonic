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
