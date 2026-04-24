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

package emitter

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestDecodeExtraData tests DecodeExtraData for all valid and invalid inputs.
func TestDecodeExtraData(t *testing.T) {
	type want struct {
		version   *string
		blockInfo *BlockNumberAndHash
		err       error
	}

	encodeInfo := func(num uint64, hash common.Hash) []byte {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, num)
		return append(b, hash.Bytes()...)
	}

	var maxHash common.Hash
	for i := range maxHash {
		maxHash[i] = 0xFF
	}

	tests := map[string]struct {
		input []byte
		want  want
	}{
		"empty input returns all nil": {
			input: []byte{},
			want:  want{},
		},
		"version only": {
			input: []byte("v-1.2.3"),
			want:  want{version: ptr("1.2.3")},
		},
		"version with tag only": {
			input: []byte("v-1.2.3-rc.1"),
			want:  want{version: ptr("1.2.3-rc.1")},
		},
		"version with empty suffix": {
			input: []byte("v-"),
			want:  want{version: ptr("")},
		},
		"block info only": {
			input: encodeInfo(42, common.Hash{0xAB}),
			want: want{blockInfo: &BlockNumberAndHash{
				Number: 42,
				Hash:   common.Hash{0xAB},
			}},
		},
		"block info and version": {
			input: append(encodeInfo(7, common.Hash{0x01}), []byte("v-1.0")...),
			want: want{version: ptr("1.0"), blockInfo: &BlockNumberAndHash{
				Number: 7,
				Hash:   common.Hash{0x01},
			}},
		},
		"block number zero": {
			input: encodeInfo(0, common.Hash{}),
			want: want{blockInfo: &BlockNumberAndHash{
				Number: 0,
				Hash:   common.Hash{},
			}},
		},
		"block number max uint64": {
			input: encodeInfo(math.MaxUint64, maxHash),
			want: want{blockInfo: &BlockNumberAndHash{
				Number: math.MaxUint64,
				Hash:   maxHash,
			}},
		},
		"short non-version input returns error": {
			input: []byte{0x01, 0x02, 0x03},
			want:  want{err: ErrInvalidExtraData},
		},
		"exactly 39 bytes without v- prefix returns error": {
			input: make([]byte, 39),
			want:  want{err: ErrInvalidExtraData},
		},
		"more than 40 bytes without v- prefix returns error": {
			input: append(make([]byte, 40), []byte("invalid")...),
			want:  want{err: ErrInvalidExtraData},
		},
		"input starting with v- shorter than 40 bytes is valid version": {
			input: []byte("v-short"),
			want:  want{version: ptr("short")},
		},
		"input starting with v- followed by 40 bytes is valid version": {
			input: append([]byte("v-"), make([]byte, 40)...),
			want:  want{version: ptr(string(make([]byte, 40)))},
		},
		"input starting with v- longer than 40 bytes is valid version": {
			input: append([]byte("v-"), make([]byte, 80)...),
			want:  want{version: ptr(string(make([]byte, 80)))},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			version, blockInfo, err := DecodeExtraData(tt.input)
			if tt.want.err != nil {
				require.ErrorIs(t, err, tt.want.err)
				return
			}
			require.NoError(t, err)
			if tt.want.version != nil {
				require.NotNil(t, version)
				require.Equal(t, *tt.want.version, *version)
			} else {
				require.Nil(t, version)
			}
			require.Equal(t, tt.want.blockInfo, blockInfo)
		})
	}
}

// TestEncodeExtraData tests encodeExtraData for all input combinations.
func TestEncodeExtraData(t *testing.T) {
	tests := map[string]struct {
		version   *string
		blockInfo *BlockNumberAndHash
		want      []byte
	}{
		"both nil produces empty slice": {
			want: []byte{},
		},
		"version only": {
			version: ptr("1.2.3"),
			want:    []byte("v-1.2.3"),
		},
		"empty version string": {
			version: ptr(""),
			want:    []byte("v-"),
		},
		"block info only": {
			blockInfo: &BlockNumberAndHash{
				Number: 42,
				Hash:   common.Hash{0xAB},
			},
			want: func() []byte {
				b := make([]byte, 8)
				b[7] = 42
				return append(b, common.Hash{0xAB}.Bytes()...)
			}(),
		},
		"block info and version": {
			version: ptr("1.0"),
			blockInfo: &BlockNumberAndHash{
				Number: 1,
				Hash:   common.Hash{0x01},
			},
			want: func() []byte {
				b := make([]byte, 8)
				b[7] = 1
				b = append(b, common.Hash{0x01}.Bytes()...)
				return append(b, []byte("v-1.0")...)
			}(),
		},
		"block number zero": {
			blockInfo: &BlockNumberAndHash{
				Number: 0,
				Hash:   common.Hash{0x00},
			},
			want: append(make([]byte, 8), common.Hash{0x00}.Bytes()...),
		},
		"block number max uint64": {
			blockInfo: &BlockNumberAndHash{
				Number: math.MaxUint64,
				Hash:   common.Hash{0xFF},
			},
			want: func() []byte {
				b := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
				return append(b, common.Hash{0xFF}.Bytes()...)
			}(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := encodeExtraData(tt.version, tt.blockInfo)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestEncodeDecodeExtraData verifies that encoding followed by decoding
// reproduces the original inputs (round-trip alignment).
func TestEncodeDecodeExtraData(t *testing.T) {
	tests := map[string]struct {
		version   *string
		blockInfo *BlockNumberAndHash
	}{
		"nil version and nil block info": {},
		"version only": {
			version: ptr("1.2.3"),
		},
		"empty version string": {
			version: ptr(""),
		},
		"block info only": {
			blockInfo: &BlockNumberAndHash{
				Number: 100,
				Hash:   common.Hash{1, 2, 3},
			},
		},
		"version and block info": {
			version: ptr("2.0.0"),
			blockInfo: &BlockNumberAndHash{
				Number: math.MaxUint64,
				Hash: common.Hash{
					0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
					0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
					0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
					0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			encoded := encodeExtraData(tt.version, tt.blockInfo)
			gotVersion, gotBlockInfo, err := DecodeExtraData(encoded)
			require.NoError(t, err)
			require.Equal(t, tt.version, gotVersion)
			require.Equal(t, tt.blockInfo, gotBlockInfo)
		})
	}
}

// ptr is a helper to get a pointer to a value of any type.
func ptr[T any](s T) *T { return &s }
