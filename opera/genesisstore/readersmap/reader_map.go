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

package readersmap

import (
	"errors"
	"io"
)

type ReaderProvider func() (io.Reader, error)

type Map map[string]ReaderProvider

type Unit struct {
	Name string
	ReaderProvider
}

var (
	ErrNotFound = errors.New("not found")
	ErrDupFile  = errors.New("unit name is duplicated")
)

func Wrap(rr []Unit) (Map, error) {
	units := make(Map)
	for _, r := range rr {
		if units[r.Name] != nil {
			return nil, ErrDupFile
		}
		units[r.Name] = r.ReaderProvider
	}
	return units, nil
}

func (mm Map) Open(name string) (io.Reader, error) {
	f := mm[name]
	if f == nil {
		return nil, ErrNotFound
	}
	return f()
}
