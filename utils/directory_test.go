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
	"errors"
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

func TestTempDir_Path_ReturnsCorrectPath(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)
	require.Equal(filepath.Join(parent, "scratch"), dir.Path())
}

func TestTempDir_Cleanup_RemovesDirectoryAndLeavesErrorNilOnSuccess(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	var retErr error
	dir.Cleanup(&retErr)
	require.NoError(retErr)

	_, err = os.Stat(dir.Path())
	require.True(os.IsNotExist(err))
}

func TestTempDir_Cleanup_JoinsWithExistingError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks are bypassed when running as root")
	}
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	require.NoError(os.WriteFile(filepath.Join(dir.Path(), "child"), []byte("x"), 0600))
	require.NoError(os.Chmod(dir.Path(), 0500))
	defer func() {
		require.NoError(os.Chmod(dir.Path(), 0700))
	}()

	original := errors.New("boom")
	retErr := original
	dir.Cleanup(&retErr)
	require.ErrorIs(retErr, original)
	require.ErrorIs(retErr, os.ErrPermission)
}

func TestTempDir_Cleanup_WithNilPointerDoesNotPanic(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	require.NotPanics(func() { dir.Cleanup(nil) })
}
