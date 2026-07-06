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

// Package guard provides the adversarial-network protections for the P2P layer:
// per-peer traffic rate limiting, connection gating, resource limits, and
// gossipsub peer scoring.
package guard

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// nowFunc is the time source used by the byte bucket. It is a package variable
// so tests can advance time deterministically.
var nowFunc = time.Now

// RateLimitConfig configures the per-peer token-bucket limits and the
// sustained-abuse policy.
type RateLimitConfig struct {
	// BytesPerSecond is the sustained inbound byte rate allowed per peer.
	BytesPerSecond int64
	// ByteBurst is the maximum burst size, in bytes, allowed per peer.
	ByteBurst int64
	// MessagesPerSecond is the sustained inbound message rate allowed per peer.
	MessagesPerSecond float64
	// MessageBurst is the maximum burst of messages allowed per peer.
	MessageBurst int
	// ViolationsPerSecond is the sustained rate of rate-limit violations
	// tolerated per peer before it is considered abusive.
	ViolationsPerSecond float64
	// ViolationBurst is the number of violations tolerated in a burst before a
	// peer is considered abusive. A short burst of violations is tolerated; a
	// sustained stream of them exhausts the allowance and flags abuse.
	ViolationBurst int
	// BanDuration is how long an abusive peer is banned after being
	// disconnected, before it may reconnect.
	BanDuration time.Duration
}

// Decision is the outcome of checking a message against the per-peer limits.
type Decision struct {
	// Allowed reports whether the message is within the traffic budget.
	Allowed bool
	// Abusive reports whether the peer has sustained enough violations to be
	// disconnected and temporarily banned.
	Abusive bool
}

// RateLimiter enforces per-peer inbound traffic limits on both bytes and
// message counts. It is safe for concurrent use. A peer is identified by an
// opaque string key (typically its libp2p peer ID).
type RateLimiter struct {
	config  RateLimitConfig
	mutex   sync.Mutex
	buckets map[string]*peerBuckets
}

// NewRateLimiter creates a RateLimiter enforcing the given configuration.
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:  config,
		buckets: make(map[string]*peerBuckets),
	}
}

// Check evaluates a message of the given size from the identified peer against
// the per-peer limits. Decision.Allowed reports whether the message is within
// the traffic budget; Decision.Abusive reports whether the peer has committed
// enough violations in a short window to warrant disconnection. Each violation
// consumes a token from the peer's violation bucket, so isolated breaches are
// tolerated while a sustained flood exhausts the allowance.
func (r *RateLimiter) Check(peer string, size int) Decision {
	buckets := r.bucketsFor(peer)
	if buckets.messages.Allow() && buckets.bytes.AllowN(nowFunc(), size) {
		return Decision{Allowed: true}
	}
	if !buckets.violations.Allow() {
		return Decision{Allowed: false, Abusive: true}
	}
	return Decision{Allowed: false}
}

// AllowMessage reports whether a message of the given size from the identified
// peer is within the traffic budget. It is a convenience wrapper over Check for
// callers that do not act on sustained abuse.
func (r *RateLimiter) AllowMessage(peer string, size int) bool {
	return r.Check(peer, size).Allowed
}

// Forget drops any accounting kept for the peer. It should be called when a
// peer disconnects to bound memory use.
func (r *RateLimiter) Forget(peer string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.buckets, peer)
}

func (r *RateLimiter) bucketsFor(peer string) *peerBuckets {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if b, ok := r.buckets[peer]; ok {
		return b
	}
	b := &peerBuckets{
		bytes:      rate.NewLimiter(rate.Limit(r.config.BytesPerSecond), int(r.config.ByteBurst)),
		messages:   rate.NewLimiter(rate.Limit(r.config.MessagesPerSecond), r.config.MessageBurst),
		violations: rate.NewLimiter(rate.Limit(r.config.ViolationsPerSecond), r.config.ViolationBurst),
	}
	r.buckets[peer] = b
	return b
}

// peerBuckets holds the token buckets tracked for a single peer: inbound bytes,
// inbound messages, and rate-limit violations (for sustained-abuse detection).
type peerBuckets struct {
	bytes      *rate.Limiter
	messages   *rate.Limiter
	violations *rate.Limiter
}
