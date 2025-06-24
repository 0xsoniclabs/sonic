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

package encryption

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/sonic/utils"
)

func writeTemporaryKeyFile(file string, content []byte) (string, error) {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return "", fmt.Errorf("failed to create keystore directory: %w", err)
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary key file: %w", err)
	}

	if _, err = f.Write(content); err != nil {
		return "", errors.Join(
			fmt.Errorf("failed to write key file: %w", err),
			utils.AnnotateIfError(f.Close(), "failed to close key file"),
			utils.AnnotateIfError(os.Remove(f.Name()), "failed to remove temporary key file"),
		)
	}

	return f.Name(), f.Close()
}
