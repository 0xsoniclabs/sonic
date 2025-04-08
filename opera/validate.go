package opera

import (
	"errors"
	"github.com/0xsoniclabs/sonic/inter"
	"math"
	"math/big"
	"time"
)

// This file handles the validation of network rules.
// Validation is performed either during the generation of a new genesis file
// or when a network rule change is initiated via a special transaction.
// This validation was introduced with the Allegro upgrade.
//
// Caution:
// The validation rules in this file MUST NOT be modified after the Allegro upgrade is activated.
// Altering these rules would result in inconsistent behavior across the network,
// with some nodes accepting the rule change while others reject it. This could cause
// a network split, where certain nodes process the rule change, and others do not.
//
// Network rules can only be modified as part of a subsequent network upgrade.
// In such cases, the validation logic must be updated to accommodate the new rules
// and activated simultaneously with the upgrade.

func validate(rules Rules) error {
	return errors.Join(
		validateDagRules(rules.Dag),
		validateEmitterRules(rules.Emitter),
		validateEpochsRules(rules.Epochs),
		validateBlocksRules(rules.Blocks),
		validateEconomyRules(rules.Economy),
		validateUpgrades(rules.Upgrades),
	)
}

func validateDagRules(rules DagRules) error {
	var issues []error

	if rules.MaxParents < 2 {
		issues = append(issues, errors.New("Dag.MaxParents is too low"))
	}

	if rules.MaxFreeParents < 2 {
		issues = append(issues, errors.New("Dag.MaxFreeParents is too low"))
	}

	if rules.MaxExtraData > 1<<20 { // 1 MB
		issues = append(issues, errors.New("Dag.MaxExtraData is too high"))
	}

	return errors.Join(issues...)
}

func validateEmitterRules(rules EmitterRules) error {

	var issues []error
	if rules.Interval > inter.Timestamp(10*time.Second) {
		issues = append(issues, errors.New("Emitter.Interval is too high"))
	}

	if rules.StallThreshold < inter.Timestamp(10*time.Second) {
		issues = append(issues, errors.New("Emitter.StallThreshold is too low"))
	}

	if rules.StalledInterval < inter.Timestamp(10*time.Second) {
		issues = append(issues, errors.New("Emitter.StalledInterval is too low"))
	}
	if rules.StalledInterval > inter.Timestamp(1*time.Minute) {
		issues = append(issues, errors.New("Emitter.StalledInterval is too high"))
	}

	return errors.Join(issues...)
}

func validateEpochsRules(rules EpochsRules) error {
	var issues []error

	// MaxEpochGas is not restricted. If it is too low, we will have an epoch per block, which is
	// not great performance-wise, but it is not invalid. If it is too high, the time limit will
	// eventually end a long epoch.

	if rules.MaxEpochDuration > inter.Timestamp(1*time.Hour) {
		issues = append(issues, errors.New("Epochs.MaxEpochDuration is too high"))
	}

	return errors.Join(issues...)
}

func validateBlocksRules(rules BlocksRules) error {
	var issues []error

	if rules.MaxBlockGas < minimumMaxBlockGas {
		issues = append(issues, errors.New("Blocks.MaxBlockGas is too low"))
	}
	if rules.MaxBlockGas > maximumMaxBlockGas {
		issues = append(issues, errors.New("Blocks.MaxBlockGas is too high"))
	}

	// The empty-block skip period is not restricted. There are no too low or too high values.

	return errors.Join(issues...)
}

var (
	// maxMinimumGasPrice is the maximum allowed minimum gas price. An upper limit is
	// added to avoid a situation where the gas-free pricing is accidentally set to such
	// a high value that another rule-change can no longer be afforded.
	maxMinimumGasPrice = new(big.Int).SetUint64(1000 * 1e9) // 1000 Gwei
)

