package makefakegenesis

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestJsonGenesis_CanApplyGeneratedFakeJsonGensis(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.SonicFeatures, nil)
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_AcceptsGenesisWithoutCommittee(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, opera.SonicFeatures, nil)
	genesis.GenesisCommittee = nil
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}
