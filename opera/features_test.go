package opera

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeature_IsPrintedByName(t *testing.T) {
	// Test a few example names to ensure the stringer is working correctly.
	require.Equal(t, "SonicCertificateChain", SonicCertificateChain.String())
	require.Equal(t, "NetworkRuleChecks", NetworkRuleChecks.String())
}

func TestFeatures_NewFeatures_CanCreateSetContainingSelectedFeatures(t *testing.T) {
	tests := map[string][]Feature{
		"empty":    nil,
		"single":   {SonicCertificateChain},
		"multiple": {SonicCertificateChain, NetworkRuleChecks},
	}

	for name, features := range tests {
		t.Run(name, func(t *testing.T) {
			featuresSet := NewFeatures(features...)
			require.Equal(t, features, featuresSet.Features())
		})
	}
}

func TestFeatures_CanHandleLargeFeatureNumbers(t *testing.T) {
	// too large feature number is simply ignored
	features := NewFeatures(Feature(250))
	require.Empty(t, features.Features())
}

func TestFeatures_CanBeEnabledAndDisabled(t *testing.T) {
	features := []Feature{SonicCertificateChain, NetworkRuleChecks}

	for _, feature := range features {
		t.Run(feature.String(), func(t *testing.T) {
			set := NewFeatures()
			require.False(t, set.Has(feature))

			set = set.Enable(feature)
			require.True(t, set.Has(feature))

			set = set.Disable(feature)
			require.False(t, set.Has(feature))
		})
	}
}

func TestFeatures_EnablingToLargeFeatureNumberIsIgnored(t *testing.T) {
	// too large feature number is simply ignored
	features := NewFeatures().Enable(Feature(250))
	require.Empty(t, features.Features())
}

func TestFeatures_DisablingToLargeFeatureNumberIsIgnored(t *testing.T) {
	// too large feature number is simply ignored
	features := NewFeatures().Disable(Feature(250))
	require.Empty(t, features.Features())
}

func TestFeatures_FeaturesArePrintedInAlphabeticalOrder(t *testing.T) {
	features := NewFeatures(NetworkRuleChecks, SonicCertificateChain, EIP7702_SetEoaCode)
	require.Equal(t, "{EIP7702_SetEoaCode,NetworkRuleChecks,SonicCertificateChain}", features.String())
}
