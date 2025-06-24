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

package devnullfile

type DevNull struct{}

func (d DevNull) Read(pp []byte) (n int, err error) {
	for i := range pp {
		pp[i] = 0
	}
	return len(pp), nil
}

func (d DevNull) Write(pp []byte) (n int, err error) {
	return len(pp), nil
}

func (d DevNull) Close() error {
	return nil
}

func (d DevNull) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (d DevNull) Drop() error {
	return nil
}
