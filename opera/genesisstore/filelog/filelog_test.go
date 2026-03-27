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

package filelog

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestWrap(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)
	fl := Wrap(r, "test", uint64(len(data)), time.Second)
	if fl == nil {
		t.Fatal("expected non-nil Filelog")
	}
	if fl.name != "test" {
		t.Fatalf("expected name 'test', got %q", fl.name)
	}
	if fl.size != uint64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), fl.size)
	}
}

func TestFilelog_Read(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)
	fl := Wrap(r, "test", uint64(len(data)), time.Hour)

	buf := make([]byte, len(data))
	n, err := fl.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("data mismatch")
	}
	if fl.consumed != uint64(len(data)) {
		t.Fatalf("expected consumed %d, got %d", len(data), fl.consumed)
	}
}

func TestFilelog_ReadAll(t *testing.T) {
	data := []byte("abcdefghij")
	r := bytes.NewReader(data)
	fl := Wrap(r, "test", uint64(len(data)), time.Hour)

	result, err := io.ReadAll(fl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Fatalf("data mismatch")
	}
}

func TestFilelog_Read_EmptyReader(t *testing.T) {
	fl := Wrap(bytes.NewReader(nil), "empty", 0, time.Hour)
	buf := make([]byte, 10)
	n, err := fl.Read(buf)
	if n != 0 {
		t.Fatalf("expected 0 bytes, got %d", n)
	}
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestFilelog_ProgressLogging(t *testing.T) {
	// Create a larger dataset and a very short period to trigger progress logging
	data := bytes.Repeat([]byte("x"), 1000)
	r := bytes.NewReader(data)
	fl := Wrap(r, "progress_test", uint64(len(data)), 0) // period=0 means always log

	buf := make([]byte, 100)
	for {
		_, err := fl.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if fl.consumed != 1000 {
		t.Fatalf("expected 1000 consumed, got %d", fl.consumed)
	}
}
