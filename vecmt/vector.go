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

package vecmt

import (
	"encoding/binary"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/vecfc"

	"github.com/0xsoniclabs/sonic/inter"
)

/*
 * Use binary form for optimization, to avoid serialization. As a result, DB cache works as elements cache.
 */

type (
	// HighestBeforeTime is a vector of highest events (their CreationTime) which are observed by source event
	HighestBeforeTime []byte

	HighestBefore struct {
		VSeq  *vecfc.HighestBeforeSeq
		VTime *HighestBeforeTime
	}
)

// NewHighestBefore creates new HighestBefore vector.
func NewHighestBefore(size idx.Validator) *HighestBefore {
	return &HighestBefore{
		VSeq:  vecfc.NewHighestBeforeSeq(size),
		VTime: NewHighestBeforeTime(size),
	}
}

// NewHighestBeforeTime creates new HighestBeforeTime vector.
func NewHighestBeforeTime(size idx.Validator) *HighestBeforeTime {
	b := make(HighestBeforeTime, size*8)
	return &b
}

// Get i's position in the byte-encoded vector clock
func (b HighestBeforeTime) Get(i idx.Validator) inter.Timestamp {
	for i >= b.Size() {
		return 0
	}
	return inter.Timestamp(binary.LittleEndian.Uint64(b[i*8 : (i+1)*8]))
}

// Set i's position in the byte-encoded vector clock
func (b *HighestBeforeTime) Set(i idx.Validator, time inter.Timestamp) {
	for i >= b.Size() {
		// append zeros if exceeds size
		*b = append(*b, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	}
	binary.LittleEndian.PutUint64((*b)[i*8:(i+1)*8], uint64(time))
}

// Size of the vector clock
func (b HighestBeforeTime) Size() idx.Validator {
	return idx.Validator(len(b) / 8)
}
