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

package p2p

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestLoadOrCreateHostKey_NoPath_GeneratesInMemory(t *testing.T) {
	key, err := loadOrCreateHostKey("")
	if err != nil {
		t.Fatalf("loadOrCreateHostKey failed: %v", err)
	}
	if _, err := peer.IDFromPrivateKey(key); err != nil {
		t.Fatalf("generated key does not yield a peer ID: %v", err)
	}
}

func TestLoadOrCreateHostKey_WithPath_PersistsAndReloadsSameIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nodekey")

	first, err := loadOrCreateHostKey(path)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected key to be persisted at %s: %v", path, err)
	}

	second, err := loadOrCreateHostKey(path)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	firstID, _ := peer.IDFromPrivateKey(first)
	secondID, _ := peer.IDFromPrivateKey(second)
	if firstID != secondID {
		t.Fatalf("expected stable peer ID across reloads, got %s then %s", firstID, secondID)
	}
}

func TestLoadOrCreateHostKey_NoPath_DoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadOrCreateHostKey(""); err != nil {
		t.Fatalf("loadOrCreateHostKey failed: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files written for in-memory key, found %d", len(entries))
	}
}
