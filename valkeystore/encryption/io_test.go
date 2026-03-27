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
	_ = os.Remove(tmpName)
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

	_ = os.Remove(tmpName)
}
