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

package kvdb2ethdb

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestWrap(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)
	if adapter == nil {
		t.Fatal("Wrap returned nil")
	}
}

func TestAdapter_ImplementsKeyValueStore(t *testing.T) {
	var _ ethdb.KeyValueStore = (*Adapter)(nil)
}

func TestAdapter_PutAndGet(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := adapter.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

func TestAdapter_Has(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	has, err := adapter.Has([]byte("missing"))
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if has {
		t.Error("expected false for missing key")
	}

	if err := adapter.Put([]byte("key"), []byte("val")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	has, err = adapter.Has([]byte("key"))
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !has {
		t.Error("expected true for existing key")
	}
}

func TestAdapter_Delete(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("key"), []byte("val")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Delete([]byte("key")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	has, _ := adapter.Has([]byte("key"))
	if has {
		t.Error("key should be deleted")
	}
}

func TestAdapter_NewBatch(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	batch := adapter.NewBatch()
	if batch == nil {
		t.Fatal("NewBatch returned nil")
	}

	if err := batch.Put([]byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("batch Put failed: %v", err)
	}
	if err := batch.Put([]byte("k2"), []byte("v2")); err != nil {
		t.Fatalf("batch Put failed: %v", err)
	}
	if err := batch.Write(); err != nil {
		t.Fatalf("batch Write failed: %v", err)
	}

	got, err := adapter.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("expected 'v1', got %q", got)
	}
}

func TestAdapter_NewBatchWithSize(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	batch := adapter.NewBatchWithSize(1024)
	if batch == nil {
		t.Fatal("NewBatchWithSize returned nil")
	}

	if err := batch.Put([]byte("key"), []byte("val")); err != nil {
		t.Fatalf("batch Put failed: %v", err)
	}
	if err := batch.Write(); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := adapter.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "val" {
		t.Errorf("expected 'val', got %q", got)
	}
}

func TestAdapter_NewIterator(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("c"), []byte("3")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	iter := adapter.NewIterator(nil, nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 entries, got %d", count)
	}
}

func TestAdapter_NewIterator_WithPrefix(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("prefix-a"), []byte("1")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("prefix-b"), []byte("2")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("other-c"), []byte("3")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	iter := adapter.NewIterator([]byte("prefix-"), nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 entries with prefix, got %d", count)
	}
}

func TestAdapter_DeleteRange(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("c"), []byte("3")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := adapter.Put([]byte("d"), []byte("4")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Delete range [b, d) - should delete b and c.
	if err := adapter.DeleteRange([]byte("b"), []byte("d")); err != nil {
		t.Fatalf("DeleteRange failed: %v", err)
	}

	// a and d should remain.
	has, _ := adapter.Has([]byte("a"))
	if !has {
		t.Error("'a' should still exist")
	}
	has, _ = adapter.Has([]byte("b"))
	if has {
		t.Error("'b' should be deleted")
	}
	has, _ = adapter.Has([]byte("c"))
	if has {
		t.Error("'c' should be deleted")
	}
	has, _ = adapter.Has([]byte("d"))
	if !has {
		t.Error("'d' should still exist")
	}
}

func TestAdapter_DeleteRange_Empty(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	// Should not error on empty range.
	if err := adapter.DeleteRange([]byte("a"), []byte("z")); err != nil {
		t.Fatalf("DeleteRange on empty DB failed: %v", err)
	}
}

func TestBatch_Replay(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	batch := adapter.NewBatch()
	if err := batch.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("batch Put failed: %v", err)
	}

	// Replay onto a different store.
	db2 := memorydb.New()
	adapter2 := Wrap(db2)
	var w kvdb.Writer = adapter2
	if err := batch.Replay(w); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	got, _ := adapter2.Get([]byte("key"))
	if string(got) != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}
