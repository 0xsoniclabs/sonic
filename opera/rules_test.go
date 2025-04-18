package opera

import (
	"reflect"
	"testing"
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a deep copy of the original rules
			original := MainNetRules()
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
