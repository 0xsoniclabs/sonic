// Copyright 2025 Sonic Operations Ltd
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

package errlock

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/0xsoniclabs/sonic/utils/caution"
)

type ErrorLock struct {
	dataDir string
}

// New creates a new ErrLock instance with the specified data directory.
func New(dataDir string) *ErrorLock {
	return &ErrorLock{
		dataDir: dataDir,
	}
}

// Check if errlock is written
func (l *ErrorLock) Check() error {
	locked, reason, eLockPath, err := read(l.dataDir)
	if err != nil {
		// This is a user-facing error, so we want to provide a clear message.
		//nolint:staticcheck // ST1005: allow capitalized error message and punctuation
		return fmt.Errorf("Node isn't allowed to start due to an error reading"+
			" the lock file %s.\n Please fix the issue. Error message:\n%w",
			eLockPath, err)
	}

	if locked {
		// This is a user-facing error, so we want to provide a clear message.
		//nolint:staticcheck // ST1005: allow capitalized error message and punctuation
		return fmt.Errorf("Node isn't allowed to start due to a previous error."+
			" Please fix the issue and then delete file \"%s\". Error message:\n%s",
			eLockPath, reason)
	}
	return nil
}

// Permanent error
func (l *ErrorLock) Permanent(err error) {
	eLockPath, _ := write(l.dataDir, err.Error())
	// This is a user-facing error, so we want to provide a clear message.
	//nolint:staticcheck // ST1005: allow capitalized error message and punctuation
	panic(fmt.Errorf("Node is permanently stopping due to an issue. Please fix"+
		" the issue and then delete file \"%s\". Error message:\n%s",
		eLockPath, err.Error()))
}

func readAll(reader io.Reader, max int) ([]byte, error) {
	buf := make([]byte, max)
	consumed := 0
	for {
		n, err := reader.Read(buf[consumed:])
		consumed += n
		if consumed == len(buf) || err == io.EOF {
			return buf[:consumed], nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// read errlock file
func read(dir string) (locked bool, reason string, eLockPath string, err error) {
	eLockPath = path.Join(dir, "errlock")

	data, err := os.Open(eLockPath)
	if err != nil {
		// if file doesn't exist, directory is not locked and it is not an error
		return false, "", eLockPath, nil
	}
	defer caution.CloseAndReportError(&err, data, "Failed to close errlock file")

	// read no more than N bytes
	maxFileLen := 5000
	eLockBytes, err := readAll(data, maxFileLen)
	if err != nil {
		return true, "", eLockPath, fmt.Errorf("failed to read lock file %v: %w", eLockPath, err)
	}
	return true, string(eLockBytes), eLockPath, nil
}

// write errlock file
func write(dir string, eLockStr string) (string, error) {
	eLockPath := path.Join(dir, "errlock")

	return eLockPath, os.WriteFile(eLockPath, []byte(eLockStr), 0600) // assume no custom encoding needed
}
