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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_WithinBudget_Allows(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:    1 << 20,
		ByteBurst:         1 << 20,
		MessagesPerSecond: 100,
		MessageBurst:      100,
	})
	for i := 0; i < 100; i++ {
		require.True(t, limiter.AllowMessage("peer-a", 64), "compliant message %d was rejected", i)
	}
}

func TestRateLimiter_ExceedsMessageRate_Rejects(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:    1 << 30,
		ByteBurst:         1 << 30,
		MessagesPerSecond: 1,
		MessageBurst:      5,
	})
	allowed := 0
	for i := 0; i < 100; i++ {
		if limiter.AllowMessage("peer-a", 1) {
			allowed++
		}
	}
	require.LessOrEqual(t, allowed, 6, "expected message burst to cap allowance near 5")
}

func TestRateLimiter_ExceedsByteRate_Rejects(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:    1000,
		ByteBurst:         1000,
		MessagesPerSecond: 1 << 20,
		MessageBurst:      1 << 20,
	})
	allowed := 0
	for i := 0; i < 100; i++ {
		if limiter.AllowMessage("peer-a", 100) {
			allowed++
		}
	}
	require.LessOrEqual(t, allowed, 11, "expected byte burst to cap allowance near 10")
}

func TestRateLimiter_TransientViolations_NotFlaggedAbusive(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:      1 << 30,
		ByteBurst:           1 << 30,
		MessagesPerSecond:   1,
		MessageBurst:        1,
		ViolationsPerSecond: 0, // no refill, so ViolationBurst is the exact tolerance
		ViolationBurst:      5,
	})
	// Use up the single-message burst, then commit exactly ViolationBurst
	// violations - all should be tolerated.
	limiter.Check("peer-a", 1)
	for i := 0; i < 5; i++ {
		require.False(t, limiter.Check("peer-a", 1).Abusive, "violation %d within the burst was flagged abusive", i)
	}
}

func TestRateLimiter_SustainedViolations_FlaggedAbusive(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:      1 << 30,
		ByteBurst:           1 << 30,
		MessagesPerSecond:   1,
		MessageBurst:        1,
		ViolationsPerSecond: 0,
		ViolationBurst:      3,
	})
	limiter.Check("peer-a", 1) // consume the message burst
	abusive := false
	for i := 0; i < 20 && !abusive; i++ {
		abusive = limiter.Check("peer-a", 1).Abusive
	}
	require.True(t, abusive, "expected sustained violations to be flagged abusive")
}

func TestRateLimiter_Abuse_TrackedPerPeer(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:      1 << 30,
		ByteBurst:           1 << 30,
		MessagesPerSecond:   1,
		MessageBurst:        1,
		ViolationsPerSecond: 0,
		ViolationBurst:      2,
	})
	// Drive peer-a into abuse.
	limiter.Check("peer-a", 1)
	for i := 0; i < 10; i++ {
		limiter.Check("peer-a", 1)
	}
	// peer-b is fresh and must not inherit peer-a's abuse.
	require.False(t, limiter.Check("peer-b", 1).Abusive, "peer-b should not be flagged abusive from peer-a's violations")
}

func TestRateLimiter_SeparatePeers_TrackedIndependently(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		BytesPerSecond:    1 << 30,
		ByteBurst:         1 << 30,
		MessagesPerSecond: 1,
		MessageBurst:      1,
	})
	require.True(t, limiter.AllowMessage("peer-a", 1), "first message from peer-a should be allowed")
	require.True(t, limiter.AllowMessage("peer-b", 1), "first message from peer-b should be allowed independently of peer-a")
}
