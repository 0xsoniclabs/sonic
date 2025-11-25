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

package opera

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestUpdateRules(t *testing.T) {
	require := require.New(t)

	base := MainNetRules()

	exp := base.Copy()
	exp.Dag.MaxParents = 5
	exp.Economy.MinGasPrice = big.NewInt(7)
	exp.Economy.MinBaseFee = big.NewInt(1e9)
	exp.Blocks.MaxBlockGas = 5000000000

	got, err := UpdateRules(exp, []byte(`{"Dag":{"MaxParents":5},"Economy":{"MinGasPrice":7},"Blocks":{"MaxBlockGas":5000000000}}`))
	require.NoError(err)
	require.Equal(exp.String(), got.String(), "should not be able to change readonly fields")

	got, err = UpdateRules(exp, []byte(`{}`))
	require.NoError(err)
	require.Equal(exp.String(), got.String(), "empty diff changed the rules")

	_, err = UpdateRules(exp, []byte(`}{`))
	require.Error(err, "should fail on invalid json")

	_, err = UpdateRules(exp, []byte(`{"Dag":{"MaxParents":1}}`))
	require.Error(err, "should fail on invalid rules")
}

func TestUpdateRules_ValidityCheckIsConductedIfCheckIsEnabledInUpdatedRuleSet(t *testing.T) {
	for _, enabledBefore := range []bool{true, false} {
		for _, enabledAfter := range []bool{true, false} {
			for _, validUpdate := range []bool{true, false} {
				t.Run(fmt.Sprintf("before=%t,after=%t,valid=%t", enabledBefore, enabledAfter, validUpdate), func(t *testing.T) {
					require := require.New(t)

					base := MainNetRules()
					base.Upgrades.Allegro = enabledBefore

					maxParents := 1
					if validUpdate {
						maxParents = 5
					}

					update := fmt.Sprintf(`{"Dag":{"MaxParents":%d}, "Upgrades":{"Allegro":%t}}`, maxParents, enabledAfter)

					_, err := UpdateRules(base, []byte(update))
					if enabledAfter && !validUpdate {
						require.Error(err)
					} else {
						require.NoError(err)
					}
				})
			}
		}
	}
}

func TestUpdateRules_CanUpdateHardForks(t *testing.T) {

	tests := map[string]struct {
		rules Rules
		diff  []byte
		want  Upgrades
	}{
		"Allegro": {
			rules: FakeNetRules(GetSonicUpgrades()),
			diff:  []byte(`{"Upgrades":{"Allegro":true}}`),
			want: Upgrades{
				Berlin:  true,
				London:  true,
				Sonic:   true,
				Allegro: true,
			},
		},
		"Brio": {
			rules: FakeNetRules(GetAllegroUpgrades()),
			diff:  []byte(`{"Upgrades":{"Brio":true}}`),
			want: Upgrades{
				Berlin:  true,
				London:  true,
				Sonic:   true,
				Allegro: true,
				Brio:    true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			require := require.New(t)

			got, err := UpdateRules(test.rules, test.diff)
			require.NoError(err)
			require.Equal(got.Upgrades, test.want)
		})
	}
}

func TestMainNetRulesRLP(t *testing.T) {
	rules := MainNetRules()
	require := require.New(t)

	b, err := rlp.EncodeToBytes(rules)
	require.NoError(err)

	decodedRules := Rules{}
	require.NoError(rlp.DecodeBytes(b, &decodedRules))

	require.Equal(rules.String(), decodedRules.String())
}

func TestUpgradesRLP_CanBeEncodedAndDecoded(t *testing.T) {
	require := require.New(t)
	setUpgrade := []func(*Upgrades){
		func(u *Upgrades) { u.Berlin = true },
		func(u *Upgrades) { u.London = true },
		func(u *Upgrades) { u.Llr = true },
		func(u *Upgrades) { u.Sonic = true },
		func(u *Upgrades) { u.Allegro = true },
		func(u *Upgrades) { u.SingleProposerBlockFormation = true },
		func(u *Upgrades) { u.Brio = true },
		func(u *Upgrades) { u.GasSubsidies = true },
	}

	for mask := range 1 << len(setUpgrade) {
		upgrades := Upgrades{}
		for i, set := range setUpgrade {
			if mask&(1<<i) != 0 {
				set(&upgrades)
			}
		}

		b, err := rlp.EncodeToBytes(upgrades)
		require.NoError(err)

		decodedUpgrades := Upgrades{}
		require.NoError(rlp.DecodeBytes(b, &decodedUpgrades))

		require.Equal(upgrades, decodedUpgrades)
	}
}

func TestRulesBerlinCompatibilityRLP(t *testing.T) {
	require := require.New(t)

	b1, err := rlp.EncodeToBytes(Upgrades{
		Berlin: true,
	})
	require.NoError(err)

	b2, err := rlp.EncodeToBytes(struct {
		Berlin bool
	}{true})
	require.NoError(err)

	require.Equal(b2, b1)
}

func TestGasRulesLLRCompatibilityRLP(t *testing.T) {
	require := require.New(t)

	b1, err := rlp.EncodeToBytes(GasRules{
		MaxEventGas:          1,
		EventGas:             2,
		ParentGas:            3,
		ExtraDataGas:         4,
		BlockVotesBaseGas:    0,
		BlockVoteGas:         0,
		EpochVoteGas:         0,
		MisbehaviourProofGas: 0,
	})
	require.NoError(err)

	b2, err := rlp.EncodeToBytes(struct {
		MaxEventGas  uint64
		EventGas     uint64
		ParentGas    uint64
		ExtraDataGas uint64
	}{1, 2, 3, 4})
	require.NoError(err)

	require.Equal(b2, b1)
}

func TestUpdateRules_RuleValidationIsPerformedStartingFromAllegro(t *testing.T) {
	hardforks := []string{"Sonic", "Allegro", "Brio"}

	for _, hardfork := range hardforks {
		base := FakeNetRules(GetSonicUpgrades())

		// send an invalid update and enable the hardfork
		update := fmt.Sprintf(`{"Dag":{"MaxParents":1}, "Upgrades":{"%s":true}}`, hardfork)

		_, err := UpdateRules(base, []byte(update))
		if hardfork == "Sonic" {
			require.NoError(t, err, "should not validate rules for Sonic hardfork")
		} else {
			require.Error(t, err, "should validate rules for %s hardfork", hardfork)
		}
	}
}
