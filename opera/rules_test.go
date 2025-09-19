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
	"math/big"
	"reflect"
	"testing"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

func TestRules_Copy_CopiesAreDisjoint(t *testing.T) {
	tests := map[string]struct {
		update func(rule *Rules)
	}{
		"update Name": {
			update: func(rule *Rules) {
				rule.Name = "updated-main"
			},
		},
		"update NetworkID": {
			update: func(rule *Rules) {
				rule.NetworkID = 12345
			},
		},
		"update Blocks.MaxBlockGas": {
			update: func(rule *Rules) {
				rule.Blocks.MaxBlockGas = 2 * rule.Blocks.MaxBlockGas
			},
		},
		"update Blocks.MaxEmptyBlockSkipPeriod": {
			update: func(rule *Rules) {
				rule.Blocks.MaxEmptyBlockSkipPeriod = 2 * rule.Blocks.MaxEmptyBlockSkipPeriod
			},
		},
		"update Economy.MinGasPrice": {
			update: func(rule *Rules) {
				rule.Economy.MinGasPrice.SetInt64(2 * rule.Economy.MinGasPrice.Int64())
			},
		},
		"update Economy.MinBaseFee": {
			update: func(rule *Rules) {
				rule.Economy.MinBaseFee.SetInt64(2 * rule.Economy.MinBaseFee.Int64())
			},
		},
		"update Economy.BlockMissedSlack": {
			update: func(rule *Rules) {
				rule.Economy.BlockMissedSlack = 2 * rule.Economy.BlockMissedSlack
			},
		},
		"update Economy.Gas.MaxEventGas": {
			update: func(rule *Rules) {
				rule.Economy.Gas.MaxEventGas = 2 * rule.Economy.Gas.MaxEventGas
			},
		},
		"update Economy.Gas.EventGas": {
			update: func(rule *Rules) {
				rule.Economy.Gas.EventGas = 2 * rule.Economy.Gas.EventGas
			},
		},
		"update Economy.Gas.ParentGas": {
			update: func(rule *Rules) {
				rule.Economy.Gas.ParentGas = 2 * rule.Economy.Gas.ParentGas
			},
		},
		"update Economy.Gas.ExtraDataGas": {
			update: func(rule *Rules) {
				rule.Economy.Gas.ExtraDataGas = 2 * rule.Economy.Gas.ExtraDataGas
			},
		},
		"update Economy.ShortGasPower.AllocPerSec": {
			update: func(rule *Rules) {
				rule.Economy.ShortGasPower.AllocPerSec = 2 * rule.Economy.ShortGasPower.AllocPerSec
			},
		},
		"update Economy.ShortGasPower.MaxAllocPeriod": {
			update: func(rule *Rules) {
				rule.Economy.ShortGasPower.MaxAllocPeriod = 2 * rule.Economy.ShortGasPower.MaxAllocPeriod
			},
		},
		"update Economy.ShortGasPower.StartupAllocPeriod": {
			update: func(rule *Rules) {
				rule.Economy.ShortGasPower.StartupAllocPeriod = 2 * rule.Economy.ShortGasPower.StartupAllocPeriod
			},
		},
		"update Economy.ShortGasPower.MinStartupGas": {
			update: func(rule *Rules) {
				rule.Economy.ShortGasPower.MinStartupGas = 2 * rule.Economy.ShortGasPower.MinStartupGas
			},
		},
		"update Economy.LongGasPower.AllocPerSec": {
			update: func(rule *Rules) {
				rule.Economy.LongGasPower.AllocPerSec = 2 * rule.Economy.LongGasPower.AllocPerSec
			},
		},
		"update Economy.LongGasPower.MaxAllocPeriod": {
			update: func(rule *Rules) {
				rule.Economy.LongGasPower.MaxAllocPeriod = 2 * rule.Economy.LongGasPower.MaxAllocPeriod
			},
		},
		"update Economy.LongGasPower.StartupAllocPeriod": {
			update: func(rule *Rules) {
				rule.Economy.LongGasPower.StartupAllocPeriod = 2 * rule.Economy.LongGasPower.StartupAllocPeriod
			},
		},
		"update Economy.LongGasPower.MinStartupGas": {
			update: func(rule *Rules) {
				rule.Economy.LongGasPower.MinStartupGas = 2 * rule.Economy.LongGasPower.MinStartupGas
			},
		},
		"update Dag.MaxParents": {
			update: func(rule *Rules) {
				rule.Dag.MaxParents = 2 * rule.Dag.MaxParents
			},
		},
		"update Dag.MaxFreeParents": {
			update: func(rule *Rules) {
				rule.Dag.MaxFreeParents = 2 * rule.Dag.MaxFreeParents
			},
		},
		"update Dag.MaxExtraData": {
			update: func(rule *Rules) {
				rule.Dag.MaxExtraData = 2 * rule.Dag.MaxExtraData
			},
		},
		"update Emitter.Interval": {
			update: func(rule *Rules) {
				rule.Emitter.Interval = 2 * rule.Emitter.Interval
			},
		},
		"update Emitter.StallThreshold": {
			update: func(rule *Rules) {
				rule.Emitter.StallThreshold = 2 * rule.Emitter.StallThreshold
			},
		},
		"update Emitter.StalledInterval": {
			update: func(rule *Rules) {
				rule.Emitter.StalledInterval = 2 * rule.Emitter.StalledInterval
			},
		},
		"update Epochs.MaxEpochGas": {
			update: func(rule *Rules) {
				rule.Epochs.MaxEpochGas = 2 * rule.Epochs.MaxEpochGas
			},
		},
		"update Epochs.MaxEpochDuration": {
			update: func(rule *Rules) {
				rule.Epochs.MaxEpochDuration = 2 * rule.Epochs.MaxEpochDuration
			},
		},
		"update Upgrades.Berlin": {
			update: func(rule *Rules) {
				rule.Upgrades.Berlin = !rule.Upgrades.Berlin
			},
		},
		"update Upgrades.London": {
			update: func(rule *Rules) {
				rule.Upgrades.London = !rule.Upgrades.London
			},
		},
		"update Upgrades.Sonic": {
			update: func(rule *Rules) {
				rule.Upgrades.Sonic = !rule.Upgrades.Sonic
			},
		},
		"update Upgrades.Allegro": {
			update: func(rule *Rules) {
				rule.Upgrades.Allegro = !rule.Upgrades.Allegro
			},
		},
		"update Upgrades.SingleProposerBlockFormation": {
			update: func(rule *Rules) {
				rule.Upgrades.SingleProposerBlockFormation = !rule.Upgrades.SingleProposerBlockFormation
			},
		},
		"upgrade Upgrades.Brio": {
			update: func(rule *Rules) {
				rule.Upgrades.Brio = !rule.Upgrades.Brio
			},
		},
		"upgrade Upgrades.GasSubsidies": {
			update: func(rule *Rules) {
				rule.Upgrades.GasSubsidies = !rule.Upgrades.GasSubsidies
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a deep copy of the original rules
			original := FakeNetRules(GetAllegroUpgrades())
			copied := original.Copy()

			// Apply the update to the copied rules
			test.update(&copied)

			// check that the original and copied rules are not the same
			if got, want := original, copied; reflect.DeepEqual(got, want) {
				t.Errorf("original and copied rules are the same: got %v, want %v", got, want)
			}
		})
	}
}