func validateEconomyRules(rules EconomyRules) error {
	var issues []error

	if rules.MinGasPrice == nil {
		issues = append(issues, errors.New("MinGasPrice is nil"))
	}

	if rules.MinBaseFee == nil {
		issues = append(issues, errors.New("MinBaseFee is nil"))
	} else {
		if rules.MinBaseFee.Sign() < 0 {
			issues = append(issues, errors.New("MinBaseFee is negative"))
		}
		if rules.MinBaseFee.Cmp(maxMinimumGasPrice) > 0 {
			issues = append(issues, errors.New("MinBaseFee is too high"))
		}
	}

	if rules.ShortGasPower.StartupAllocPeriod != rules.LongGasPower.StartupAllocPeriod {
		issues = append(issues, errors.New("ShortGasPower.StartupAllocPeriod and LongGasPower.StartupAllocPeriod differ"))
	}
	if rules.ShortGasPower.MaxAllocPeriod != rules.LongGasPower.MaxAllocPeriod {
		issues = append(issues, errors.New("ShortGasPower.MaxAllocPeriod and LongGasPower.MaxAllocPeriod differ"))
	}
	if rules.ShortGasPower.AllocPerSec != rules.LongGasPower.AllocPerSec {
		issues = append(issues, errors.New("ShortGasPower.AllocPerSec and LongGasPower.AllocPerSec differ"))
	}
	if rules.ShortGasPower.MinStartupGas != rules.LongGasPower.MinStartupGas {
		issues = append(issues, errors.New("ShortGasPower.MinStartupGas and LongGasPower.MinStartupGas differ"))
	}

	// There are deliberately no checks for the BlockMissedSlack. This can be set to any value.

	issues = append(issues, validateGasRules(rules.Gas))
	issues = append(issues, validateGasPowerRules("Economy.ShortGasPower", rules.ShortGasPower))
	issues = append(issues, validateGasPowerRules("Economy.LongGasPower", rules.LongGasPower))

	return errors.Join(issues...)
}

const (
	// upperBoundForRuleChangeGasCosts is a safe over-approximation of the gas costs of a rule change.
	upperBoundForRuleChangeGasCosts = 1_000_000 // < TODO: verify this number

	// minimumMaxBlockGas is the minimum allowed max block gas.
	//It must be large enough to allow internal transactions to seal blocks
	minimumMaxBlockGas = 5_000_000_000

	// maximumMaxBlockGas is the maximum allowed max block gas.
	// It should fit into 64-bit signed integers to avoid parsing errors in third-party libraries
	maximumMaxBlockGas = math.MaxInt64
)

func validateGasRules(rules GasRules) error {
	var issues []error

	if rules.MaxEventGas < upperBoundForRuleChangeGasCosts {
		issues = append(issues, errors.New("Gas.MaxEventGas is too low"))
	}

	if rules.EventGas > rules.MaxEventGas {
		issues = append(issues, errors.New("Gas.EventGas is too high"))
	}

	if rules.MaxEventGas < upperBoundForRuleChangeGasCosts+rules.EventGas {
		issues = append(issues, errors.New("Gas.EventGas is too high"))
	}

	// Right now, we do not have a rule that would limit the ParentGas, or ExtraDataGas.

	return errors.Join(issues...)
}

func validateGasPowerRules(prefix string, rules GasPowerRules) error {
	// The main aim of those rule-checks is to prevent a situation where
	// accidentally the gas-power is reduced to a level where no new rule
	// change can be processed anymore.

	var issues []error

	if rules.AllocPerSec < 10*upperBoundForRuleChangeGasCosts {
		issues = append(issues, errors.New(prefix+".AllocPerSec is too low"))
	}

	if rules.MaxAllocPeriod < inter.Timestamp(1*time.Second) {
		issues = append(issues, errors.New(prefix+".MaxAllocPeriod is too low"))
	}
	if rules.MaxAllocPeriod > inter.Timestamp(1*time.Minute) {
		issues = append(issues, errors.New(prefix+".MaxAllocPeriod is too high"))
	}

	if rules.StartupAllocPeriod < inter.Timestamp(1*time.Second) {
		issues = append(issues, errors.New(prefix+".StartupAllocPeriod is too low"))
	}

	return errors.Join(issues...)
}

func validateUpgrades(upgrade Upgrades) error {
	var issues []error

	if upgrade.Llr {
		issues = append(issues, errors.New("LLR upgrade is not supported"))
	}

	if !upgrade.London {
		issues = append(issues, errors.New("London upgrade is required"))
	}

	if !upgrade.Berlin {
		issues = append(issues, errors.New("Berlin upgrade is required"))
	}

	if upgrade.Sonic && !upgrade.London {
		issues = append(issues, errors.New("Sonic upgrade requires London"))
	}
	if upgrade.London && !upgrade.Berlin {
		issues = append(issues, errors.New("London upgrade requires Berlin"))
	}

	if !upgrade.Sonic {
		issues = append(issues, errors.New("Sonic upgrade is required"))
	}

	if !upgrade.Allegro {
		issues = append(issues, errors.New("Allegro upgrade is required"))
	}

	return errors.Join(issues...)
}
