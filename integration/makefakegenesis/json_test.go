// Copyright 2026 Sonic Operations Ltd
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
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestJsonGenesis_CanApplyGeneratedFakeJsonGensis(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(opera.GetSonicUpgrades(), CreateEqualValidatorStake(1))
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_AcceptsGenesisWithoutCommittee(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(opera.GetSonicUpgrades(), CreateEqualValidatorStake(1))
	genesis.GenesisCommittee = nil
	_, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
}

func TestJsonGenesis_Network_RulesValidated_WithAllegroAndLater(t *testing.T) {

	expectNotValidated := func(t *testing.T, err error) {
		require.NoError(t, err)
	}

	expectValidated := func(t *testing.T, err error) {
		require.Error(t, err)
		require.Contains(t, err.Error(), "LLR upgrade is not supported")
	}

	for name, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {

			genesis := GenerateFakeJsonGenesis(upgrades, CreateEqualValidatorStake(1))
			genesis.Rules.Upgrades.Llr = true // LLR is not supported in any hardfork
			_, err := ApplyGenesisJson(genesis)

			// Validation of network rules introduced in Allegro
			if name == "Sonic" {
				expectNotValidated(t, err)
			} else {
				expectValidated(t, err)
			}
		})
	}
}

func TestJsonGenesis_GetGenesisIdFromJson(t *testing.T) {
	genesis := GenerateFakeJsonGenesis(opera.GetSonicUpgrades(), CreateEqualValidatorStake(1))

	store, err := ApplyGenesisJson(genesis)
	require.NoError(t, err)
	want := common.Hash(store.Genesis().GenesisID)

	got, err := GetGenesisIdFromJson(genesis)
	require.NoError(t, err)
	require.NotZero(t, got)

	require.Equal(t, want, got, "unexpected genesis ID")
}

func TestJsonGenesis_GetGenesisIdFromJson_ReportsErrorsFromApplyGenesis(t *testing.T) {

	genesis := GenerateFakeJsonGenesis(opera.GetSonicUpgrades(), CreateEqualValidatorStake(1))
	genesis.BlockZeroTime = time.Time{} // invalid time

	_, err := GetGenesisIdFromJson(genesis)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply genesis json")
}
