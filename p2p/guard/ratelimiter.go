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

// RateLimitConfig configures the per-peer token-bucket limits.
type RateLimitConfig struct {
	// BytesPerSecond is the sustained inbound byte rate allowed per peer.
	BytesPerSecond int64
	// ByteBurst is the maximum burst size, in bytes, allowed per peer.
	ByteBurst int64
	// MessagesPerSecond is the sustained inbound message rate allowed per peer.
	MessagesPerSecond float64
	// MessageBurst is the maximum burst of messages allowed per peer.
	MessageBurst int
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

// AllowMessage reports whether a message of the given size from the identified
// peer is within both the byte and message rate limits. A false result means
// the peer has exceeded a limit and the caller should reject the message and,
// on sustained abuse, disconnect the peer.
func (r *RateLimiter) AllowMessage(peer string, size int) bool {
	buckets := r.bucketsFor(peer)
	if !buckets.messages.Allow() {
		return false
	}
	return buckets.bytes.AllowN(nowFunc(), size)
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
		bytes:    rate.NewLimiter(rate.Limit(r.config.BytesPerSecond), int(r.config.ByteBurst)),
		messages: rate.NewLimiter(rate.Limit(r.config.MessagesPerSecond), r.config.MessageBurst),
	}
	r.buckets[peer] = b
	return b
}

// peerBuckets holds the two token buckets tracked for a single peer.
type peerBuckets struct {
	bytes    *rate.Limiter
	messages *rate.Limiter
}
