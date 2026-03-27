package encryption

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTemporaryKeyFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "keyfile.json")
	content := []byte(`{"test": true}`)

	tmpName, err := writeTemporaryKeyFile(target, content)
	if err != nil {
		t.Fatalf("failed to write temporary key file: %v", err)
	}

	// Temporary file should exist
	data, err := os.ReadFile(tmpName)
	if err != nil {
		t.Fatalf("failed to read temporary file: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("content mismatch: %s", string(data))
	}

	// Clean up
	os.Remove(tmpName)
}

func TestWriteTemporaryKeyFile_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "subdir", "keyfile.json")
	content := []byte(`{"test": true}`)

	tmpName, err := writeTemporaryKeyFile(target, content)
	if err != nil {
		t.Fatalf("failed to write temporary key file: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(filepath.Join(dir, "subdir"))
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}

	os.Remove(tmpName)
}
