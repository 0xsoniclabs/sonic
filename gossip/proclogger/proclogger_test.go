package proclogger

import (
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	if l == nil {
		t.Fatal("expected non-nil Logger")
	}
	if l.Log == nil {
		t.Fatal("expected non-nil Log")
	}
}

func TestLogger_Summary_NoSummary(t *testing.T) {
	l := NewLogger()
	l.noSummary = true
	// Should not panic or log
	l.summary(time.Now())
}

func TestLogger_Summary_EmptySums(t *testing.T) {
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Time{} // zero time, so now is after it
	// Should not panic - empty sums mean nothing to log
	l.summary(time.Now())
}

func TestLogger_Summary_WithDagSum(t *testing.T) {
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Time{}
	l.dagSum.connected = 5
	l.dagSum.totalProcessing = time.Second
	// Should not panic
	l.summary(time.Now())
	// After summary, counters should be reset
	if l.dagSum.connected != 0 {
		t.Fatal("expected dagSum to be reset")
	}
}

func TestLogger_Summary_WithLlrSum(t *testing.T) {
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Time{}
	l.llrSum.bvs = 3
	l.llrSum.brs = 2
	l.llrSum.evs = 1
	l.llrSum.ers = 1
	l.lastLlrTime = 2000 // greater than lastEventTime
	l.lastEventTime = 1000
	// Should not panic
	l.summary(time.Now())
	if l.llrSum.bvs != 0 {
		t.Fatal("expected llrSum to be reset")
	}
}

func TestLogger_Summary_LlrTimeNone(t *testing.T) {
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Time{}
	l.llrSum.bvs = 1
	l.lastLlrTime = 500
	l.lastEventTime = 1000 // lastLlrTime <= lastEventTime means "none"
	l.summary(time.Now())
}

func TestLogger_Summary_NotYetTime(t *testing.T) {
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Now().Add(time.Hour) // far in the future
	l.dagSum.connected = 5
	l.summary(time.Now())
	// Should not reset since we haven't reached nextLogging
	if l.dagSum.connected != 5 {
		t.Fatal("expected dagSum to NOT be reset")
	}
}