func TestRules_MinBaseFee_NoCopy_PreAllegro(t *testing.T) {
	original := FakeNetRules(GetSonicUpgrades())
	copied := original.Copy()

	copied.Economy.MinBaseFee.SetInt64(2 * copied.Economy.MinBaseFee.Int64())

	if got, want := original.Economy.MinBaseFee.Int64(), copied.Economy.MinBaseFee.Int64(); got != want {
		t.Errorf("original and copied rules must be the same - shallow copy for preAllegro: got %d, want %d", got, want)
	}
}

func TestCreateTransientEvmChainConfig_ContainsUpgradesBasedOnConstructionTimeBlockHeigh(t *testing.T) {

	chainID := uint64(12345)

	tests := map[string]Upgrades{
		"Sonic":   GetSonicUpgrades(),
		"Allegro": GetAllegroUpgrades(),
	}

	for name, upgrades := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			timestamp := uint64(1)
			blockNumber := uint64(123)
			upgradeHeight := UpgradeHeight{
				Upgrades: upgrades,
				Height:   idx.Block(blockNumber),
			}
			// transient chain config is statically configured independent of block height
			// so we can use any block height for the test.
			anyBlockHeigh := big.NewInt(0)

			// test upgrades at the same height where the upgrade is enabled
			chainConfigAfterUpdate := CreateTransientEvmChainConfig(chainID, []UpgradeHeight{upgradeHeight}, idx.Block(blockNumber))
			require.NotNil(chainConfigAfterUpdate, "chainConfig should not be nil")
			require.True(chainConfigAfterUpdate.IsCancun(anyBlockHeigh, timestamp))
			require.Equal(upgrades.Allegro, chainConfigAfterUpdate.IsPrague(anyBlockHeigh, timestamp), "Allegro upgrade should match")

			// test upgrades at a height before the upgrade was enabled
			chainConfigBeforeUpdate := CreateTransientEvmChainConfig(chainID, []UpgradeHeight{upgradeHeight}, idx.Block(blockNumber-1))
			require.NotNil(chainConfigBeforeUpdate, "chainConfig should not be nil")
			require.True(chainConfigBeforeUpdate.IsCancun(anyBlockHeigh, timestamp), "Before Allegro upgrade, Cancun should be true")
			require.False(chainConfigBeforeUpdate.IsPrague(anyBlockHeigh, timestamp), "Before Allegro upgrade, Prague should be false")
		})
	}
}

