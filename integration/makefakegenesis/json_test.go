package makefakegenesis

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestJsonGenesis_CanApplyGeneratedFakeJsonGensis(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.GetSonicUpgrades())
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_AcceptsGenesisWithoutCommittee(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.GetSonicUpgrades())
	genesis.GenesisCommittee = nil
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_Network_Rules_Validated_Allegro_Only(t *testing.T) {
	tests := map[string]struct {
		featureSet opera.Upgrades
		assert     func(t *testing.T, err error)
	}{
		"sonic": {
			featureSet: opera.GetSonicUpgrades(),
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"allegro": {
			featureSet: opera.GetAllegroUpgrades(),
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "LLR upgrade is not supported")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			genesis := GenerateFakeJsonGenesis(1, test.featureSet)
			genesis.Rules.Upgrades.Llr = true // LLR is not supported in Allegro and Sonic
			_, err := ApplyGenesisJson(genesis)
			test.assert(t, err)
		})
	}
}
