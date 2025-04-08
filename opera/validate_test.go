package opera

import (
	"fmt"
	"github.com/0xsoniclabs/sonic/inter"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultRulesAreValid(t *testing.T) {
	rules := map[string]Rules{
		"mainnet": MainNetRules(),
		"fakenet": FakeNetRules(AllegroFeatures),
	}
	for name, r := range rules {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, r.Validate())
		})
	}
}

func TestValidate_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules Rules
		issue string
	}{
		"dag rules issue": {
			rules: Rules{Dag: DagRules{MaxParents: 1}},
			issue: "Dag.MaxParents is too low",
		},
		"emitter rules issue": {
			rules: Rules{Emitter: EmitterRules{Interval: inter.Timestamp(11 * time.Second)}},
			issue: "Emitter.Interval is too high",
		},
		"epochs rules issue": {
			rules: Rules{Epochs: EpochsRules{MaxEpochDuration: inter.Timestamp(24*time.Hour) + 1}},
			issue: "Epochs.MaxEpochDuration is too high",
		},
		"blocks rules issue": {
			rules: Rules{Blocks: BlocksRules{MaxBlockGas: 0}},
			issue: "Blocks.MaxBlockGas is too low",
		},
		"economy rules issue": {
			rules: Rules{Economy: EconomyRules{Gas: GasRules{MaxEventGas: 1}}},
			issue: "Gas.MaxEventGas is too low",
		},
		"upgrades issue": {
			rules: Rules{Upgrades: Upgrades{Llr: true}},
			issue: "LLR upgrade is not supported",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := test.rules.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestDagRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules DagRules
		issue string
	}{
		"zero parents are not enough": {
			rules: DagRules{MaxParents: 0},
			issue: "MaxParents is too low",
		},
		"one parent is not enough": {
			rules: DagRules{MaxParents: 1},
			issue: "MaxParents is too low",
		},
		"zero free parents are not enough": {
			rules: DagRules{MaxFreeParents: 0},
			issue: "MaxFreeParents is too low",
		},
		"one free parent is not enough": {
			rules: DagRules{MaxFreeParents: 1},
			issue: "MaxFreeParents is too low",
		},
		"too much extra data": {
			rules: DagRules{MaxExtraData: 1<<20 + 1},
			issue: "MaxExtraData is too high",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateDagRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestDagRulesValidation_AcceptsValidRules(t *testing.T) {
	rules := []DagRules{
		{MaxParents: 2, MaxFreeParents: 2, MaxExtraData: 0},
		{MaxParents: 2, MaxFreeParents: 2, MaxExtraData: 1 << 20},
		{MaxParents: 10, MaxFreeParents: 10, MaxExtraData: 1 << 20},
	}

	for _, test := range rules {
		require.NoError(t, validateDagRules(test))
	}
}

func TestEmitterRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules EmitterRules
		issue string
	}{
		"more than 10 seconds emitter times is too high": {
			rules: EmitterRules{Interval: inter.Timestamp(10*time.Second) + 1},
			issue: "Interval is too high",
		},
		"hour long emitter times is too high": {
			rules: EmitterRules{Interval: inter.Timestamp(10 * time.Hour)},
			issue: "Interval is too high",
		},
		"stall threshold must be at least 10 seconds": {
			rules: EmitterRules{StallThreshold: inter.Timestamp(10*time.Second) - 1},
			issue: "StallThreshold is too low",
		},
		"stall interval must be at least 10 seconds": {
			rules: EmitterRules{StallThreshold: inter.Timestamp(10*time.Second) - 1},
			issue: "StallThreshold is too low",
		},
		"stall intervals of more than 1 minute are too high": {
			rules: EmitterRules{StalledInterval: inter.Timestamp(1*time.Minute) + 1},
			issue: "StalledInterval is too high",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateEmitterRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestEmitterRulesValidation_AcceptsValidRules(t *testing.T) {
	rules := []EmitterRules{
		{
			Interval:        inter.Timestamp(10 * time.Second),
			StallThreshold:  inter.Timestamp(10 * time.Second),
			StalledInterval: inter.Timestamp(10 * time.Second),
		},
		{
			Interval:        inter.Timestamp(100 * time.Millisecond),
			StallThreshold:  inter.Timestamp(10 * time.Second),
			StalledInterval: inter.Timestamp(1 * time.Minute),
		},
		{
			Interval:        inter.Timestamp(3 * time.Second),
			StallThreshold:  inter.Timestamp(1 * time.Hour),
			StalledInterval: inter.Timestamp(30 * time.Second),
		},
	}

	for i, test := range rules {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			require.NoError(t, validateEmitterRules(test))
		})
	}
}

func TestEpochsRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules EpochsRules
		issue string
	}{
		"more than day long epochs are too high": {
			rules: EpochsRules{MaxEpochDuration: inter.Timestamp(24*time.Hour) + 1},
			issue: "MaxEpochDuration is too high",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateEpochsRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestEpochsRulesValidation_AcceptsValidRules(t *testing.T) {
	rules := []EpochsRules{
		{MaxEpochDuration: inter.Timestamp(1 * time.Hour)},
		{MaxEpochDuration: inter.Timestamp(0)},
	}

	for _, test := range rules {
		require.NoError(t, validateEpochsRules(test))
	}
}

func TestBlocksRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules BlocksRules
		issue string
	}{
		"no block gas": {
			rules: BlocksRules{MaxBlockGas: 0},
			issue: "MaxBlockGas is too low",
		},
		"max block gas less than minimum is too low": {
			rules: BlocksRules{MaxBlockGas: MinimumMaxBlockGas - 1},
			issue: "MaxBlockGas is too low",
		},
		"max block gas more than maximum is too high": {
			rules: BlocksRules{MaxBlockGas: MaximumMaxBlockGas + 1},
			issue: "MaxBlockGas is too high",
		},
		"uint64 max block gas": {
			rules: BlocksRules{MaxBlockGas: math.MaxUint64},
			issue: "MaxBlockGas is too high",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateBlocksRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestBlocksRulesValidation_AcceptsValidRules(t *testing.T) {
	rules := []BlocksRules{
		{MaxBlockGas: MinimumMaxBlockGas},
		{MaxBlockGas: MaximumMaxBlockGas},
		{MaxBlockGas: MaximumMaxBlockGas / 2},
	}

	for _, test := range rules {
		require.NoError(t, validateBlocksRules(test))
	}
}

func TestGasRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules GasRules
		issue string
	}{
		"zero max event gas is too low": {
			rules: GasRules{MaxEventGas: 0},
			issue: "MaxEventGas is too low",
		},
		"zero event gas is too low": {
			rules: GasRules{EventGas: 0},
			issue: "EventGas is too low",
		},
		"less than rule-update gas is too low": {
			rules: GasRules{MaxEventGas: upperBoundForRuleChangeGasCosts - 1},
			issue: "MaxEventGas is too low",
		},
		"too high event gas costs": {
			rules: GasRules{EventGas: 1},
			issue: "EventGas is too high",
		},
		"insufficient capacity for rule update": {
			rules: GasRules{MaxEventGas: upperBoundForRuleChangeGasCosts, EventGas: 1},
			issue: "EventGas is too high",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateGasRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestGasRulesValidation_AcceptsValidRules(t *testing.T) {
	rules := []GasRules{
		{MaxEventGas: upperBoundForRuleChangeGasCosts, EventGas: 0},
		{MaxEventGas: upperBoundForRuleChangeGasCosts + 10, EventGas: 10},
		{MaxEventGas: upperBoundForRuleChangeGasCosts + 15, EventGas: 10},
		{MaxEventGas: 1000 * upperBoundForRuleChangeGasCosts, EventGas: 10000},
	}

	for _, test := range rules {
		require.NoError(t, validateGasRules(test))
	}
}

func TestEconomyRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules EconomyRules
		issue string
	}{
		"min gas price must not be nil": {
			rules: EconomyRules{MinGasPrice: nil},
			issue: "MinGasPrice is nil",
		},
		"min base fee must not be nil": {
			rules: EconomyRules{MinBaseFee: nil},
			issue: "MinBaseFee is nil",
		},
		"negative min base fee is too low": {
			rules: EconomyRules{MinBaseFee: big.NewInt(-1)},
			issue: "MinBaseFee is negative",
		},
		"too high min base fee is too high": {
			rules: EconomyRules{MinBaseFee: new(big.Int).Add(maxMinimumGasPrice, big.NewInt(1))},
			issue: "MinBaseFee is too high",
		},
		"too low event gas": {
			rules: EconomyRules{Gas: GasRules{MaxEventGas: 1}},
			issue: "MaxEventGas is too low",
		},
		"too low short-gas allocation per second": {
			rules: EconomyRules{ShortGasPower: GasPowerRules{AllocPerSec: 1}},
			issue: "ShortGasPower.AllocPerSec is too low",
		},
		"too low long-gas allocation per second": {
			rules: EconomyRules{LongGasPower: GasPowerRules{AllocPerSec: 1}},
			issue: "LongGasPower.AllocPerSec is too low",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateEconomyRules(test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestEconomyRulesValidation_Long_Short_Power_Differs(t *testing.T) {
	tests := map[string]struct {
		long  GasPowerRules
		short GasPowerRules
		issue string
	}{
		"alloc": {
			long:  GasPowerRules{AllocPerSec: 20 * upperBoundForRuleChangeGasCosts},
			short: GasPowerRules{AllocPerSec: 30 * upperBoundForRuleChangeGasCosts},
			issue: "ShortGasPower.AllocPerSec and LongGasPower.AllocPerSec differ",
		},
		"max alloc": {
			long:  GasPowerRules{MaxAllocPeriod: inter.Timestamp(10 * time.Second)},
			short: GasPowerRules{MaxAllocPeriod: inter.Timestamp(20 * time.Second)},
			issue: "ShortGasPower.MaxAllocPeriod and LongGasPower.MaxAllocPeriod differ",
		},
		"startup alloc": {
			long:  GasPowerRules{StartupAllocPeriod: inter.Timestamp(10 * time.Second)},
			short: GasPowerRules{StartupAllocPeriod: inter.Timestamp(20 * time.Second)},
			issue: "ShortGasPower.StartupAllocPeriod and LongGasPower.StartupAllocPeriod differ",
		},
		"min startup gas": {
			long:  GasPowerRules{MinStartupGas: 10 * upperBoundForRuleChangeGasCosts},
			short: GasPowerRules{MinStartupGas: 20 * upperBoundForRuleChangeGasCosts},
			issue: "ShortGasPower.MinStartupGas and LongGasPower.MinStartupGas differ",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateEconomyRules(EconomyRules{
				ShortGasPower: test.short,
				LongGasPower:  test.long,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestEconomyRulesValidation_AcceptsValidRules(t *testing.T) {
	valid := EconomyRules{
		Gas: GasRules{
			MaxEventGas: upperBoundForRuleChangeGasCosts,
		},
		MinGasPrice: big.NewInt(0),
		MinBaseFee:  big.NewInt(0),
		ShortGasPower: GasPowerRules{
			AllocPerSec:        10 * upperBoundForRuleChangeGasCosts,
			MaxAllocPeriod:     inter.Timestamp(time.Second),
			StartupAllocPeriod: inter.Timestamp(time.Second),
		},
		LongGasPower: GasPowerRules{
			AllocPerSec:        10 * upperBoundForRuleChangeGasCosts,
			MaxAllocPeriod:     inter.Timestamp(time.Second),
			StartupAllocPeriod: inter.Timestamp(time.Second),
		},
	}

	require.NoError(t, validateEconomyRules(valid))
}

func TestGasPowerRulesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		rules GasPowerRules
		issue string
	}{
		"zero gas allocation per second is too low": {
			rules: GasPowerRules{AllocPerSec: 0},
			issue: "AllocPerSec is too low",
		},
		"too low allocation per second": {
			rules: GasPowerRules{AllocPerSec: 10*upperBoundForRuleChangeGasCosts - 1},
			issue: "AllocPerSec is too low",
		},
		"no allocation period is too low": {
			rules: GasPowerRules{MaxAllocPeriod: 0},
			issue: "AllocPeriod is too low",
		},
		"less than a second allocation period is too low": {
			rules: GasPowerRules{MaxAllocPeriod: inter.Timestamp(time.Second) - 1},
			issue: "AllocPeriod is too low",
		},
		"more than a minute allocation period is too high": {
			rules: GasPowerRules{MaxAllocPeriod: inter.Timestamp(time.Minute) + 1},
			issue: "AllocPeriod is too high",
		},
		"less than a second startup period is too low": {
			rules: GasPowerRules{StartupAllocPeriod: inter.Timestamp(time.Second) - 1},
			issue: "StartupAllocPeriod is too low",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateGasPowerRules("", test.rules)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestGasPowerRulesValidation_AcceptsValidRules(t *testing.T) {
	sec := inter.Timestamp(time.Second)
	min := inter.Timestamp(time.Minute)
	rules := []GasPowerRules{
		{AllocPerSec: 10 * upperBoundForRuleChangeGasCosts, MaxAllocPeriod: 1 * sec, StartupAllocPeriod: 1 * sec},
		{AllocPerSec: 10 * upperBoundForRuleChangeGasCosts, MaxAllocPeriod: 1 * min, StartupAllocPeriod: 1 * sec},
		{AllocPerSec: 10 * upperBoundForRuleChangeGasCosts, MaxAllocPeriod: 1 * min, StartupAllocPeriod: 1 * min},
		{AllocPerSec: math.MaxUint64, MaxAllocPeriod: 1 * min, StartupAllocPeriod: 1 * min},
	}

	for _, test := range rules {
		require.NoError(t, validateGasPowerRules("", test))
	}
}

func TestUpgradesValidation_DetectsIssues(t *testing.T) {
	issues := map[string]struct {
		upgrade Upgrades
		issue   string
	}{
		"Berlin upgrade is required": {
			upgrade: Upgrades{},
			issue:   "Berlin upgrade is required",
		},
		"London upgrade is required": {
			upgrade: Upgrades{},
			issue:   "London upgrade is required",
		},
		"LLR upgrade is not supported": {
			upgrade: Upgrades{Llr: true},
			issue:   "LLR upgrade is not supported",
		},
		"Sonic upgrade requires London": {
			upgrade: Upgrades{Sonic: true},
			issue:   "Sonic upgrade requires London",
		},
		"London upgrade requires Berlin": {
			upgrade: Upgrades{London: true},
			issue:   "London upgrade requires Berlin",
		},
		"Sonic upgrade is required": {
			upgrade: Upgrades{},
			issue:   "Sonic upgrade is required",
		},
		"Allegro upgrade is required": {
			upgrade: Upgrades{},
			issue:   "Allegro upgrade is required",
		},
	}

	for name, test := range issues {
		t.Run(name, func(t *testing.T) {
			err := validateUpgrades(test.upgrade)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.issue)
		})
	}
}

func TestUpgradesValidation_AcceptsValidRules(t *testing.T) {
	upgrades := []Upgrades{
		{Berlin: true, London: true, Sonic: true, Allegro: true},
	}

	for _, test := range upgrades {
		require.NoError(t, validateUpgrades(test))
	}
}
