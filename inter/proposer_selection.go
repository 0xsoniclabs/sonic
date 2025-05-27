package inter

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// GetProposer returns the designated proposer for a given turn.
// The proposer is determined through deterministic sampling of validators
// proportional to the validator's stake.
func GetProposer(
	validators *pos.Validators,
	epoch idx.Epoch,
	turn Turn,
) (idx.ValidatorID, error) {

	// The selection of the proposer for a given round is conducted as follows:
	//  1. f := sha256(epoch || turn) / 2^256, (where || is the concatenation operator)
	//  2. limit := f * total_weight
	//  3. from the list of validators sorted by their stake, find the first
	//     validator whose cumulative weight is greater than or equal to limit.

	// -- Preconditions --
	ids := validators.SortedIDs()
	if len(ids) == 0 {
		return 0, fmt.Errorf("no validators")
	}

	// Note that we use big.Rat to preserve precision in the division.
	// limit := (sha256(epoch || turn) * total_weight) / 2^256
	data := make([]byte, 0, 4+4)
	data = binary.BigEndian.AppendUint32(data, uint32(epoch))
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
