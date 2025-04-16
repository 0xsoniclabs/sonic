package inter

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func TestGetProposer_IsDeterministic(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	builder.Set(1, 10)
	builder.Set(2, 20)
	builder.Set(3, 30)
	validators := builder.Build()

	for turn := range Turn(5) {
		a, err := GetProposer(validators, turn)
		require.NoError(err)
		b, err := GetProposer(validators, turn)
		require.NoError(err)
		require.Equal(a, b, "proposer selection is not deterministic")
	}
}

func TestGetProposer_EmptyValidatorSet_Fails(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	validators := builder.Build()

	_, err := GetProposer(validators, 0)
	require.ErrorContains(err, "no validators")
}

func TestGetProposer_ProposersAreSelectedProportionalToStake(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	builder.Set(1, 10)
	builder.Set(2, 20)
	builder.Set(3, 30)
	builder.Set(4, 40)
	validators := builder.Build()

	const Samples = 100000
	counters := map[idx.ValidatorID]int{}
	for turn := range Turn(Samples) {
		proposer, err := GetProposer(validators, turn)
		require.NoError(err)
		counters[proposer]++
	}

	tolerance := float64(Samples / 100) // 1% tolerance
	for id, idx := range validators.Idxs() {
		expected := int(Samples * validators.GetWeightByIdx(idx) / validators.TotalWeight())
		require.InDelta(
			counters[id], expected, tolerance,
			"validator %d is not selected proportional to stake", id,
		)
	}
}
