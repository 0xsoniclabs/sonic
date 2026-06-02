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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "single byte",
			input:    "ff",
			expected: []byte{0xff},
		},
		{
			name:     "multiple bytes",
			input:    "deadbeef",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:     "uppercase hex",
			input:    "DEADBEEF",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:     "mixed case",
			input:    "DeAdBeEf",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:     "with spaces between bytes",
			input:    "de ad be ef",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:     "with multiple spaces",
			input:    "de  ad  be  ef",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:     "all zeros",
			input:    "00000000",
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "all ones",
			input:    "ffffffff",
			expected: []byte{0xff, 0xff, 0xff, 0xff},
		},
		{
			name:     "single zero byte",
			input:    "00",
			expected: []byte{0x00},
		},
		{
			name:     "long input",
			input:    "0123456789abcdef0123456789abcdef",
			expected: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MustHexToBytes(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHexToBytes_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "non-hex characters", input: "zz"},
		{name: "odd length", input: "abc"},
		{name: "special characters", input: "!@#$"},
		{name: "0x prefix", input: "0xdeadbeef"},
		{name: "newline in string", input: "de\nad"},
		{name: "tab in string", input: "de\tad"},
		{name: "single non-hex char", input: "g"},
		{name: "valid prefix invalid suffix", input: "deadbegf"},
		{name: "unicode characters", input: "café"},
		{name: "trailing garbage", input: "deadbeef!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HexToBytes(tt.input)
			require.Error(t, err)
		})
	}
}

func TestMustHexToBytes_Panics(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "non-hex characters", input: "zz"},
		{name: "odd length", input: "abc"},
		{name: "special characters", input: "!@#$"},
		{name: "0x prefix", input: "0xdeadbeef"},
		{name: "newline in string", input: "de\nad"},
		{name: "tab in string", input: "de\tad"},
		{name: "single non-hex char", input: "g"},
		{name: "valid prefix invalid suffix", input: "deadbegf"},
		{name: "unicode characters", input: "café"},
		{name: "trailing garbage", input: "deadbeef!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Panics(t, func() {
				MustHexToBytes(tt.input)
			})
		})
	}
}

func TestMustHexToBytes_DoesNotPanic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{name: "spaces only", input: "   ", expected: []byte{}},
		{name: "space-separated single byte", input: " ff ", expected: []byte{0xff}},
		{name: "space inside byte pair", input: "d e a d", expected: []byte{0xde, 0xad}},
		{name: "lowercase a-f", input: "abcdef", expected: []byte{0xab, 0xcd, 0xef}},
		{name: "digits only", input: "0123456789", expected: []byte{0x01, 0x23, 0x45, 0x67, 0x89}},
		{name: "32 bytes (address-like)", input: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", expected: []byte{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
			0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				result := MustHexToBytes(tt.input)
				require.Equal(t, tt.expected, result)
			})
		})
	}
}

func TestBytesToHex(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty slice",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "single byte",
			input:    []byte{0xff},
			expected: "FF",
		},
		{
			name:     "multiple bytes",
			input:    []byte{0xde, 0xad, 0xbe, 0xef},
			expected: "DEADBEEF",
		},
		{
			name:     "all zeros",
			input:    []byte{0x00, 0x00, 0x00, 0x00},
			expected: "00000000",
		},
		{
			name:     "all ones",
			input:    []byte{0xff, 0xff, 0xff, 0xff},
			expected: "FFFFFFFF",
		},
		{
			name:     "single zero byte",
			input:    []byte{0x00},
			expected: "00",
		},
		{
			name:     "sequential bytes",
			input:    []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expected: "0123456789ABCDEF",
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToHex(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		hex  string
	}{
		{name: "deadbeef", hex: "DEADBEEF"},
		{name: "empty", hex: ""},
		{name: "single byte", hex: "AB"},
		{name: "all zeros", hex: "00000000"},
		{name: "all ones", hex: "FFFFFFFF"},
		{name: "long value", hex: "0123456789ABCDEF0123456789ABCDEF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := MustHexToBytes(tt.hex)
			result := BytesToHex(b)
			require.Equal(t, tt.hex, result)
		})
	}
}

func TestRoundTripBytesToHexToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "deadbeef", input: []byte{0xde, 0xad, 0xbe, 0xef}},
		{name: "empty", input: []byte{}},
		{name: "single byte", input: []byte{0xab}},
		{name: "all zeros", input: []byte{0x00, 0x00, 0x00, 0x00}},
		{name: "sequential", input: []byte{0x01, 0x02, 0x03, 0x04, 0x05}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexStr := BytesToHex(tt.input)
			result := MustHexToBytes(hexStr)
			require.Equal(t, tt.input, result)
		})
	}
}
