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

package ethapi

import (
	"bytes"
	"encoding/json"
	"errors"
)

const (
	// bufferStartSize default buffer start size in bytes
	bufferStartSize = 10 * 1024
)

var (
	// ErrMaxResponseSize is returned if size of the RPC call
	// response is over defined limit
	ErrMaxResponseSize = errors.New("response size exceeded")
	// ErrResponseTooLarge is returned when the result buffer cannot be expanded
	ErrResponseTooLarge = errors.New("response too large")
)

// JsonResultBuffer is a bytes buffer for jsonRawMessage result
type JsonResultBuffer struct {
	bytes.Buffer
	maxResultSize int // limits maximum size of RPC response in bytes
}

// NewJsonResultBuffer creates new bytes buffer
func NewJsonResultBuffer(maxResultSize int) (*JsonResultBuffer, error) {

	b := &JsonResultBuffer{
		maxResultSize: maxResultSize,
	}

	// grow buffer to default start size
	b.Grow(bufferStartSize)
	if err := b.writeString("["); err != nil {
		return nil, err
	}
	return b, nil
}

// GetResult returns finalized byte array
func (b *JsonResultBuffer) GetResult() ([]byte, error) {
	if err := b.writeString("]"); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// AddObject marshals and adds object into buffer.
// In case that buffer size is over defined limit, error is returned
func (b *JsonResultBuffer) AddObject(obj interface{}) error {
	res, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if b.Len() > 1 {
		if err := b.writeString(","); err != nil {
			return err
		}
	}
	if err = b.writeBytes(res); err != nil {
		return err
	}

	return nil
}

// writeString appends the contents of s to the buffer and handle possible panic
func (b *JsonResultBuffer) writeString(s string) error {

	if b.Len()+len(s) > b.maxResultSize {
		return ErrResponseTooLarge
	}

	if _, err := b.WriteString(s); err != nil {
		return err
	}

	return nil
}

// writeBytes appends the contents of arr to the buffer and handle possible panic
func (b *JsonResultBuffer) writeBytes(arr []byte) error {

	if b.Len()+len(arr) > b.maxResultSize {
		return ErrResponseTooLarge
	}

	if _, err := b.Write(arr); err != nil {
		return err
	}

	return nil
}
