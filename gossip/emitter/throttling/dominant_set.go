package throttling

import (
	"cmp"
	"slices"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

type ValidatorStake struct {
	Id    idx.ValidatorID
	Stake pos.Weight
}

func PartialSum(stakes []ValidatorStake) []uint64 {
	res := make([]uint64, len(stakes))
	accumulated := uint64(0)
	for i, stake := range stakes {
		accumulated += uint64(stake.Stake)
		res[i] = accumulated
	}
	return res
}

func ComputeDominantSet(stakes []ValidatorStake, thresholdStake uint64) (map[idx.ValidatorID]pos.Weight, bool) {
	slices.SortFunc(stakes, func(a, b ValidatorStake) int {
		return cmp.Compare(uint64(b.Stake), uint64(a.Stake))
	})

	res := make(map[idx.ValidatorID]pos.Weight)
	partial := PartialSum(stakes)
	for i, sum := range partial {
		res[stakes[i].Id] = stakes[i].Stake
		if sum >= thresholdStake {
			return res, true
		}
	}

	return nil, false
}
