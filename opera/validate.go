package opera

import (
	"errors"
	"math/big"
	"time"

	"github.com/Fantom-foundation/go-opera/inter"
)

var (
	maxMinimumGasPrice = new(big.Int).SetUint64(1000 * 1e9) // 1000 Gwei
)

func validate(rules Rules) error {
	return errors.Join(
		validateDagRules(rules.Dag),
		validateEmitterRules(rules.Emitter),
		validateEpochsRules(rules.Epochs),
		validateBlockRules(rules.Blocks),
		validateEconomyRules(rules.Economy),
		validateUpgrades(rules.Upgrades),
	)
}

func validateDagRules(rules DagRules) error {
	var issues []error

	if rules.MaxParents < 2 {
		issues = append(issues, errors.New("Dag.MaxParents is too low"))
	}

	if rules.MaxExtraData > 1<<20 { // 1 MB
		issues = append(issues, errors.New("Dag.MaxExtraData is too high"))
	}

	return errors.Join(issues...)
}

func validateEmitterRules(rules EmitterRules) error {

	var issues []error
	if rules.Interval < inter.Timestamp(100*time.Millisecond) {
		issues = append(issues, errors.New("Emitter.Interval is too low"))
	}
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

	if rules.MaxEpochDuration > inter.Timestamp(24*time.Hour) {
		issues = append(issues, errors.New("Epochs.MaxEpochDuration is too high"))
	}

	return errors.Join(issues...)
}

func validateBlockRules(rules BlocksRules) error {
	var issues []error

	if rules.MaxBlockGas < MinimumMaxBlockGas {
		issues = append(issues, errors.New("MaxBlockGas is too low"))
	}
	if rules.MaxBlockGas > MaximumMaxBlockGas {
		issues = append(issues, errors.New("MaxBlockGas is too high"))
	}

	// The empty-block skip period is not restricted. There are no too low or too high values.

	return errors.Join(issues...)
}

func validateEconomyRules(rules EconomyRules) error {
	var issues []error

	if rules.MinGasPrice == nil {
		issues = append(issues, errors.New("MinGasPrice is nil"))
	} else {
		if rules.MinGasPrice.Sign() < 0 {
			issues = append(issues, errors.New("MinGasPrice is negative"))
		}
		if rules.MinGasPrice.Cmp(maxMinimumGasPrice) > 0 {
			issues = append(issues, errors.New("MinGasPrice is too high"))
		}
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

	// TODO: check BlockMissedSlack

	issues = append(issues, validateGasRules(rules.Gas))
	issues = append(issues, validateGasPowerRules("Economy.ShortGasPower", rules.ShortGasPower))
	issues = append(issues, validateGasPowerRules("Economy.LongGasPower", rules.LongGasPower))

	return errors.Join(issues...)
}

func validateGasRules(rules GasRules) error {
	var issues []error

	// TODO: implement

	return errors.Join(issues...)
}

func validateGasPowerRules(prefix string, rules GasPowerRules) error {
	var issues []error

	// TODO: implement

	return errors.Join(issues...)
}

func validateUpgrades(upgrade Upgrades) error {
	var issues []error

	if upgrade.Llr {
		issues = append(issues, errors.New("LLR upgrade is not supported"))
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

	return errors.Join(issues...)
}
