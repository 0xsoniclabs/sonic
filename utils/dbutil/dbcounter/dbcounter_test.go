package dbcounter

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
)

func newTestStore(t *testing.T, warn bool) (*Store, kvdb.Store) {
	t.Helper()
	underlying := memorydb.New()
	store := WrapStore(underlying, "test", warn)
	return store, underlying
}

func TestWrapStore(t *testing.T) {
	store, _ := newTestStore(t, false)
	if store == nil {
		t.Fatal("WrapStore returned nil")
	}
}

func TestStore_PutAndGet(t *testing.T) {
	store, _ := newTestStore(t, false)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	if err := store.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := store.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

func TestStore_Close_NoLeaks(t *testing.T) {
	store, _ := newTestStore(t, false)

	if err := store.Close(); err != nil {
		t.Fatalf("Close should succeed with no leaks: %v", err)
	}
}

func TestStore_Close_IteratorLeak_Error(t *testing.T) {
	store, _ := newTestStore(t, false)

	// Create an iterator but don't release it.
	iter := store.NewIterator(nil, nil)
	_ = iter

	err := store.Close()
	if err == nil {
		t.Fatal("Close should return error when iterators are leaked")
	}

	// Clean up.
	iter.Release()
}

func TestStore_Close_IteratorLeak_Warn(t *testing.T) {
	store, _ := newTestStore(t, true)

	// Create an iterator but don't release it.
	iter := store.NewIterator(nil, nil)
	_ = iter

	// With warn=true, Close should not return an error, just log a warning.
	err := store.Close()
	if err != nil {
		t.Fatalf("Close with warn=true should not return error: %v", err)
	}

	// Clean up.
	iter.Release()
}

func TestStore_Iterator_Release_Decrements(t *testing.T) {
	store, _ := newTestStore(t, false)

	iter := store.NewIterator(nil, nil)
	iter.Release()

	// After releasing, close should succeed.
	if err := store.Close(); err != nil {
		t.Fatalf("Close after iterator release should succeed: %v", err)
	}
}

func TestStore_Snapshot_Release_Decrements(t *testing.T) {
	store, _ := newTestStore(t, false)

	snap, err := store.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}

	snap.Release()

	if err := store.Close(); err != nil {
		t.Fatalf("Close after snapshot release should succeed: %v", err)
	}
}

func TestStore_Snapshot_Leak_Error(t *testing.T) {
	store, _ := newTestStore(t, false)

	snap, err := store.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	_ = snap

	closeErr := store.Close()
	if closeErr == nil {
		t.Fatal("Close should return error when snapshots are leaked")
	}

	snap.Release()
}

func TestStore_IoStats_NotMeasurable(t *testing.T) {
	store, _ := newTestStore(t, false)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	_, err := store.IoStats()
	if err == nil {
		t.Error("IoStats should return error for non-measurable store")
	}
}

func TestStore_UsedDiskSpace_NotMeasurable(t *testing.T) {
	store, _ := newTestStore(t, false)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	_, err := store.UsedDiskSpace()
	if err == nil {
		t.Error("UsedDiskSpace should return error for non-measurable store")
	}
}

func TestStore_MultipleIterators(t *testing.T) {
	store, _ := newTestStore(t, false)

	iter1 := store.NewIterator(nil, nil)
	iter2 := store.NewIterator(nil, nil)

	// Close should fail with 2 leaked iterators.
	err := store.Close()
	if err == nil {
		t.Fatal("Close should return error with 2 leaked iterators")
	}

	iter1.Release()
	iter2.Release()
}
