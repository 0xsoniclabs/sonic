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
	"github.com/stretchr/testify/require"
)

func TestLoadOrCreateHostKey_NoPath_GeneratesInMemory(t *testing.T) {
	key, err := loadOrCreateHostKey("")
	require.NoError(t, err, "loadOrCreateHostKey failed")
	_, err = peer.IDFromPrivateKey(key)
	require.NoError(t, err, "generated key does not yield a peer ID")
}

func TestLoadOrCreateHostKey_WithPath_PersistsAndReloadsSameIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nodekey")

	first, err := loadOrCreateHostKey(path)
	require.NoError(t, err, "first load failed")
	_, err = os.Stat(path)
	require.NoError(t, err, "expected key to be persisted at %s", path)

	second, err := loadOrCreateHostKey(path)
	require.NoError(t, err, "second load failed")

	firstID, _ := peer.IDFromPrivateKey(first)
	secondID, _ := peer.IDFromPrivateKey(second)
	require.Equal(t, firstID, secondID, "expected stable peer ID across reloads")
}

func TestLoadOrCreateHostKey_NoPath_DoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	_, err := loadOrCreateHostKey("")
	require.NoError(t, err, "loadOrCreateHostKey failed")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "failed to read temp dir")
	require.Empty(t, entries, "expected no files written for in-memory key")
}
