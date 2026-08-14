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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/stretchr/testify/require"
)

// testConfig is what the reference registry answers, so the arithmetic below is the arithmetic the
// chain does.
var testConfig = GasConfig{FundBackedOverhead: 210_000, TrackedOverhead: 230_000}

// verdict is what a rule table says, or "" when nothing in it applies.
func verdict(rules []core.Rule) core.Prediction {
	for _, rule := range rules {
		if rule.Applies {
			return core.Allow(rule.Reason, rule.Outcomes...)
		}
	}
	return core.Allow("nothing applies", core.OutcomeExecuted)
}

// domainWith is a domain whose iteration state says one sender's fund holds funds and its requests ask
// requiredGas of it.
func domainWith(mode Mode, funds *big.Int, requiredGas uint64) *Domain {
	return &Domain{
		Mode:     mode,
		Config:   testConfig,
		senders:  1,
		coverage: []coverage{{funds: funds, requiredGas: requiredGas}},
	}
}

func TestPricingRules_DecideCoverageInThreeBands(t *testing.T) {
	const gas = 100_000
	baseFee := big.NewInt(1_000)

	// What one request of that gas asks of the fund, and what the batch asks altogether.
	alone := new(big.Int).Mul(big.NewInt(gas+230_000), baseFee)
	required := new(big.Int).Mul(big.NewInt(2*(gas+230_000)), baseFee) // two such requests

	tests := map[string]struct {
		funds    *big.Int
		outcomes []core.Outcome
	}{
		"an empty fund covers nothing": {
			funds:    new(big.Int),
			outcomes: core.Dropped,
		},
		"a fund below a quarter of one request covers nothing": {
			funds:    new(big.Int).Sub(core.Scale(alone, 1, 4), big.NewInt(1)),
			outcomes: core.Dropped,
		},
		"a fund near what the batch asks may or may not still cover this one": {
			funds:    required,
			outcomes: core.Either,
		},
		"a fund at four times what the batch asks covers all of it": {
			funds:    core.Scale(required, 4, 1),
			outcomes: nil, // nothing about the price objects
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			domain := domainWith(ModeFundBacked, test.funds, 2*(gas+230_000))
			rules := domain.PricingRules(regular.PricingRules)(
				request(0, gas), core.SenderState{Balance: core.AccountBalance},
				baseFee, testNetwork(opera.GetBrioUpgrades()),
			)

			if test.outcomes == nil {
				require.Equal(t, core.Allow("nothing applies", core.OutcomeExecuted).Allowed,
					verdict(rules).Allowed)
				return
			}
			require.Equal(t, test.outcomes, verdict(rules).Allowed, verdict(rules).Reason)
		})
	}
}

func TestPricingRules_NetworkSponsorshipsNeedNoFund(t *testing.T) {
	for _, mode := range []Mode{ModeNetwork, ModeNetworkTracked} {
		domain := domainWith(mode, new(big.Int), 331_000)

		rules := domain.PricingRules(regular.PricingRules)(
			request(0, 100_000), core.SenderState{Balance: new(big.Int)},
			big.NewInt(1_000), testNetwork(opera.GetBrioUpgrades()),
		)
		require.Equal(t, []core.Outcome{core.OutcomeExecuted}, verdict(rules).Allowed,
			"%s must not object to a request over an empty fund", mode)
	}
}

func TestPricingRules_LeaveAnythingPayingItsOwnWayToTheInnerDomain(t *testing.T) {
	baseFee := big.NewInt(1_000)
	domain := domainWith(ModeFundBacked, new(big.Int), 0)
	network := testNetwork(opera.GetBrioUpgrades())

	// A transaction offering a viable fee cap is judged by the balance, not by any fund.
	rules := domain.PricingRules(regular.PricingRules)(
		paying(0, 100_000), core.SenderState{Balance: new(big.Int)}, baseFee, network,
	)
	require.Equal(t, core.Dropped, verdict(rules).Allowed)
	require.Contains(t, verdict(rules).Reason, "cannot back the gas")

	// And with a balance behind it, nothing objects.
	rules = domain.PricingRules(regular.PricingRules)(
		paying(0, 100_000), core.SenderState{Balance: core.AccountBalance}, baseFee, network,
	)
	require.Equal(t, []core.Outcome{core.OutcomeExecuted}, verdict(rules).Allowed)
}

func TestPricingRules_IgnoreEverythingWhereTheFeatureIsOff(t *testing.T) {
	baseFee := big.NewInt(1_000)
	domain := domainWith(ModeFundBacked, new(big.Int), 331_000)

	network := testNetwork(opera.GetBrioUpgrades())
	network.Upgrades.GasSubsidies = false

	// Without the feature, a zero fee cap is simply below the base fee.
	rules := domain.PricingRules(regular.PricingRules)(
		request(0, 100_000), core.SenderState{Balance: core.AccountBalance}, baseFee, network,
	)
	require.Equal(t, core.Dropped, verdict(rules).Allowed)
	require.Contains(t, verdict(rules).Reason, "below the block's base fee")
}

func TestRequirementAndCharge_PriceTheGasTheRegistryReserves(t *testing.T) {
	baseFee := big.NewInt(7)

	fundBacked := domainWith(ModeFundBacked, new(big.Int), 0)
	require.Equal(t, big.NewInt((100_000+230_000)*7),
		fundBacked.Requirement(request(0, 100_000), baseFee),
		"coverage is asked for the larger overhead of the two, as IsCovered does")
	require.Equal(t, big.NewInt((50_000+210_000)*7), fundBacked.Charge(50_000, baseFee),
		"the charge is the gas actually used plus this mode's own overhead")

	tracked := domainWith(ModeNetworkTracked, new(big.Int), 0)
	require.Equal(t, big.NewInt((50_000+230_000)*7), tracked.Charge(50_000, baseFee))

	network := domainWith(ModeNetwork, new(big.Int), 0)
	require.Equal(t, big.NewInt(50_000*7), network.Charge(50_000, baseFee),
		"a mode with no follow-up reserves no overhead")
}

func TestDonation_FollowsTheLevel(t *testing.T) {
	baseFee := big.NewInt(10)
	const requiredGas = 331_000

	require.Zero(t, donation(FundNothing, requiredGas, baseFee).Sign())
	require.Equal(t, big.NewInt(requiredGas*10), donation(FundExactly, requiredGas, baseFee))
	require.Equal(t, big.NewInt(requiredGas*10*amplyOver), donation(FundAmply, requiredGas, baseFee))
}

func TestMode_KnowsWhatFollowsASponsoredTransaction(t *testing.T) {
	require.Equal(t, 1, ModeFundBacked.PostTransactions())
	require.Equal(t, 0, ModeNetwork.PostTransactions())
	require.Equal(t, 1, ModeNetworkTracked.PostTransactions())

	require.Equal(t, uint64(210_000), ModeFundBacked.Overhead(testConfig))
	require.Equal(t, uint64(0), ModeNetwork.Overhead(testConfig))
	require.Equal(t, uint64(230_000), ModeNetworkTracked.Overhead(testConfig))
	require.Equal(t, uint64(230_000), testConfig.MaxOverhead())
}
