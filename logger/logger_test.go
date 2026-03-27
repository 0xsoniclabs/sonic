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

package logger

import (
	"testing"
)

func TestNew_NoArgs(t *testing.T) {
	inst := New()
	if inst.Log == nil {
		t.Fatal("expected non-nil Log")
	}
}

func TestNew_WithName(t *testing.T) {
	inst := New("testmodule")
	if inst.Log == nil {
		t.Fatal("expected non-nil Log")
	}
}

func TestSetLevel_Valid(t *testing.T) {
	levels := []string{"debug", "dbug", "info", "warn", "error", "eror"}
	for _, l := range levels {
		// Should not panic
		SetLevel(l)
	}
	// Restore to default
	SetLevel("info")
}

func TestSetLevel_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid level")
		}
	}()
	SetLevel("invalid_level")
}

func TestLevelFromString_Valid(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"debug"},
		{"dbug"},
		{"info"},
		{"warn"},
		{"error"},
		{"eror"},
	}
	for _, tt := range tests {
		_, err := levelFromString(tt.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.input, err)
		}
	}
}

func TestLevelFromString_Invalid(t *testing.T) {
	_, err := levelFromString("unknown")
	if err == nil {
		t.Fatal("expected error for unknown level")
	}
}

func TestPeriodic_Info(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	// Should not panic
	p.Info(0, "test message", "key", "value")
}

func TestPeriodic_Warn(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	p.Warn(0, "test warning")
}

func TestPeriodic_Error(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	p.Error(0, "test error")
}

func TestPeriodic_Debug(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	p.Debug(0, "test debug")
}

func TestPeriodic_Trace(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	p.Trace(0, "test trace")
}

func TestPeriodic_RateLimiting(t *testing.T) {
	p := &Periodic{Instance: New("test")}
	// First call should log (period=0 means always)
	p.Info(0, "first call")
	// With a very long period, second call should not log but also should not panic
	p.Info(1<<62, "second call - should be skipped")
}
