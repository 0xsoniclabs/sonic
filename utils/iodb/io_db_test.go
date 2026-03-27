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

package iodb

import (
	"bytes"
	"io"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
)

func encodeEntry(key, value []byte) []byte {
	var buf []byte
	buf = append(buf, bigendian.Uint32ToBytes(uint32(len(key)))...)
	buf = append(buf, key...)
	buf = append(buf, bigendian.Uint32ToBytes(uint32(len(value)))...)
	buf = append(buf, value...)
	return buf
}

func TestIterator_Empty(t *testing.T) {
	reader := bytes.NewReader(nil)
	it := NewIterator(reader)
	if it.Next() {
		t.Fatal("expected Next to return false on empty reader")
	}
	if it.Error() != nil {
		t.Fatalf("expected no error, got %v", it.Error())
	}
}

func TestIterator_SingleEntry(t *testing.T) {
	data := encodeEntry([]byte("key1"), []byte("val1"))
	it := NewIterator(bytes.NewReader(data))

	if !it.Next() {
		t.Fatal("expected Next to return true")
	}
	if string(it.Key()) != "key1" {
		t.Fatalf("expected key1, got %s", it.Key())
	}
	if string(it.Value()) != "val1" {
		t.Fatalf("expected val1, got %s", it.Value())
	}
	if it.Next() {
		t.Fatal("expected Next to return false after last entry")
	}
	if it.Error() != nil {
		t.Fatalf("unexpected error: %v", it.Error())
	}
}

func TestIterator_MultipleEntries(t *testing.T) {
	var data []byte
	data = append(data, encodeEntry([]byte("a"), []byte("1"))...)
	data = append(data, encodeEntry([]byte("bb"), []byte("22"))...)
	data = append(data, encodeEntry([]byte("ccc"), []byte("333"))...)

	it := NewIterator(bytes.NewReader(data))

	expected := []struct {
		key, value string
	}{
		{"a", "1"},
		{"bb", "22"},
		{"ccc", "333"},
	}

	for i, exp := range expected {
		if !it.Next() {
			t.Fatalf("expected Next to return true for entry %d", i)
		}
		if string(it.Key()) != exp.key {
			t.Fatalf("entry %d: expected key %q, got %q", i, exp.key, it.Key())
		}
		if string(it.Value()) != exp.value {
			t.Fatalf("entry %d: expected value %q, got %q", i, exp.value, it.Value())
		}
	}

	if it.Next() {
		t.Fatal("expected Next to return false after all entries")
	}
	if it.Error() != nil {
		t.Fatalf("unexpected error: %v", it.Error())
	}
}

func TestIterator_EmptyKeyAndValue(t *testing.T) {
	data := encodeEntry([]byte{}, []byte{})
	it := NewIterator(bytes.NewReader(data))

	if !it.Next() {
		t.Fatal("expected Next to return true")
	}
	if len(it.Key()) != 0 {
		t.Fatalf("expected empty key, got %v", it.Key())
	}
	if len(it.Value()) != 0 {
		t.Fatalf("expected empty value, got %v", it.Value())
	}
}

func TestIterator_TruncatedKeyLength(t *testing.T) {
	// Only 2 bytes of a 4-byte length field
	data := []byte{0x00, 0x01}
	it := NewIterator(bytes.NewReader(data))

	if it.Next() {
		t.Fatal("expected Next to return false on truncated data")
	}
	// EOF on the first read is treated as end-of-stream, not error
	// but partial reads result in io.EOF or io.ErrUnexpectedEOF
	_ = it.Error()
}

func TestIterator_TruncatedKey(t *testing.T) {
	// Says key is 10 bytes but only has 2
	var data []byte
	data = append(data, bigendian.Uint32ToBytes(10)...)
	data = append(data, []byte("ab")...)
	it := NewIterator(bytes.NewReader(data))

	if it.Next() {
		t.Fatal("expected Next to return false on truncated key")
	}
	if it.Error() == nil {
		t.Fatal("expected an error for truncated key")
	}
}

func TestIterator_TruncatedValueLength(t *testing.T) {
	// Valid key, but truncated value length
	var data []byte
	data = append(data, bigendian.Uint32ToBytes(2)...)
	data = append(data, []byte("ab")...)
	data = append(data, []byte{0x00}...) // incomplete 4-byte length
	it := NewIterator(bytes.NewReader(data))

	if it.Next() {
		t.Fatal("expected Next to return false")
	}
	if it.Error() == nil {
		t.Fatal("expected an error for truncated value length")
	}
}

func TestIterator_TruncatedValue(t *testing.T) {
	// Valid key + value length, but truncated value
	var data []byte
	data = append(data, bigendian.Uint32ToBytes(2)...)
	data = append(data, []byte("ab")...)
	data = append(data, bigendian.Uint32ToBytes(10)...)
	data = append(data, []byte("cd")...)
	it := NewIterator(bytes.NewReader(data))

	if it.Next() {
		t.Fatal("expected Next to return false")
	}
	if it.Error() == nil {
		t.Fatal("expected an error for truncated value")
	}
}

func TestIterator_Release(t *testing.T) {
	it := NewIterator(bytes.NewReader(nil))
	it.Release()
	// After release, reader should be nil
	internal := it.(*Iterator)
	if internal.reader != nil {
		t.Fatal("expected reader to be nil after Release")
	}
}

func TestIterator_ErrorPersists(t *testing.T) {
	it := &Iterator{
		reader: &errorReader{},
		err:    nil,
	}
	// First Next fails due to error
	if it.Next() {
		t.Fatal("expected Next to return false on error reader")
	}
	if it.Error() == nil {
		t.Fatal("expected error to be set")
	}
	// Subsequent Next should also return false
	if it.Next() {
		t.Fatal("expected Next to return false when error is set")
	}
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
