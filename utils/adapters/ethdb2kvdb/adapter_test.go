package ethdb2kvdb

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestWrap(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)
	if adapter == nil {
		t.Fatal("Wrap returned nil")
	}
}

func TestAdapter_ImplementsKvdbStore(t *testing.T) {
	var _ kvdb.Store = (*Adapter)(nil)
}

func TestAdapter_PutAndGet(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	key := []byte("test-key")
	value := []byte("test-value")

	if err := adapter.Put(key, value); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := adapter.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestAdapter_Has(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	key := []byte("key")
	has, err := adapter.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if has {
		t.Error("expected Has to return false for missing key")
	}

	if err := adapter.Put(key, []byte("val")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	has, err = adapter.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !has {
		t.Error("expected Has to return true for existing key")
	}
}

func TestAdapter_Delete(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	key := []byte("key")
	if err := adapter.Put(key, []byte("val")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := adapter.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	has, err := adapter.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if has {
		t.Error("key should not exist after Delete")
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
		t.Errorf("expected v1, got %q", got)
	}
}

func TestAdapter_NewIterator(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	if err := adapter.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatal(err)
	}

	iter := adapter.NewIterator(nil, nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 entries, got %d", count)
	}
}

func TestAdapter_Drop_Panics(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Drop to panic")
		}
	}()
	adapter.Drop()
}

func TestAdapter_GetSnapshot_Panics(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected GetSnapshot to panic")
		}
	}()
	adapter.GetSnapshot()
}

func TestBatch_Replay(t *testing.T) {
	db := memorydb.New()
	adapter := Wrap(db)

	b := adapter.NewBatch()
	if err := b.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatal(err)
	}

	// Replay onto a separate writer (another batch).
	db2 := memorydb.New()
	adapter2 := Wrap(db2)
	var w ethdb.KeyValueWriter = adapter2
	if err := b.Replay(w); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	got, err := adapter2.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}
