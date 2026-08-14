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

package subsidies

import (
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
)

// Mode is how the installed registry answers chooseFund, and so who pays for a sponsored
// transaction. It is a property of the registry rather than of a transaction, which is why it is
// fixed per network here.
type Mode uint8

const (
	// ModeFundBacked is mode 1: a fund covers the fee and a deductFees transaction charges it
	// afterwards. Coverage then depends on what the fund holds, which is the interesting axis.
	ModeFundBacked Mode = iota

	// ModeNetwork is mode 2: the network absorbs the cost and no follow-up transaction is added.
	ModeNetwork

	// ModeNetworkTracked is mode 3: the network absorbs the cost and a track transaction records it.
	ModeNetworkTracked
)

func (m Mode) String() string {
	switch m {
	case ModeFundBacked:
		return "fund-backed"
	case ModeNetwork:
		return "network-sponsored"
	case ModeNetworkTracked:
		return "network-sponsored-with-tracking"
	}
	return "unknown"
}

// PostTransactions reports how many transactions a sponsorship of this mode appends to the block
// after the sponsored one.
func (m Mode) PostTransactions() int {
	if m == ModeNetwork {
		return 0
	}
	return 1
}

// Overhead is the gas this mode charges beyond the transaction's own.
func (m Mode) Overhead(config GasConfig) uint64 {
	switch m {
	case ModeFundBacked:
		return config.FundBackedOverhead
	case ModeNetworkTracked:
		return config.TrackedOverhead
	}
	return 0
}

// coverage is what the model knows about one sender's funding for the batch being planned: what its
// fund held once the iteration's top-up had settled, and what the requests of that batch will ask of
// it altogether.
type coverage struct {
	funds *big.Int

	// requiredGas is the gas the batch's requests from this sender ask of the fund, kept as gas
	// rather than as wei so that the rules price it at the base fee they are judging against.
	requiredGas uint64
}

// PricingRules decide whether a sponsorship request can pay its way, which it does out of a fund
// rather than out of its sender's balance -- so the affordability rules do not apply to one at all,
// and a zero gas price is not the fatal fee cap it is for an ordinary transaction. Anything that is
// not a request is handed to the rules of the domain being wrapped.
//
// Coverage is decided in three bands, because within one block the chosen fund is charged between
// requests, chooseFund tests the gas limit while deductFees charges the gas used, and the order the
// requests reach the block in is the scrambler's to choose. Naming one outcome in the middle band
// would manufacture failures; the bands still catch every real defect, since a request that must not
// execute but does, or must execute but does not, falls outside them.
func (d *Domain) PricingRules(fallback core.PricingRules) core.PricingRules {
	return func(
		spec core.TxSpec,
		sender core.SenderState,
		baseFee *big.Int,
		cfg core.NetworkConfig,
	) []core.Rule {

		if !cfg.Upgrades.GasSubsidies || !IsSponsorshipRequest(spec) {
			return fallback(spec, sender, baseFee, cfg)
		}

		// Modes 2 and 3 cover every request, so nothing about the price can stop one.
		if d.Mode != ModeFundBacked {
			return nil
		}

		seen := d.coverageOf(spec)
		alone := d.Requirement(spec, baseFee)
		required := new(big.Int).Mul(new(big.Int).SetUint64(seen.requiredGas), baseFee)
		return []core.Rule{
			{Applies: seen.funds.Cmp(core.Scale(alone, 1, 4)) < 0,
				Reason: fmt.Sprintf("no fund holds enough to cover the fee of this sponsorship "+
					"request: %v wei against the %v its %d gas needs at a base fee of %v",
					seen.funds, alone, spec.Gas()+d.Config.MaxOverhead(), baseFee),
				Outcomes: core.Dropped},
			{Applies: seen.funds.Cmp(core.Scale(required, 4, 1)) < 0,
				Reason: fmt.Sprintf("the fund is too close to what this sender's requests ask of it "+
					"to say which of them it still covers: %v wei against %v", seen.funds, required),
				Outcomes: core.Either},
		}
	}
}

// Requirement is what a fund must hold for chooseFund to accept a request, mirroring IsCovered: the
// base fee on the gas limit plus the largest overhead any mode reserves. deductFees later charges
// the gas actually used, which is never more.
func (d *Domain) Requirement(spec core.TxSpec, baseFee *big.Int) *big.Int {
	gas := new(big.Int).SetUint64(core.SaturatingAdd(spec.Gas(), d.Config.MaxOverhead()))
	return gas.Mul(gas, baseFee)
}

// Charge is what a mode's follow-up transaction takes from the fund for a request that executed.
func (d *Domain) Charge(gasUsed uint64, baseFee *big.Int) *big.Int {
	gas := new(big.Int).SetUint64(core.SaturatingAdd(gasUsed, d.Mode.Overhead(d.Config)))
	return gas.Mul(gas, baseFee)
}
