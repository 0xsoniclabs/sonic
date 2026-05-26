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

package hexutils

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

// HexToBytes converts a hex string to a byte sequence.
// The hex string can have spaces between bytes.
func HexToBytes(s string) []byte {
	s = strings.ReplaceAll(s, " ", "")
	b := make([]byte, hex.DecodedLen(len(s)))
	_, err := hex.Decode(b, []byte(s))
	if err != nil {
		log.Fatal(err)
	}
	return b[:]
}

// BytesToHex returns a hex string of b.
func BytesToHex(b []byte) string {
	return fmt.Sprintf("%X", b)
}
