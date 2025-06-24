// Copyright 2025 Sonic Operations Ltd
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

package emitter

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestGetEmitterIntervalLimit_ZeroIsAValidInterval(t *testing.T) {
	ms := time.Microsecond
	rules := opera.EmitterRules{
		Interval:       0,
		StallThreshold: inter.Timestamp(200 * ms),
	}
	for _, delay := range []time.Duration{0, 100 * ms, 199 * ms, 200 * ms, 201 * ms} {
		interval := getEmitterIntervalLimit(rules, delay)
		if interval != 0 {
			t.Fatal("should be zero")
		}
	}
}

func TestGetEmitterIntervalLimit_SwitchesToStallIfDelayed(t *testing.T) {
	ms := time.Millisecond
	regular := 100 * ms
	stallThreshold := 200 * ms
	stalled := 300 * ms

	rules := opera.EmitterRules{
		Interval:        inter.Timestamp(regular),
		StallThreshold:  inter.Timestamp(stallThreshold),
		StalledInterval: inter.Timestamp(stalled),
	}

	for _, delay := range []time.Duration{0, 100 * ms, 199 * ms, 200 * ms, 201 * ms, 60 * time.Minute} {
		got := getEmitterIntervalLimit(rules, delay)
		want := regular
		if delay > stallThreshold {
			want = stalled
		}
		if want != got {
			t.Errorf("for delay %v, want %v, got %v", delay, want, got)
		}
	}
}
