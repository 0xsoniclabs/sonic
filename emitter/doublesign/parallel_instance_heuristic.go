package doublesign

import "time"

// DetectParallelInstance should be called after downloading a self-event which wasn't created on this instance
// Returns true if a parallel instance is likely be running
func DetectParallelInstance(s SyncStatus, threshold time.Duration) bool {
	if s.ExternalSelfEventCreated.Before(s.Startup) {
		return false
	}
	return s.Since(s.ExternalSelfEventCreated) < threshold
}
