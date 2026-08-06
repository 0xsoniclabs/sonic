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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempDir_MakeTempDir_CreatesEmptyDirectoryWipingPreexistingContent(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	stale := filepath.Join(parent, "scratch", "stale.txt")
	require.NoError(os.MkdirAll(filepath.Dir(stale), 0700))
	require.NoError(os.WriteFile(stale, []byte("old"), 0600))

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)
	require.Equal(filepath.Join(parent, "scratch"), dir.Path())

	entries, err := os.ReadDir(dir.Path())
	require.NoError(err)
	require.Empty(entries)

	info, err := os.Stat(dir.Path())
	require.NoError(err)
	require.True(info.IsDir())
}

func TestTempDir_MakeTempDir_ReturnsErrorWhenParentIsNotADirectory(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()
	notADir := filepath.Join(parent, "file")
	require.NoError(os.WriteFile(notADir, []byte(""), 0600))

	dir, err := MakeTempDir(notADir, "scratch")
	require.Error(err)
	require.Empty(dir.Path())
}

func TestTempDir_MakeTempDir_RejectsNonBasenameNames(t *testing.T) {
	parent := t.TempDir()

	// Create a sibling directory that must not be touched by MakeTempDir when
	// a caller attempts to escape the intended parent via a relative name.
	sibling := filepath.Join(parent, "sibling")
	require.NoError(t, os.MkdirAll(sibling, 0700))
	sentinel := filepath.Join(sibling, "keep.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0600))

	testCases := map[string]string{
		"empty":            "",
		"dot":              ".",
		"double-dot":       "..",
		"parent-traversal": "../sibling",
		"nested":           "foo/bar",
		"absolute":         "/tmp/attack",
		"leading-slash":    "/scratch",
		"trailing-slash":   "scratch/",
	}

	scratchParent := filepath.Join(parent, "scratch-parent")
	require.NoError(t, os.MkdirAll(scratchParent, 0700))

	for name, invalidName := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			dir, err := MakeTempDir(scratchParent, invalidName)
			require.Error(err)
			require.Empty(dir.Path())
			require.Contains(err.Error(), "invalid directory name")

			// Sibling directory and its sentinel file must still exist:
			// MakeTempDir must not have escaped the intended parent.
			_, statErr := os.Stat(sentinel)
			require.NoError(statErr)
		})
	}
}

func TestTempDir_Path_ReturnsCorrectPath(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)
	require.Equal(filepath.Join(parent, "scratch"), dir.Path())
}

func TestTempDir_Cleanup_RemovesDirectory(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	err = dir.Cleanup()
	require.NoError(err)

	_, err = os.Stat(dir.Path())
	require.True(os.IsNotExist(err))
}
