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

package cert

import (
	"fmt"
	"strings"

	"github.com/0xsoniclabs/sonic/utils/jsonhex"
)

// BitSet is a variable-size bit-mask based unsigned integer set representation.
type BitSet[T unsigned] struct {
	mask []byte
}

// Add adds the given entry to the set.
func (b *BitSet[T]) Add(entry T) {
	byteIndex := entry / 8
	bitIndex := entry % 8
	for uint64(len(b.mask)) <= uint64(byteIndex) {
		b.mask = append(b.mask, 0)
	}
	b.mask[byteIndex] |= 1 << uint(bitIndex)
}

// Contains determines whether the given entry is included in the set.
func (b BitSet[T]) Contains(entry T) bool {
	byteIndex := entry / 8
	bitIndex := entry % 8
	if uint64(byteIndex) >= uint64(len(b.mask)) {
		return false
	}
	return b.mask[byteIndex]&(1<<bitIndex) != 0
}

// Entries enumerates all entries in this set. It is intended to be
// used as a generator for a range-based for loop.
func (b BitSet[T]) Entries() []T {
	res := []T{}
	for byteIndex, byte := range b.mask {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			if byte&(1<<uint(bitIndex)) != 0 {
				res = append(res, T(byteIndex)*8+T(bitIndex))
			}
		}
	}
	return res
}

// String produces a human-readable summary of the BitSet mainly for debugging.
func (b BitSet[T]) String() string {
	builder := strings.Builder{}
	builder.WriteString("{")
	for i, entry := range b.Entries() {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("%v", entry))
	}
	builder.WriteString("}")
	return builder.String()
}

// unsigned is used as a class of types accepted as elements for the BitSet.
type unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// MarshalJSON converts the BitSet into a JSON-compatible hex string.
func (b BitSet[T]) MarshalJSON() ([]byte, error) {
	return jsonhex.Bytes(b.mask).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a BitSet.
func (b *BitSet[T]) UnmarshalJSON(data []byte) error {
	hexBytes := jsonhex.Bytes{}
	err := hexBytes.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	b.mask = hexBytes[:]
	return nil
}
