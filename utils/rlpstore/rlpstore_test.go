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

package rlpstore

import (
	"testing"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
)

func TestHelper_SetAndGet(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer func() { _ = db.Close() }()

	key := []byte("testkey")
	val := uint64(42)

	h.Set(db, key, val)

	var result uint64
	got := h.Get(db, key, &result)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

func TestHelper_Get_MissingKey(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer func() { _ = db.Close() }()

	var result uint64
	got := h.Get(db, []byte("missing"), &result)
	if got != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestHelper_SetOverwrite(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer func() { _ = db.Close() }()

	key := []byte("key")
	h.Set(db, key, uint64(1))
	h.Set(db, key, uint64(2))

	var result uint64
	h.Get(db, key, &result)
	if result != 2 {
		t.Fatalf("expected 2, got %d", result)
	}
}

func TestHelper_SetAndGet_String(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer func() { _ = db.Close() }()

	key := []byte("strkey")
	val := "hello"

	h.Set(db, key, val)

	var result string
	got := h.Get(db, key, &result)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestHelper_SetAndGet_Bytes(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer func() { _ = db.Close() }()

	key := []byte("byteskey")
	val := []byte{0x01, 0x02, 0x03}

	h.Set(db, key, val)

	var result []byte
	got := h.Get(db, key, &result)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 3 || result[0] != 0x01 || result[1] != 0x02 || result[2] != 0x03 {
		t.Fatalf("unexpected result: %v", result)
	}
}
