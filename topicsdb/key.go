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

package topicsdb

import (
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/common"
)

const (
	uint8Size  = 1
	uint64Size = 8
	hashSize   = common.HashLength

	// logrecKeySize is the size of the key for a log record in the database.
	// It consists of:
	// - 8 bytes for the block number (uint64)
	// - 32 bytes for the transaction hash (common.Hash)
	// - 8 bytes for the log index (uint64)
	// - 8 bytes for the timestamp (uint64)
	logrecKeySize = uint64Size + hashSize + uint64Size + uint64Size

	// topicKeySize is the size of the key for a topic in the database.
	// It consists of:
	// - 32 bytes for the topic hash (common.Hash)
	// - 1 byte for the position (uint8)
	// - logrecKeySize for the log record key
	topicKeySize = hashSize + uint8Size + logrecKeySize
)

type (
	// ID of log record
	ID [logrecKeySize]byte
)

func NewID(block uint64, tx common.Hash, logIndex uint64, timestamp uint64) (id ID) {
	copy(id[:], uintToBytes(block))
	copy(id[uint64Size:], tx.Bytes())
	copy(id[uint64Size+hashSize:], uintToBytes(uint64(logIndex)))
	copy(id[uint64Size+hashSize+uint64Size:], uintToBytes(timestamp))
	return
}

func (id *ID) Bytes() []byte {
	return (*id)[:]
}

func (id *ID) BlockNumber() uint64 {
	// Block number is stored in the first 8 bytes of the ID
	return bytesToUint((*id)[:uint64Size])
}

func (id *ID) TxHash() (tx common.Hash) {
	// Transaction hash is stored in the bytes from 8 to 40
	copy(tx[:], (*id)[uint64Size:uint64Size+hashSize])
	return
}

func (id *ID) Index() uint {
	// Log index is stored in the bytes from 40 to 48
	return uint(bytesToUint(
		(*id)[uint64Size+hashSize : uint64Size+hashSize+uint64Size]))
}

func (id *ID) Timestamp() uint64 {
	// Timestamp is stored in the last 8 bytes of the ID
	// It starts at index 8 + 32 + 8 = 48
	return bytesToUint((*id)[uint64Size+hashSize+uint64Size : uint64Size+hashSize+uint64Size+uint64Size])
}

func topicKey(topic common.Hash, pos uint8, logrec ID) []byte {
	key := make([]byte, 0, topicKeySize)

	key = append(key, topic.Bytes()...)
	key = append(key, posToBytes(pos)...)
	key = append(key, logrec.Bytes()...)

	return key
}

func posToBytes(pos uint8) []byte {
	return []byte{pos}
}

func bytesToPos(b []byte) uint8 {
	return uint8(b[0])
}

func uintToBytes(n uint64) []byte {
	return bigendian.Uint64ToBytes(n)
}

func bytesToUint(b []byte) uint64 {
	return bigendian.BytesToUint64(b)
}

func extractLogrecID(key []byte) (id ID) {
	switch len(key) {
	case topicKeySize:
		copy(id[:], key[hashSize+uint8Size:])
		return
	default:
		panic("wrong key type")
	}
}
