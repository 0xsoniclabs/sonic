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

package utils

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

const permissionMask = os.FileMode(0o777)

func TestOpenFile_CreatesWithOwnerOnlyPermissions(t *testing.T) {
	// Zero the umask so the on-disk mode matches what OpenFile requests.
	prev := syscall.Umask(0)
	defer syscall.Umask(prev)

	path := filepath.Join(t.TempDir(), "emitted-event")

	fh := OpenFile(path, false)
	require.NotNil(t, fh)
	require.NoError(t, fh.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode()&permissionMask)
}

// TestOpenFile_PreservesExistingFilePermissions guards the in-place upgrade
// path: open(2)'s mode argument only applies at creation, so an existing
// file must keep its prior permissions and contents.
func TestOpenFile_PreservesExistingFilePermissions(t *testing.T) {
	prev := syscall.Umask(0)
	defer syscall.Umask(prev)

	preExistingModes := map[string]os.FileMode{
		"legacy 0666":         0o666,
		"legacy 0644":         0o644,
		"already 0600":        0o600,
		"group readable 0640": 0o640,
	}

	for name, existingMode := range preExistingModes {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "emitted-event")

			require.NoError(t, os.WriteFile(path, []byte("legacy-payload"), existingMode))
			require.NoError(t, os.Chmod(path, existingMode))

			fh := OpenFile(path, false)
			require.NotNil(t, fh)

			data, err := io.ReadAll(fh)
			require.NoError(t, err)
			require.Equal(t, "legacy-payload", string(data))
			require.NoError(t, fh.Close())

			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, existingMode, info.Mode()&permissionMask)
		})
	}
}

func TestOpenFile_UmaskInteraction(t *testing.T) {
	// 0o077 strips group+other bits; 0600 passes through unchanged.
	prev := syscall.Umask(0o077)
	defer syscall.Umask(prev)

	require.Equal(t, os.FileMode(0o077), currentUmask(t))

	path := filepath.Join(t.TempDir(), "emitted-event")
	fh := OpenFile(path, false)
	require.NotNil(t, fh)
	require.NoError(t, fh.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode()&permissionMask)
}

// currentUmask reads the process umask via the standard set-and-restore
// round-trip (syscall.Umask has no read-only variant).
func currentUmask(t *testing.T) os.FileMode {
	t.Helper()
	prev := syscall.Umask(0)
	syscall.Umask(prev)
	return os.FileMode(prev)
}
