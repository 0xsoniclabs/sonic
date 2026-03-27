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

package devnullfile

import (
	"io"
	"testing"
)

func TestDevNull_Read(t *testing.T) {
	d := DevNull{}
	buf := []byte{1, 2, 3, 4, 5}
	n, err := d.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("expected %d, got %d", len(buf), n)
	}
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("expected buf[%d] == 0, got %d", i, b)
		}
	}
}

func TestDevNull_ReadEmpty(t *testing.T) {
	d := DevNull{}
	buf := []byte{}
	n, err := d.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestDevNull_Write(t *testing.T) {
	d := DevNull{}
	data := []byte("hello world")
	n, err := d.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d, got %d", len(data), n)
	}
}

func TestDevNull_WriteEmpty(t *testing.T) {
	d := DevNull{}
	n, err := d.Write([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestDevNull_Close(t *testing.T) {
	d := DevNull{}
	if err := d.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDevNull_Seek(t *testing.T) {
	d := DevNull{}
	tests := []struct {
		offset int64
		whence int
	}{
		{0, io.SeekStart},
		{100, io.SeekCurrent},
		{-50, io.SeekEnd},
	}
	for _, tt := range tests {
		pos, err := d.Seek(tt.offset, tt.whence)
		if err != nil {
			t.Fatalf("unexpected error for offset=%d whence=%d: %v", tt.offset, tt.whence, err)
		}
		if pos != 0 {
			t.Fatalf("expected position 0, got %d", pos)
		}
	}
}

func TestDevNull_Drop(t *testing.T) {
	d := DevNull{}
	if err := d.Drop(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
