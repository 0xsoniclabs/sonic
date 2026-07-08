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

package guard

import (
	"sync"

	"golang.org/x/time/rate"
)

// FailureLimitConfig configures the per-peer tolerance for repeated failures
// (e.g. failed authentication handshakes) before a peer is considered a flooder.
type FailureLimitConfig struct {
	// FailuresPerSecond is the sustained rate of failures tolerated per peer.
	FailuresPerSecond float64
	// FailureBurst is the number of failures tolerated in a burst before a peer
	// is flagged for banning. A short burst absorbs transient conditions (e.g.
	// epoch-boundary membership skew); a sustained stream trips the limit.
	FailureBurst int
}

// FailureLimiter meters repeated per-peer failures with a token bucket, so
// isolated failures are tolerated while a sustained stream is flagged for a ban.
// It is safe for concurrent use.
type FailureLimiter struct {
	config  FailureLimitConfig
	mutex   sync.Mutex
	buckets map[string]*rate.Limiter
}

// NewFailureLimiter creates a FailureLimiter enforcing the given configuration.
func NewFailureLimiter(config FailureLimitConfig) *FailureLimiter {
	return &FailureLimiter{
		config:  config,
		buckets: make(map[string]*rate.Limiter),
	}
}

// Record notes a failure by the identified peer and reports whether the peer has
// exceeded its tolerated burst and should therefore be banned.
func (f *FailureLimiter) Record(peer string) (banNow bool) {
	return !f.bucketFor(peer).Allow()
}

// Forget drops any accounting kept for the peer. It should be called when a peer
// disconnects or recovers, to bound memory use and let a recovered peer start
// clean.
func (f *FailureLimiter) Forget(peer string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.buckets, peer)
}

func (f *FailureLimiter) bucketFor(peer string) *rate.Limiter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if bucket, ok := f.buckets[peer]; ok {
		return bucket
	}
	bucket := rate.NewLimiter(rate.Limit(f.config.FailuresPerSecond), f.config.FailureBurst)
	f.buckets[peer] = bucket
	return bucket
}
