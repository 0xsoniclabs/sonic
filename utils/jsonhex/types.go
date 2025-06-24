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

package jsonhex

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Bytes represents a byte slice that is serialized as a hexadecimal.
type Bytes []byte

// MarshalJSON converts the Bytes into a JSON-compatible hex string with a "0x" prefix.
func (h Bytes) MarshalJSON() ([]byte, error) {
	return []byte(h.String()), nil
}

// UnmarshalJSON parses a JSON hex string into a Bytes.
// The input string must have a "0x" prefix or be "null".
func (h *Bytes) UnmarshalJSON(data []byte) error {
	var rawData json.RawMessage
	err := json.Unmarshal(data, &rawData)
	s := string(rawData)
	if err != nil {
		return err
	}
	if s == `null` {
		*h = nil
		return nil
	}
	s = strings.Trim(s, `\"`)
	if !strings.HasPrefix(s, "0x") {
		return fmt.Errorf("invalid hex string %s", s)
	}
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*h = raw
	return nil
}

// String returns the hex string representation of Bytes.
func (h Bytes) String() string {
	if h == nil {
		return `null`
	}
	return fmt.Sprintf(`"0x%x"`, []byte(h))
}

// UnmarshalFixLengthBytes decodes a JSON hex string into a fixed-length Bytes slice.
// Returns an error if the decoded length does not match the expected length.
func UnmarshalFixLengthBytes(data []byte, length int) (Bytes, error) {
	var h Bytes
	err := h.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	if len(h) != length {
		return nil, fmt.Errorf("invalid length %d, expected %d", len(h), length)
	}
	return h, nil
}

// Bytes48 is a fixed-size [48]byte array that serializes as a hex string with a "0x" prefix
type Bytes48 [48]byte

// MarshalJSON converts the Bytes48 into a JSON-compatible hex string.
func (h Bytes48) MarshalJSON() ([]byte, error) {
	return Bytes(h[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a Bytes48.
func (h *Bytes48) UnmarshalJSON(data []byte) error {
	Bytes, err := UnmarshalFixLengthBytes(data, 48)
	if err != nil {
		return err
	}
	*h = Bytes48(Bytes)
	return nil
}

// String returns the hex string representation of Bytes48.
func (h Bytes48) String() string {
	return Bytes(h[:]).String()
}

// Bytes96 is a fixed-size [96]byte array that serializes as a hex string with a "0x" prefix.
type Bytes96 [96]byte

// MarshalJSON converts the Bytes96 into a JSON-compatible hex string.
func (h Bytes96) MarshalJSON() ([]byte, error) {
	return Bytes(h[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a Bytes96.
func (h *Bytes96) UnmarshalJSON(data []byte) error {
	Bytes, err := UnmarshalFixLengthBytes(data, 96)
	if err != nil {
		return err
	}
	*h = Bytes96(Bytes)
	return nil
}

// String returns the hex string representation of Bytes96.
func (h Bytes96) String() string {
	return Bytes(h[:]).String()
}
