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
		if fs.String() != expected {
			t.Errorf("Expected %s, got %s", expected, fs.String())
		}
	}
}

func TestFeatureSet_CanBeConvertedToUpgrades(t *testing.T) {

	tests := map[FeatureSet]struct {
		expectedUpgrades Upgrades
		expectedErr      string
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
		FeatureSet(math.MaxInt): {
			expectedErr: "unknown feature set",
		},
	}

	for featureSet, test := range tests {
		t.Run(featureSet.String(), func(t *testing.T) {
			got, err := featureSet.ToUpgrades()

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedUpgrades, got)
			}
		})

	}
}
