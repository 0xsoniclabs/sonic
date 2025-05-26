package inter

import (
	"fmt"
	"math"
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

	for epoch := range idx.Epoch(5) {
		for turn := range Turn(5) {
			a, err := GetProposer(validators, epoch, turn)
			require.NoError(err)
			b, err := GetProposer(validators, epoch, turn)
			require.NoError(err)
			require.Equal(a, b, "proposer selection is not deterministic")
		}
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
	for epoch := range idx.Epoch(N) {
		for turn := range Turn(N) {
			got, err := GetProposer(validators1, epoch, turn)
			require.NoError(err)
			want = append(want, got)
		}
	}

	for range 10 {
		counter := 0
		for epoch := range idx.Epoch(N) {
			for turn := range Turn(N) {
				got, err := GetProposer(validators2, epoch, turn)
				require.NoError(err)
				require.Equal(got, want[counter])
				counter++
			}
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

	for epoch := range idx.Epoch(5) {
		for turn := range Turn(50) {
			a, err := GetProposer(validators, epoch, turn)
			require.NoError(err)
			require.Equal(idx.ValidatorID(2), a, "unexpected proposer")
		}
	}
}

func TestGetProposer_EmptyValidatorSet_Fails(t *testing.T) {
	require := require.New(t)

	builder := pos.ValidatorsBuilder{}
	validators := builder.Build()

	_, err := GetProposer(validators, 0, 0)
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
		{samples: 50, tolerance: 15},
		{samples: 100, tolerance: 10},
		{samples: 1_000, tolerance: 2.5},
		{samples: 10_000, tolerance: 1.6},
		{samples: 100_000, tolerance: 0.3},
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
				t.Run(fmt.Sprintf("byTurns/samples=%v", size.samples), func(t *testing.T) {
					checkDistribution(
						t, validators, size.samples, size.tolerance,
						func(i int) (idx.ValidatorID, error) {
							return GetProposer(validators, 0, Turn(i))
						},
					)
				})

				t.Run(fmt.Sprintf("byEpochs/samples=%v", size.samples), func(t *testing.T) {
					checkDistribution(
						t, validators, size.samples, size.tolerance,
						func(i int) (idx.ValidatorID, error) {
							return GetProposer(validators, idx.Epoch(i), 0)
						},
					)
				})

				t.Run(fmt.Sprintf("byTurnsAndEpochs/samples=%v", size.samples), func(t *testing.T) {
					sqrt := int(math.Pow(float64(size.samples), 0.5))
					checkDistribution(
						t, validators, size.samples, size.tolerance,
						func(i int) (idx.ValidatorID, error) {
							epoch := idx.Epoch(i / sqrt)
							turn := Turn(i % sqrt)
							return GetProposer(validators, epoch, turn)
						},
					)
				})
			}
		})
	}
}

func checkDistribution(
	t *testing.T,
	validators *pos.Validators,
	samples int,
	tolerance float64,
	get func(i int) (idx.ValidatorID, error),
) {
	t.Helper()
	require := require.New(t)
	t.Parallel()
	counters := map[idx.ValidatorID]int{}
	for i := range samples {
		proposer, err := get(i)
		require.NoError(err)
		require.True(validators.Exists(proposer))
		counters[proposer]++
	}

	tolerance = float64(samples) * tolerance / 100
	total := int(validators.TotalWeight())
	for id, idx := range validators.Idxs() {
		weight := int(validators.GetWeightByIdx(idx))
		expected := samples * weight / total
		require.InDelta(
			counters[id], expected, tolerance,
			"validator %d is not selected proportional to stake", id,
		)
	}
}
