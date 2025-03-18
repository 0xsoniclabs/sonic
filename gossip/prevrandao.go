package gossip

import (
	"crypto/sha256"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
)

// computePrevRandao computes the prevRandao from event hashes.
func computePrevRandao(events []hash.Event) common.Hash {
	bts := [24]byte{}
	for _, event := range events {
		for i := 0; i < 24; i++ {
			// first 8 bytes should be ignored as they are not pseudo-random.
			bts[i] = bts[i] ^ event[i+8]
		}
	}
	return sha256.Sum256(bts[:])
}
