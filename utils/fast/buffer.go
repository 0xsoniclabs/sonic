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

package fast

type Reader struct {
	buf    []byte
	offset int
}

type Writer struct {
	buf []byte
}

// NewReader wraps bytes with reading buffer.
func NewReader(bb []byte) *Reader {
	return &Reader{
		buf:    bb,
		offset: 0,
	}
}

// NewWriter wraps bytes with writing buffer.
func NewWriter(bb []byte) *Writer {
	return &Writer{
		buf: bb,
	}
}

// MustWriteByte to the buffer.
func (b *Writer) MustWriteByte(v byte) {
	b.buf = append(b.buf, v)
}

// Write the byte to the buffer.
func (b *Writer) Write(v []byte) {
	b.buf = append(b.buf, v...)
}

// Read n bytes.
func (b *Reader) Read(n int) []byte {
	res := b.buf[b.offset : b.offset+n]
	b.offset += n
	return res
}

// MustReadByte reads 1 byte.
func (b *Reader) MustReadByte() byte {
	res := b.buf[b.offset]
	b.offset++
	return res
}

// Position of internal cursor.
func (b *Reader) Position() int {
	return b.offset
}

// Bytes of internal buffer
func (b *Reader) Bytes() []byte {
	return b.buf
}

// Bytes of internal buffer
func (b *Writer) Bytes() []byte {
	return b.buf
}

// Empty returns true if the whole buffer is consumed
func (b *Reader) Empty() bool {
	return len(b.buf) == b.offset
}
