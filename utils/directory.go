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
)

type TempDir string

// MakeTempDir creates a temporary directory at the specified path with the given name.
// Everything in the directory is removed before creation. The function returns the path to the created directory,
// a cleanup function to remove the directory, and an error if any occurred during the process.
func MakeTempDir(path string, name string) (TempDir, error) {
	dir := filepath.Join(path, name)
	err := os.RemoveAll(dir)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	return TempDir(dir), nil
}

func (td TempDir) Path() string {
	return string(td)
}

func (td TempDir) Cleanup(retErr *error) {
	if retErr == nil {
		retErr = new(error)
	}
	*retErr = errors.Join(*retErr, os.RemoveAll(string(td)))
}
