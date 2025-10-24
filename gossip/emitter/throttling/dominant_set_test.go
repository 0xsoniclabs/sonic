package throttling

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func TestDominantSet(t *testing.T) {

	const numValidators = 35

	distribution, err := utils.NewFromMedianAndPercentile(5_000, 0.95, 100_000, nil)
	require.NoError(t, err)
	validatorsStake := make([]ValidatorStake, 0, numValidators)
	totalStake := uint64(0)
	for i := range numValidators {
		stake := uint64(distribution.Sample())
		validatorsStake = append(validatorsStake,
			ValidatorStake{Id: idx.ValidatorID(i), Stake: pos.Weight(stake)})
		totalStake += stake
	}

	thresholdStake := (totalStake*2)/3 + 1
	dominantSet, exist := ComputeDominantSet(validatorsStake, thresholdStake)
	require.True(t, exist, "dominant set should exist")

	stakePercentage := make([]float64, 0, len(validatorsStake))
	for _, v := range validatorsStake {
		stakePercentage = append(stakePercentage, float64(v.Stake)*100/float64(totalStake))
	}
	partialS := PartialSum(validatorsStake)

	fmt.Println("Stakes:  ", validatorsStake)
	fmt.Println("dominant:", dominantSet)
	fmt.Println("partialS:", partialS)
	fmt.Println("Percent: ", stakePercentage)

	sumDominant := uint64(0)
	for _, stake := range dominantSet {
		sumDominant += uint64(stake)
	}
	fmt.Println("Sum dominant stake:", sumDominant, "out of", totalStake, "percentage:", float64(sumDominant)*100/float64(totalStake))
	require.GreaterOrEqual(t, sumDominant, thresholdStake)
}
