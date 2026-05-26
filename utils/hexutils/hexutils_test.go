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
	"os"
	"os/exec"
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
			result := HexToBytes(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHexToBytes_InvalidInput(t *testing.T) {
	if os.Getenv("TEST_FATAL") == "1" {
		HexToBytes("zz")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestHexToBytes_InvalidInput")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")
	err := cmd.Run()
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.False(t, exitErr.Success())
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
			b := HexToBytes(tt.hex)
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
			result := HexToBytes(hexStr)
			require.Equal(t, tt.input, result)
		})
	}
}
