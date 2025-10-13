// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package makefakegenesis

import (
	"testing"

	sonic "github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestJsonGenesis_CanApplyGeneratedFakeJsonGensis(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, sonic.GetSonicUpgrades())
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_AcceptsGenesisWithoutCommittee(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(1, sonic.GetSonicUpgrades())
	genesis.GenesisCommittee = nil
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_Network_Rules_Validated_Allegro_Only(t *testing.T) {
	tests := map[string]struct {
		featureSet sonic.Upgrades
		assert     func(t *testing.T, err error)
	}{
		"sonic": {
			featureSet: sonic.GetSonicUpgrades(),
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"allegro": {
			featureSet: sonic.GetAllegroUpgrades(),
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
