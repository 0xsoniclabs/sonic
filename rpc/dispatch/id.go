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

package dispatch

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ID names a subscription or a polled filter. It has the form of
// go-ethereum's rpc.ID: 16 random bytes in hex without leading zeros, so
// that one cannot be guessed.
type ID string

// NewID creates an id.
func NewID() ID {
	random := make([]byte, 16)
	_, _ = rand.Read(random) // never fails
	digits := strings.TrimLeft(hex.EncodeToString(random), "0")
	if digits == "" {
		digits = "0"
	}
	return ID("0x" + digits)
}

// UnmarshalJSON decodes a string, with the wording of the decoding of
// go-ethereum's rpc.ID.
func (id *ID) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("json: cannot unmarshal %s into Go value of type rpc.ID", jsonKind(data))
	}
	*id = ID(text)
	return nil
}

// jsonKind names the kind of a JSON value as encoding/json's errors do.
func jsonKind(data []byte) string {
	switch strings.TrimSpace(string(data))[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case 't', 'f':
		return "bool"
	}
	return "number"
}
