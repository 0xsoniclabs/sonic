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

package oracle

import (
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
)

// constructivePrice is the price the gas power the validators have left asks
// for: the less is left of the most they can hold, the higher the multiple
// of the adjusted minimum price. The offsets are taken off the gas left and
// off its ratio.
func constructivePrice(rules *opera.Rules, gasPowerLeft, gasOffset, ratioOffset uint64, adjustedMinPrice *big.Int) *big.Int {
	left := gasPowerLeft
	if left > gasOffset {
		left -= gasOffset
	} else {
		left = 0
	}
	freeRatioBig := new(big.Int).Mul(new(big.Int).SetUint64(left), decimalUnit)
	freeRatioBig.Div(freeRatioBig, maxTotalGasPower(rules))
	freeRatio := freeRatioBig.Uint64()
	if freeRatio > ratioOffset {
		freeRatio -= ratioOffset
	} else {
		freeRatio = 0
	}
	freeRatio = min(freeRatio, DecimalUnit)
	price := new(big.Int).SetUint64(freeRatioToPriceRatio(freeRatio))
	price.Mul(price, adjustedMinPrice)
	return price.Div(price, decimalUnit)
}

// maxTotalGasPower is the most long-term gas power the validators can hold.
func maxTotalGasPower(rules *opera.Rules) *big.Int {
	power := new(big.Int).SetUint64(rules.Economy.LongGasPower.AllocPerSec)
	power.Mul(power, new(big.Int).SetUint64(uint64(rules.Economy.LongGasPower.MaxAllocPeriod)))
	return power.Div(power, big.NewInt(int64(time.Second)))
}

var freeRatioToPriceRatio = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 25 * DecimalUnit},
	{X: 0.3 * DecimalUnit, Y: 9 * DecimalUnit},
	{X: 0.5 * DecimalUnit, Y: 3.75 * DecimalUnit},
	{X: 0.8 * DecimalUnit, Y: 1.5 * DecimalUnit},
	{X: 0.95 * DecimalUnit, Y: 1.05 * DecimalUnit},
	{X: DecimalUnit, Y: DecimalUnit},
})
