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

func TestFailureLimiter_BurstOfFailures_Tolerated(t *testing.T) {
	limiter := NewFailureLimiter(FailureLimitConfig{FailuresPerSecond: 0, FailureBurst: 3})
	for i := 0; i < 3; i++ {
		require.False(t, limiter.Record("peer-a"), "failure %d within the burst should be tolerated", i)
	}
}

func TestFailureLimiter_SustainedFailures_FlaggedForBan(t *testing.T) {
	limiter := NewFailureLimiter(FailureLimitConfig{FailuresPerSecond: 0, FailureBurst: 3})
	banned := false
	for i := 0; i < 10 && !banned; i++ {
		banned = limiter.Record("peer-a")
	}
	require.True(t, banned, "expected sustained failures to be flagged for a ban")
}

func TestFailureLimiter_SeparatePeers_TrackedIndependently(t *testing.T) {
	limiter := NewFailureLimiter(FailureLimitConfig{FailuresPerSecond: 0, FailureBurst: 1})
	require.False(t, limiter.Record("peer-a"), "first failure from peer-a should be tolerated")
	require.False(t, limiter.Record("peer-b"), "first failure from peer-b should be tolerated independently of peer-a")
}

func TestFailureLimiter_Forget_ResetsPeer(t *testing.T) {
	limiter := NewFailureLimiter(FailureLimitConfig{FailuresPerSecond: 0, FailureBurst: 1})
	limiter.Record("peer-a") // consume the burst
	require.True(t, limiter.Record("peer-a"), "expected peer-a to be flagged after exhausting its burst")
	limiter.Forget("peer-a")
	require.False(t, limiter.Record("peer-a"), "expected a forgotten peer to start with a fresh burst")
}
