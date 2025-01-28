package emitter

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestGetEmitterIntervalLimit_IsOffWhenIntervalIsZero(t *testing.T) {
	ms := time.Microsecond
	rules := opera.EmitterRules{
		Interval:       0,
		StallThreshold: inter.Timestamp(200 * ms),
	}
	for _, delay := range []time.Duration{0, 100 * ms, 199 * ms, 200 * ms, 201 * ms} {
		_, enabled := getEmitterIntervalLimit(rules, delay)
		if enabled {
			t.Fatal("should be disabled")
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
		got, enabled := getEmitterIntervalLimit(rules, delay)
		if !enabled {
			t.Fatalf("should be enabled for delay %v", delay)
		}
		want := regular
		if delay > stallThreshold {
			want = stalled
		}
		if want != got {
			t.Errorf("for delay %v, want %v, got %v", delay, want, got)
		}
	}
}
