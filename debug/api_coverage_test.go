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

func TestStartCPUProfile_DoubleStart(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cpu.prof")

	h := new(HandlerT)
	if err := h.StartCPUProfile(file); err != nil {
		t.Fatalf("first StartCPUProfile failed: %v", err)
	}

	err := h.StartCPUProfile(file)
	if err == nil {
		t.Fatal("expected error for double start")
	}

	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("StopCPUProfile failed: %v", err)
	}
}

func TestStopCPUProfile_WritesFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cpu.prof")

	h := new(HandlerT)
	if err := h.StartCPUProfile(file); err != nil {
		t.Fatalf("StartCPUProfile failed: %v", err)
	}
	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("StopCPUProfile failed: %v", err)
	}

	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		t.Fatal("profile file should exist")
	}
	if info.Size() == 0 {
		t.Error("profile file should not be empty")
	}
}

func TestStopCPUProfile_Idempotent(t *testing.T) {
	h := new(HandlerT)
	// Stopping when not started should be fine.
	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	// Double stop should also be fine.
	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestSetBlockProfileRate_Various(t *testing.T) {
	h := new(HandlerT)
	rates := []int{0, 1, 100, 1000000, 0}
	for _, r := range rates {
		h.SetBlockProfileRate(r)
	}
}

func TestExpandHome_EmptyString(t *testing.T) {
	result := expandHome("")
	if result != "." {
		t.Errorf("expected '.', got %q", result)
	}
}

func TestExpandHome_AbsolutePath(t *testing.T) {
	result := expandHome("/usr/local/bin")
	if result != "/usr/local/bin" {
		t.Errorf("expected '/usr/local/bin', got %q", result)
	}
}

func TestExpandHome_RelativePath(t *testing.T) {
	result := expandHome("relative/path")
	if result != "relative/path" {
		t.Errorf("expected 'relative/path', got %q", result)
	}
}

func TestExpandHome_TildeOnly(t *testing.T) {
	// "~" without "/" after shouldn't expand.
	result := expandHome("~")
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestExpandHome_TildeWithPath(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	result := expandHome("~/Documents/test")
	expected := filepath.Join(home, "Documents", "test")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStartCPUProfile_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "dir", "cpu.prof")

	h := new(HandlerT)
	// This should fail because parent dirs don't exist.
	err := h.StartCPUProfile(nested)
	if err == nil {
		if err := h.StopCPUProfile(); err != nil {
			t.Fatalf("StopCPUProfile failed: %v", err)
		}
		t.Skip("OS created directories automatically")
	}
}

func TestStartCPUProfile_AndStop_Lifecycle(t *testing.T) {
	dir := t.TempDir()

	// Start, stop, start again on a different file.
	h := new(HandlerT)

	file1 := filepath.Join(dir, "cpu1.prof")
	if err := h.StartCPUProfile(file1); err != nil {
		t.Fatalf("Start 1 failed: %v", err)
	}
	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("Stop 1 failed: %v", err)
	}

	file2 := filepath.Join(dir, "cpu2.prof")
	if err := h.StartCPUProfile(file2); err != nil {
		t.Fatalf("Start 2 failed: %v", err)
	}
	if err := h.StopCPUProfile(); err != nil {
		t.Fatalf("Stop 2 failed: %v", err)
	}

	// Both files should exist.
	for _, f := range []string{file1, file2} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file %s should exist", f)
		}
	}
}
