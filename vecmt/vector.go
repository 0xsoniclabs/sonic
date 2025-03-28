package vecmt

import (
	"encoding/binary"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/consensus/vecengine"

	"github.com/0xsoniclabs/sonic/inter"
)

/*
 * Use binary form for optimization, to avoid serialization. As a result, DB cache works as elements cache.
 */

type (
	// HighestBeforeTime is a vector of highest events (their CreationTime) which are observed by source event
	HighestBeforeTime []byte

	HighestBefore struct {
		VSeq  *vecengine.HighestBeforeSeq
		VTime *HighestBeforeTime
	}
)

// NewHighestBefore creates new HighestBefore vector.
func NewHighestBefore(size consensus.ValidatorIndex) *HighestBefore {
	return &HighestBefore{
		VSeq:  vecengine.NewHighestBeforeSeq(size),
		VTime: NewHighestBeforeTime(size),
	}
}

// NewHighestBeforeTime creates new HighestBeforeTime vector.
func NewHighestBeforeTime(size consensus.ValidatorIndex) *HighestBeforeTime {
	b := make(HighestBeforeTime, size*8)
	return &b
}

// Get i's position in the byte-encoded vector clock
func (b HighestBeforeTime) Get(i consensus.ValidatorIndex) inter.Timestamp {
	for i >= b.Size() {
		return 0
	}
	return inter.Timestamp(binary.LittleEndian.Uint64(b[i*8 : (i+1)*8]))
}

// Set i's position in the byte-encoded vector clock
func (b *HighestBeforeTime) Set(i consensus.ValidatorIndex, time inter.Timestamp) {
	for i >= b.Size() {
		// append zeros if exceeds size
		*b = append(*b, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	}
	binary.LittleEndian.PutUint64((*b)[i*8:(i+1)*8], uint64(time))
}

// Size of the vector clock
func (b HighestBeforeTime) Size() consensus.ValidatorIndex {
	return consensus.ValidatorIndex(len(b) / 8)
}
