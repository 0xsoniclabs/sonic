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

func TestMakeTempDir_CreatesEmptyDirectoryWipingPreexistingContent(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	stale := filepath.Join(parent, "scratch", "stale.txt")
	require.NoError(os.MkdirAll(filepath.Dir(stale), 0700))
	require.NoError(os.WriteFile(stale, []byte("old"), 0600))

	dir, cleanup, err := MakeTempDir(parent, "scratch")
	require.NoError(err)
	require.Equal(filepath.Join(parent, "scratch"), dir)
	require.NotNil(cleanup)

	entries, err := os.ReadDir(dir)
	require.NoError(err)
	require.Empty(entries)

	info, err := os.Stat(dir)
	require.NoError(err)
	require.True(info.IsDir())
}

func TestMakeTempDir_ReturnsErrorWhenParentIsNotADirectory(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()
	notADir := filepath.Join(parent, "file")
	require.NoError(os.WriteFile(notADir, []byte(""), 0600))

	dir, cleanup, err := MakeTempDir(notADir, "scratch")
	require.Error(err)
	require.Empty(dir)
	require.Nil(cleanup)
}

func TestMakeTempDir_CleanupRemovesDirectoryAndLeavesErrorNilOnSuccess(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	dir, cleanup, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	var retErr error
	cleanup(&retErr)
	require.NoError(retErr)

	_, err = os.Stat(dir)
	require.True(os.IsNotExist(err))
}

func TestMakeTempDir_CleanupJoinsWithExistingError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks are bypassed when running as root")
	}
	require := require.New(t)
	parent := t.TempDir()

	dir, cleanup, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	require.NoError(os.WriteFile(filepath.Join(dir, "child"), []byte("x"), 0600))
	require.NoError(os.Chmod(dir, 0500))
	defer func() {
		require.NoError(os.Chmod(dir, 0700))
	}()

	original := errors.New("boom")
	retErr := original
	cleanup(&retErr)
	require.ErrorIs(retErr, original)
	require.ErrorIs(retErr, os.ErrPermission)
}

func TestMakeTempDir_CleanupWithNilPointerDoesNotPanic(t *testing.T) {
	require := require.New(t)
	parent := t.TempDir()

	_, cleanup, err := MakeTempDir(parent, "scratch")
	require.NoError(err)

	require.NotPanics(func() { cleanup(nil) })
}
