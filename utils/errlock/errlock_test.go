package errlock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	l := New("/tmp/test")
	if l == nil {
		t.Fatal("expected non-nil ErrorLock")
	}
	if l.dataDir != "/tmp/test" {
		t.Fatalf("expected /tmp/test, got %s", l.dataDir)
	}
}

func TestCheck_NoLockFile(t *testing.T) {
	dir := t.TempDir()
	l := New(dir)
	if err := l.Check(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheck_WithLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "errlock")
	err := os.WriteFile(lockPath, []byte("some error reason"), 0600)
	if err != nil {
		t.Fatalf("failed to write lock file: %v", err)
	}

	l := New(dir)
	err = l.Check()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "some error reason") {
		t.Fatalf("error should contain the lock reason, got: %v", err)
	}
	if !strings.Contains(err.Error(), "errlock") {
		t.Fatalf("error should reference the lock file path, got: %v", err)
	}
}

func TestCheck_EmptyLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "errlock")
	err := os.WriteFile(lockPath, []byte(""), 0600)
	if err != nil {
		t.Fatalf("failed to write lock file: %v", err)
	}

	l := New(dir)
	err = l.Check()
	if err == nil {
		t.Fatal("expected error for existing (even empty) lock file")
	}
}

func TestPermanent_Panics(t *testing.T) {
	dir := t.TempDir()
	l := New(dir)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("expected error type, got %T", r)
		}
		if !strings.Contains(err.Error(), "test error") {
			t.Fatalf("error should contain 'test error', got: %v", err)
		}
	}()

	l.Permanent(errForTest("test error"))
}

func TestPermanent_WritesLockFile(t *testing.T) {
	dir := t.TempDir()
	l := New(dir)

	defer func() {
		recover() // catch the panic
		lockPath := filepath.Join(dir, "errlock")
		data, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("lock file should exist: %v", err)
		}
		if string(data) != "permanent error" {
			t.Fatalf("lock file content should be 'permanent error', got: %s", string(data))
		}
	}()

	l.Permanent(errForTest("permanent error"))
}

type errForTest string

func (e errForTest) Error() string {
	return string(e)
}

func TestReadAll_MaxBytes(t *testing.T) {
	data := strings.Repeat("x", 100)
	reader := strings.NewReader(data)
	result, err := readAll(reader, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 50 {
		t.Fatalf("expected 50 bytes, got %d", len(result))
	}
}

func TestReadAll_LessThanMax(t *testing.T) {
	data := "short"
	reader := strings.NewReader(data)
	result, err := readAll(reader, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != data {
		t.Fatalf("expected %q, got %q", data, string(result))
	}
}

func TestWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path, err := write(dir, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(dir, "errlock") {
		t.Fatalf("unexpected path: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "test content" {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestRead_NoFile(t *testing.T) {
	dir := t.TempDir()
	locked, reason, path, err := read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Fatal("should not be locked")
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
	if path != filepath.Join(dir, "errlock") {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestRead_WithFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "errlock")
	os.WriteFile(lockPath, []byte("error msg"), 0600)

	locked, reason, path, err := read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Fatal("should be locked")
	}
	if reason != "error msg" {
		t.Fatalf("expected 'error msg', got %q", reason)
	}
	if path != lockPath {
		t.Fatalf("unexpected path: %s", path)
	}
}
