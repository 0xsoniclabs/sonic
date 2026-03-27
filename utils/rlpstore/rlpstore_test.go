package rlpstore

import (
	"testing"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
)

func TestHelper_SetAndGet(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer db.Close()

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
	defer db.Close()

	var result uint64
	got := h.Get(db, []byte("missing"), &result)
	if got != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestHelper_SetOverwrite(t *testing.T) {
	h := &Helper{Instance: logger.New()}
	db := memorydb.New()
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
