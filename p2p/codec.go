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

package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

// ErrMessageTooLarge is returned when a framed message declares, or would
// serialize to, a size exceeding the caller-provided limit. It is reported
// before the message body is read, so an oversized frame never causes a large
// allocation.
var ErrMessageTooLarge = errors.New("p2p: message exceeds size limit")

// WriteMessage encodes m as a length-delimited protobuf frame and writes it to
// w. maxSize bounds the encoded body size; the message is rejected before any
// bytes are written if it is too large. It returns the number of bytes written.
func WriteMessage(w io.Writer, m proto.Message, maxSize int) (int, error) {
	body, err := proto.Marshal(m)
	if err != nil {
		return 0, fmt.Errorf("p2p: failed to marshal message: %w", err)
	}
	if len(body) > maxSize {
		return 0, fmt.Errorf("%w: %d > %d", ErrMessageTooLarge, len(body), maxSize)
	}

	var header [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(header[:], uint64(len(body)))
	if written, err := w.Write(header[:n]); err != nil {
		return written, err
	}
	written, err := w.Write(body)
	return n + written, err
}

// ReadMessage reads a single length-delimited protobuf frame from r into m.
// maxSize bounds the declared body size: the length prefix is validated against
// it before the body is read, so an oversized or malicious frame is rejected
// without allocating its body. Because maxSize is supplied per call, different
// message types on the same protocol can enforce different limits.
func ReadMessage(r io.Reader, m proto.Message, maxSize int) (int, error) {
	length, headerLen, err := readUvarint(r)
	if err != nil {
		return headerLen, err
	}
	if length > uint64(maxSize) {
		return headerLen, fmt.Errorf("%w: %d > %d", ErrMessageTooLarge, length, maxSize)
	}

	body := make([]byte, length)
	read, err := io.ReadFull(r, body)
	if err != nil {
		return headerLen + read, fmt.Errorf("p2p: failed to read message body: %w", err)
	}
	if err := proto.Unmarshal(body, m); err != nil {
		return headerLen + read, fmt.Errorf("p2p: failed to unmarshal message: %w", err)
	}
	return headerLen + read, nil
}

// readUvarint reads a base-128 varint from r one byte at a time (streams do not
// implement io.ByteReader), returning the value and the number of bytes read.
func readUvarint(r io.Reader) (uint64, int, error) {
	var value uint64
	var shift uint
	var buf [1]byte
	for i := range binary.MaxVarintLen64 {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, i, err
		}
		b := buf[0]
		if b < 0x80 {
			return value | uint64(b)<<shift, i + 1, nil
		}
		value |= uint64(b&0x7f) << shift
		shift += 7
	}
	return 0, binary.MaxVarintLen64, errors.New("p2p: message length prefix overflows 64 bits")
}
