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

package utils

import (
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
)

func OpenFile(path string, isSyncMode bool) *os.File {
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		log.Crit("Failed to create file dir", "file", path, "err", err)
	}
	sync := 0
	if isSyncMode {
		sync = os.O_SYNC
	}
	fh, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|sync, 0666)
	if err != nil {
		log.Crit("Failed to open file", "file", path, "err", err)
	}
	return fh
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
