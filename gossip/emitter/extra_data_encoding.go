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
	"bytes"
	"encoding/binary"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/ethereum/go-ethereum/common"
)

// ErrInvalidExtraData is returned when parsing of extra data failed.
const ErrInvalidExtraData = cc.ConstError("invalid extra data format")

// BlockNumberAndHash contains a pair of a block number and a block hash, as
// it may be encoded in the extra data of an event emitted by validators.
type BlockNumberAndHash struct {
	Number uint64
	Hash   common.Hash
}

// DecodeExtraData decodes the extra data of an event emitted by validators,
// which may contain a version string and/or a block number and hash.
// If the format is invalid, an [ErrInvalidExtraData] error is returned.
func DecodeExtraData(data []byte) (
	version *string,
	blockInfo *BlockNumberAndHash,
	err error,
) {
	// Decode the block information, if available.
	if len(data) >= 8+32 && !bytes.Equal(data[:2], []byte("v-")) {
		blockInfo = &BlockNumberAndHash{}
		blockInfo.Number = binary.BigEndian.Uint64(data[:8])
		copy(blockInfo.Hash[:], data[8:40])
		data = data[40:]
	}

	// Decode the version information, if available.
	if len(data) == 0 {
		return nil, blockInfo, nil
	}

	if !bytes.Equal(data[:2], []byte("v-")) {
		return nil, nil, ErrInvalidExtraData
	}
	v := string(data[2:])
	return &v, blockInfo, nil
}

// encodeExtraData encodes the given version and block information into a byte
// slice to be included in the extra data of an event emitted by validators.
func encodeExtraData(
	version *string,
	blockInfo *BlockNumberAndHash,
) []byte {
	// Encoding format:
	//  {[<blockNumber><blockHash>]}{[v-<version>]}
	//  - blockNumber is encoded as a big-endian uint64
	//  - blockHash is encoded as a 32-byte array
	//  - version is encoded as a string prefixed with "v-"
	part1 := make([]byte, 0, 8+32)
	if blockInfo != nil {
		part1 = binary.BigEndian.AppendUint64(part1, blockInfo.Number)
		part1 = append(part1, blockInfo.Hash.Bytes()...)
	}

	var part2 []byte
	if version != nil {
		part2 = append(part2, []byte("v-")...)
		part2 = append(part2, []byte(*version)...)
	}
	return append(part1, part2...)
}
