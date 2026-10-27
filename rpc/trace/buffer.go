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

package trace

import (
	"bytes"
	"encoding/json"
	"errors"
)

// bufferStartSize is the size a result buffer starts with.
const bufferStartSize = 10 * 1024

// errResponseTooLarge is the reference's answer to traces above
// MaxResponseSize.
var errResponseTooLarge = errors.New("response too large")

// resultBuffer collects encoded traces into a JSON list no longer than its
// limit, as the reference's JsonResultBuffer did; a limit of zero admits
// nothing, as there.
type resultBuffer struct {
	bytes.Buffer
	limit int
}

func newResultBuffer(limit int) (*resultBuffer, error) {
	buffer := &resultBuffer{limit: limit}
	buffer.Grow(bufferStartSize)
	if err := buffer.write([]byte("[")); err != nil {
		return nil, err
	}
	return buffer, nil
}

// add appends one object to the list.
func (buffer *resultBuffer) add(object any) error {
	encoded, err := json.Marshal(object)
	if err != nil {
		return err
	}
	if buffer.Len() > 1 {
		if err := buffer.write([]byte(",")); err != nil {
			return err
		}
	}
	return buffer.write(encoded)
}

// left is what is left of the limit for one more object in the list.
func (buffer *resultBuffer) left() int {
	return buffer.limit - buffer.Len() - len(",]")
}

// result closes the list.
func (buffer *resultBuffer) result() ([]byte, error) {
	if err := buffer.write([]byte("]")); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (buffer *resultBuffer) write(encoded []byte) error {
	if buffer.Len()+len(encoded) > buffer.limit {
		return errResponseTooLarge
	}
	_, err := buffer.Write(encoded)
	return err
}
