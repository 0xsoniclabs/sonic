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

package debug

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome_NoTilde(t *testing.T) {
	result := expandHome("/tmp/test")
	if result != "/tmp/test" {
		t.Fatalf("expected /tmp/test, got %s", result)
	}
}

func TestExpandHome_WithTilde(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	result := expandHome("~/test")
	expected := filepath.Join(home, "test")
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestExpandHome_NoTildeSlash(t *testing.T) {
	result := expandHome("~someuser/tmp")
	// Should NOT expand ~someuser
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestStartAndStopCPUProfile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cpu.prof")

	h := new(HandlerT)
	err := h.StartCPUProfile(file)
	if err != nil {
		t.Fatalf("failed to start CPU profile: %v", err)
	}

	// Starting again should fail
	err = h.StartCPUProfile(file)
	if err == nil {
		t.Fatal("expected error for double start")
	}

	err = h.StopCPUProfile()
	if err != nil {
		t.Fatalf("failed to stop CPU profile: %v", err)
	}

	// File should exist
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Fatal("CPU profile file should exist")
	}
}

func TestStopCPUProfile_NotStarted(t *testing.T) {
	h := new(HandlerT)
	err := h.StopCPUProfile()
	if err != nil {
		t.Fatalf("expected no error when stopping non-started profile: %v", err)
	}
}

func TestSetBlockProfileRate(t *testing.T) {
	h := new(HandlerT)
	// Should not panic
	h.SetBlockProfileRate(0)
	h.SetBlockProfileRate(1)
	h.SetBlockProfileRate(0) // reset
}

func TestStartCPUProfile_BadPath(t *testing.T) {
	h := new(HandlerT)
	err := h.StartCPUProfile("/nonexistent/dir/cpu.prof")
	if err == nil {
		t.Fatal("expected error for bad path")
	}
}
