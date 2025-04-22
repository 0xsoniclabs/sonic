package inter

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// Turn is the turn number of a proposal. Turns are used to orchestrate the
// sequence of block proposals in the consensus protocol. Turns are processed
// in order. A turn ends with a proposer making a proposal or a timeout.
type Turn uint32

// GetProposer returns the designated proposer for a given turn.
// The proposer is determined through deterministic sampling of validators
// proportional to the validator's stake.
func GetProposer(
	validators *pos.Validators,
	turn Turn,
) (idx.ValidatorID, error) {

	// The selection of the proposer for a given round is conducted as follows:
	//  1. f := sha256(turn) / 2^256
	//  2. limit := f * total_weight
	//  3. from the list of validators sorted by their stake, find the first
	//     validator whose cumulative weight is greater than or equal to limit.

	// -- Preconditions --
	ids := validators.SortedIDs()
	if len(ids) == 0 {
		return 0, fmt.Errorf("no validators")
	}

	// Note that we use big.Rat to preserve precision in the division.
	// limit := (sha256(turn) * total_weight) / 2^256
	data := make([]byte, 0, 4)
	data = binary.BigEndian.AppendUint32(data, uint32(turn))
	hash := sha256.Sum256(data)
	limit := new(big.Rat).Quo(
		new(big.Rat).SetInt(
			new(big.Int).Mul(
				new(big.Int).SetBytes(hash[:]),
				big.NewInt(int64(validators.TotalWeight())),
			),
		),
		new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 256)),
	)

	// Walk through the validators sorted by their stake (and ID as a tiebreaker)
	// and accumulate their weights until we reach the limit calculated above.
	res := ids[0]
	cumulated := big.NewRat(0, 1)
	for i, weight := range validators.SortedWeights() {
		cumulated.Num().Add(cumulated.Num(), big.NewInt(int64(weight)))
		if cumulated.Cmp(limit) >= 0 {
			res = ids[i]
			break
		}
	}
	return res, nil
}
