package opera

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureSet_CanBeConvertedToString(t *testing.T) {

	tests := map[string]FeatureSet{
		"sonic":   SonicFeatures,
		"allegro": AllegroFeatures,
		"unknown": FeatureSet(math.MaxInt),
	}

	for expected, fs := range tests {
		require.Equal(t, expected, fs.String())
	}
}

func TestFeatureSet_CanBeConvertedToUpgrades(t *testing.T) {

	tests := map[FeatureSet]struct {
		expectedUpgrades Upgrades
	}{
		SonicFeatures: {
			expectedUpgrades: Upgrades{
				Berlin:  true,
				London:  true,
				Llr:     false,
				Sonic:   true,
				Allegro: false,
			},
		},
		AllegroFeatures: {
			expectedUpgrades: Upgrades{
				Berlin:  true,
				London:  true,
				Llr:     false,
				Sonic:   true,
				Allegro: true,
			},
		},
	}

	for featureSet, test := range tests {
		t.Run(featureSet.String(), func(t *testing.T) {
			got := featureSet.ToUpgrades()
			require.Equal(t, test.expectedUpgrades, got)
		})
	}
}

func TestFeatureSet_ToUpgradesReturnsDefaultIfUnknown(t *testing.T) {
	fs := FeatureSet(math.MaxInt)
	expected := Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   false,
		Allegro: false,
	}

	got := fs.ToUpgrades()
	require.Equal(t, expected, got)
}
