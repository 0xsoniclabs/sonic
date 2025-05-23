package makefakegenesis

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestJsonGenesis_CanApplyGeneratedFakeJsonGensis(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.Sonic)
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_AcceptsGenesisWithoutCommittee(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.Sonic)
	genesis.GenesisCommittee = nil
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_Network_Rules_Validated_Allegro_Only(t *testing.T) {
	tests := map[string]struct {
		hardFork opera.HardFork
		assert   func(t *testing.T, err error)
	}{
		"sonic": {
			hardFork: opera.Sonic,
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"allegro": {
			hardFork: opera.Allegro,
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "LLR upgrade is not supported")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			genesis := GenerateFakeJsonGenesis(1, test.hardFork)
			genesis.Rules.Upgrades.Llr = true // LLR is not supported in Allegro and Sonic
			_, err := ApplyGenesisJson(genesis)
			test.assert(t, err)
		})
	}
}
