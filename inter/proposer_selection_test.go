package inter

import (
	"fmt"
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

func TestGetProposer_EqualStakes_SelectionIsDeterministic(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	builder.Set(1, 10)
	builder.Set(2, 10)
	validators1 := builder.Build()

	builder = pos.ValidatorsBuilder{}
	builder.Set(2, 10)
	builder.Set(1, 10)
	validators2 := builder.Build()

	const N = 50
	want := []idx.ValidatorID{}
	for turn := range Turn(N) {
		got, err := GetProposer(validators1, turn)
		require.NoError(err)
		want = append(want, got)
	}

	for range 10 {
		for turn := range Turn(N) {
			got, err := GetProposer(validators2, turn)
			require.NoError(err)
			require.Equal(got, want[turn], "proposer selection is not deterministic")
		}
	}
}

func TestGetProposer_ZeroStake_IsIgnored(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	builder.Set(1, 0)
	builder.Set(2, 1)
	validators := builder.Build()

	require.Len(validators.Idxs(), 1, "validator with zero stake should be ignored")

	for turn := range Turn(50) {
		a, err := GetProposer(validators, turn)
		require.NoError(err)
		require.Equal(idx.ValidatorID(2), a, "unexpected proposer")
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
	t.Parallel()

	validators := map[string][]struct {
		id     idx.ValidatorID
		weight pos.Weight
	}{
		"single": {
			{id: 1, weight: 10},
		},
		"two-uniform": {
			{id: 1, weight: 10},
			{id: 2, weight: 10},
		},
		"two-biased": {
			{id: 1, weight: 10},
			{id: 2, weight: 20},
		},
		"four-biased": {
			{id: 1, weight: 10},
			{id: 2, weight: 20},
			{id: 3, weight: 30},
			{id: 4, weight: 40},
		},
		"id-gaps": {
			{id: 17, weight: 123},
			{id: 23, weight: 321},
		},
	}

	sizes := []struct {
		samples   int
		tolerance float64 // in percent
	}{
		{samples: 1, tolerance: 100},
		{samples: 10, tolerance: 20},
		{samples: 50, tolerance: 10},
		{samples: 100, tolerance: 3},
		{samples: 1_000, tolerance: 1.5},
		{samples: 10_000, tolerance: 0.6},
		{samples: 100_000, tolerance: 0.25},
	}

	for name, vals := range validators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			builder := pos.ValidatorsBuilder{}
			for _, v := range vals {
				builder.Set(v.id, v.weight)
			}
			validators := builder.Build()

			for _, size := range sizes {
				t.Run(fmt.Sprintf("samples=%v", size.samples), func(t *testing.T) {
					require := require.New(t)
					t.Parallel()
					counters := map[idx.ValidatorID]int{}
					for turn := range Turn(size.samples) {
						proposer, err := GetProposer(validators, turn)
						require.NoError(err)
						require.True(validators.Exists(proposer))
						counters[proposer]++
					}

					tolerance := float64(size.samples) * size.tolerance / 100
					total := int(validators.TotalWeight())
					for id, idx := range validators.Idxs() {
						weight := int(validators.GetWeightByIdx(idx))
						expected := size.samples * weight / total
						require.InDelta(
							counters[id], expected, tolerance,
							"validator %d is not selected proportional to stake", id,
						)
					}

				})
			}
		})
	}
}
