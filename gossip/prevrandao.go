package gossip

import (
	"crypto/sha256"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/ethereum/go-ethereum/common"
)

// computePrevRandao computes the prevRandao from event hashes.
func computePrevRandao(events []consensus.EventHash) common.Hash {
	bts := [24]byte{}
	for _, event := range events {
		for i := 0; i < 24; i++ {
			// first 8 bytes should be ignored as they are not pseudo-random.
			bts[i] = bts[i] ^ event[i+8]
		}
	}
	return sha256.Sum256(bts[:])
}
