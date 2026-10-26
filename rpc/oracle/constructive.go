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

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"

	"github.com/0xsoniclabs/sonic/opera"
)

var second = new(big.Int).SetUint64(uint64(time.Second))

// freeRatioToPriceRatio maps the share of the gas power the validators have
// left to the multiple of the minimum price a transaction must pay.
var freeRatioToPriceRatio = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 25 * DecimalUnit},
	{X: 0.3 * DecimalUnit, Y: 9 * DecimalUnit},
	{X: 0.5 * DecimalUnit, Y: 3.75 * DecimalUnit},
	{X: 0.8 * DecimalUnit, Y: 1.5 * DecimalUnit},
	{X: 0.95 * DecimalUnit, Y: 1.05 * DecimalUnit},
	{X: DecimalUnit, Y: DecimalUnit},
})

// maxTotalGasPower is the gas power the validators can hold at most under
// the rules.
func maxTotalGasPower(rules opera.Rules) *big.Int {
	allocation := new(big.Int).SetUint64(rules.Economy.LongGasPower.AllocPerSec)
	period := new(big.Int).SetUint64(uint64(rules.Economy.LongGasPower.MaxAllocPeriod))
	total := new(big.Int).Mul(allocation, period)
	return total.Div(total, second)
}

// constructivePrice is the price the gas power left asks for once the gas of
// the pool and a margin are spent from it.
func constructivePrice(rules opera.Rules, gasPowerLeft, gasOffset, ratioOffset uint64, adjustedMinPrice *big.Int) *big.Int {
	maximum := maxTotalGasPower(rules)
	if gasPowerLeft > gasOffset {
		gasPowerLeft -= gasOffset
	} else {
		gasPowerLeft = 0
	}
	freeRatio := uint64(0)
	if maximum.Sign() > 0 {
		freeRatioValue := new(big.Int).SetUint64(gasPowerLeft)
		freeRatioValue.Mul(freeRatioValue, decimalUnit)
		freeRatio = freeRatioValue.Div(freeRatioValue, maximum).Uint64()
	}
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