func TestCreateTransientEvmChainConfig_RespectsBlockHeightOfUpgradeHeight(t *testing.T) {

	// update this test with upgrades which expose feature flags in the chain config
	upgrades := []Upgrades{
		GetSonicUpgrades(),
		GetAllegroUpgrades(),
		{
			Allegro: true,
			Brio:    true,
		},
	}

	var upgradeHeights []UpgradeHeight
	for i, upgrade := range upgrades {
		upgradeHeights = append(upgradeHeights, UpgradeHeight{
			Upgrades: upgrade,
			Height:   idx.Block(i),
		})
	}
	anyBlockHeigh := big.NewInt(0)

	for testUpgradeHeights := range utils.Permute(upgradeHeights) {

		t.Run("Permutation", func(t *testing.T) {

			t.Run("SonicUpgrades", func(t *testing.T) {
				require := require.New(t)
				chainConfig := CreateTransientEvmChainConfig(
					12345,
					testUpgradeHeights,
					idx.Block(0),
				)

				require.True(chainConfig.IsCancun(anyBlockHeigh, 0), "Sonic upgrades should be Cancun")
				require.False(chainConfig.IsPrague(anyBlockHeigh, 0), "Sonic upgrades should not be Prague")
				require.False(chainConfig.IsOsaka(anyBlockHeigh, 0), "Sonic upgrades should not be Prague")
			})

			t.Run("AllegroUpgrades", func(t *testing.T) {
				require := require.New(t)
				chainConfig := CreateTransientEvmChainConfig(
					12345,
					testUpgradeHeights,
					idx.Block(1),
				)

				require.True(chainConfig.IsCancun(anyBlockHeigh, 0), "Allegro upgrades should be Cancun")
				require.True(chainConfig.IsPrague(anyBlockHeigh, 0), "Allegro upgrades should be Prague")
				require.False(chainConfig.IsOsaka(anyBlockHeigh, 0), "Allegro upgrades should not be Osaka")
			})

			t.Run("BrioUpgrades", func(t *testing.T) {
				require := require.New(t)
				chainConfig := CreateTransientEvmChainConfig(
					12345,
					testUpgradeHeights,
					idx.Block(2),
				)

				require.True(chainConfig.IsCancun(anyBlockHeigh, 0), "Brio upgrades should be Cancun")
				require.True(chainConfig.IsPrague(anyBlockHeigh, 0), "Brio upgrades should be Prague")
				require.True(chainConfig.IsOsaka(anyBlockHeigh, 0), "Brio upgrades should be Osaka")
			})
		})
	}
}
