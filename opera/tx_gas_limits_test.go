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

package opera

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestMaxTxExecutionGas_LeavesRoomForTheLargestEventOverhead(t *testing.T) {
	rules := Rules{
		Economy: EconomyRules{Gas: GasRules{
			MaxEventGas:  10_000,
			EventGas:     1_000,
			ParentGas:    100,
			ExtraDataGas: 10,
		}},
		Dag: DagRules{MaxParents: 5, MaxFreeParents: 3, MaxExtraData: 4},
	}
	require.Equal(t, uint64(10_000-1_000-2*100-4*10), rules.MaxTxExecutionGas())

	rules.Economy.Gas.MaxEventGas = 1_000
	require.Zero(t, rules.MaxTxExecutionGas())
}

func TestMaxTxGasLimit_FromCantoIsBoundedByTheGasOfASingleBlock(t *testing.T) {
	brio := Upgrades{Brio: true}
	canto := Upgrades{Brio: true, Canto: true}
	singleProposer := Upgrades{Brio: true, Canto: true, SingleProposerBlockFormation: true}

	tests := map[string]struct {
		upgrades    Upgrades
		maxBlockGas uint64
		want        uint64
	}{
		"Brio":                         {upgrades: brio, maxBlockGas: 1_000_000, want: 10_000},
		"Canto":                        {upgrades: canto, maxBlockGas: 1_000_000, want: 1_000_000},
		"Canto, above the total cap":   {upgrades: canto, maxBlockGas: math.MaxUint64, want: params.MaxTxTotalGas},
		"Canto, single proposer":       {upgrades: singleProposer, maxBlockGas: 1_000_000, want: 2 * 1_000},
		"Canto, single proposer, tiny": {upgrades: singleProposer, maxBlockGas: 1_500, want: 1_500},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rules := Rules{
				Upgrades: test.upgrades,
				Economy: EconomyRules{
					Gas:           GasRules{MaxEventGas: 10_000},
					ShortGasPower: GasPowerRules{AllocPerSec: 1_000},
				},
				Blocks: BlocksRules{MaxBlockGas: test.maxBlockGas},
			}
			require.Equal(t, test.want, rules.MaxTxGasLimit())
		})
	}
}
