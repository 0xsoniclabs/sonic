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
