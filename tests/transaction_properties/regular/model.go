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

package regular

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
)

// PricingRules decide whether an ordinary transaction can pay its way, which it does out of its
// sender's own balance: it has to offer at least the block's base fee and hold what buyGas will
// debit. They are the tail of the model's rule table, and are written from the specification rather
// than by calling the validators under test, which would assert nothing.
//
// Where the answer depends on a base fee that moves between being read and being applied, the
// prediction is a set rather than a guess: the band is four times wide in both directions, which is
// why the generator never draws a fee cap or a balance near the base fee.
func PricingRules(
	spec core.TxSpec,
	sender core.SenderState,
	baseFee *big.Int,
	cfg core.NetworkConfig,
) []core.Rule {

	var (
		feeCap      = spec.FeeCap()
		lowBaseFee  = core.Scale(baseFee, 1, 4)
		highBaseFee = core.Scale(baseFee, 4, 1)
	)

	return []core.Rule{
		{Applies: feeCap.Cmp(lowBaseFee) < 0,
			Reason:   "gas fee cap is below the block's base fee",
			Outcomes: core.Dropped},
		{Applies: feeCap.Cmp(highBaseFee) < 0,
			Reason:   "gas fee cap is too close to the base fee to predict",
			Outcomes: core.Either},
		{Applies: core.Affordability(spec, highBaseFee).BitLen() > 256,
			Reason:   "the balance buyGas requires does not fit in 256 bits, so the check fails outright",
			Outcomes: core.Dropped},
		{Applies: sender.Balance.Cmp(core.Affordability(spec, lowBaseFee)) < 0,
			Reason:   "the sender cannot back the gas its limit reserves",
			Outcomes: core.Dropped},
		{Applies: sender.Balance.Cmp(core.Affordability(spec, highBaseFee)) < 0,
			Reason:   "the sender's balance is too close to what buyGas requires to predict",
			Outcomes: core.Either},
	}
}
