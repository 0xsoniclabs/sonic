package logger

import (
	"testing"
	"time"
)

func TestSetTestMode(t *testing.T) {
	// Should not panic.
	SetTestMode(t)

	// After SetTestMode, logging should go through test output.
	inst := New("test")
	inst.Log.Info("test message from SetTestMode")
}

func TestNew_MultipleNames(t *testing.T) {
	// Only first name is used.
	inst := New("first", "second")
	if inst.Log == nil {
		t.Fatal("expected non-nil Log")
	}
}

func TestPeriodic_ActualRateLimiting(t *testing.T) {
	SetTestMode(t)
	p := &Periodic{Instance: New("test")}

	// First call with zero period should always log.
	p.Info(0, "always log")

	// Set prevLogTime to now; next call with long period should be suppressed.
	p.prevLogTime = time.Now()
	initialTime := p.prevLogTime

	// With a 1-hour period, this should NOT update prevLogTime.
	p.Info(time.Hour, "suppressed")
	if p.prevLogTime != initialTime {
		t.Error("expected prevLogTime to remain unchanged for suppressed log")
	}

	// With zero period, this should update prevLogTime.
	p.Info(0, "not suppressed")
	if p.prevLogTime.Equal(initialTime) {
		t.Error("expected prevLogTime to be updated for non-suppressed log")
	}
}

func TestPeriodic_Warn_RateLimiting(t *testing.T) {
	SetTestMode(t)
	p := &Periodic{Instance: New("test")}

	p.Warn(0, "first warn")
	p.prevLogTime = time.Now()
	initialTime := p.prevLogTime

	p.Warn(time.Hour, "suppressed warn")
	if p.prevLogTime != initialTime {
		t.Error("expected prevLogTime unchanged for suppressed warn")
	}
}

func TestPeriodic_Error_RateLimiting(t *testing.T) {
	SetTestMode(t)
	p := &Periodic{Instance: New("test")}

	p.Error(0, "first error")
	p.prevLogTime = time.Now()
	initialTime := p.prevLogTime

	p.Error(time.Hour, "suppressed error")
	if p.prevLogTime != initialTime {
		t.Error("expected prevLogTime unchanged for suppressed error")
	}
}

func TestPeriodic_Debug_RateLimiting(t *testing.T) {
	SetTestMode(t)
	p := &Periodic{Instance: New("test")}

	p.Debug(0, "first debug")
	p.prevLogTime = time.Now()
	initialTime := p.prevLogTime

	p.Debug(time.Hour, "suppressed debug")
	if p.prevLogTime != initialTime {
		t.Error("expected prevLogTime unchanged for suppressed debug")
	}
}

func TestPeriodic_Trace_RateLimiting(t *testing.T) {
	SetTestMode(t)
	p := &Periodic{Instance: New("test")}

	p.Trace(0, "first trace")
	p.prevLogTime = time.Now()
	initialTime := p.prevLogTime

	p.Trace(time.Hour, "suppressed trace")
	if p.prevLogTime != initialTime {
		t.Error("expected prevLogTime unchanged for suppressed trace")
	}
}

func TestSetLevel_AllValidLevels(t *testing.T) {
	levels := []string{"debug", "dbug", "info", "warn", "error", "eror"}
	for _, l := range levels {
		SetLevel(l)
	}
	SetLevel("info") // restore
}

func TestLogger_Interface(t *testing.T) {
	// Verify Logger interface can be satisfied.
	inst := New("test")
	var _ Logger = inst.Log
}

func TestTestLogHandler_WithAttrs(t *testing.T) {
	SetTestMode(t)
	inst := New("test")
	// Log with attributes to exercise testLogHandler.
	inst.Log.Info("msg", "key1", "val1", "key2", "val2")
}

func TestTestLogHandler_WithGroup(t *testing.T) {
	SetTestMode(t)
	inst := New("test")
	inst.Log.Info("grouped msg")
}
