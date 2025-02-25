package doublesign

import (
	"errors"
	"time"
)

var (
	ErrNoConnections       = errors.New("no connections")
	ErrP2PSyncOngoing      = errors.New("P2P synchronization isn't finished")
	ErrSelfEventsOngoing   = errors.New("not downloaded all the self-events")
	ErrJustBecameValidator = errors.New("just joined the validators group")
	ErrJustConnected       = errors.New("recently connected")
	ErrJustP2PSynced       = errors.New("waiting additional time")
)

type SyncStatus struct {
	PeersNum                  int
	Now                       time.Time
	Startup                   time.Time
	LastConnected             time.Time
	P2PSynced                 time.Time
	BecameValidator           time.Time
	ExternalSelfEventCreated  time.Time
	ExternalSelfEventDetected time.Time
}

func (s *SyncStatus) Since(t time.Time) time.Duration {
	return s.Now.Sub(t)
}

type maxWaitError struct {
	wait    time.Duration
	waitErr error
}

func (m *maxWaitError) apply(wait time.Duration, waitErr error) {
	if m.wait < wait {
		m.wait = wait
		m.waitErr = waitErr
	}
}

// SyncedToEmit should be called before emitting any events
// It returns nil if node is allowed to emit events
// Otherwise, node returns a minimum duration of how long node should wait before emitting
func SyncedToEmit(s SyncStatus, threshold time.Duration) (time.Duration, error) {
	if s.PeersNum == 0 {
		return 0, ErrNoConnections
	}
	if s.P2PSynced.IsZero() {
		return 0, ErrP2PSyncOngoing
	}
	var max maxWaitError
	if s.Since(s.ExternalSelfEventDetected) < threshold {
		max.apply(threshold-s.Since(s.ExternalSelfEventDetected), ErrSelfEventsOngoing)
	}
	if s.Since(s.ExternalSelfEventCreated) < threshold {
		max.apply(threshold-s.Since(s.ExternalSelfEventCreated), ErrSelfEventsOngoing)
	}
	if s.Since(s.BecameValidator) < threshold {
		max.apply(threshold-s.Since(s.BecameValidator), ErrJustBecameValidator)
	}
	if s.Since(s.LastConnected) < threshold {
		max.apply(threshold-s.Since(s.LastConnected), ErrJustConnected)
	}
	if s.Since(s.P2PSynced) < threshold {
		max.apply(threshold-s.Since(s.P2PSynced), ErrJustP2PSynced)
	}

	return max.wait, max.waitErr
}
