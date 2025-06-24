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

package gaspricelimits

import "math/big"

// GetSuggestedGasPriceForNewTransactions a gas price that should be suggested
// to users for new transactions based on the current base fee. This function
// returns a value that is 10% higher than the base fee to provide a buffer
// for the price to increase before the transaction is included in a block.
func GetSuggestedGasPriceForNewTransactions(baseFee *big.Int) *big.Int {
	return addPercentage(baseFee, 10)
}

// GetMinimumFeeCapForTransactionPool returns the gas price the transaction pool
// should check for when accepting new transactions. This function returns a
// value that is 5% higher than the base fee to provide a buffer for the price
// to increase before the transaction is included in a block.
func GetMinimumFeeCapForTransactionPool(baseFee *big.Int) *big.Int {
	return addPercentage(baseFee, 5)
}

// GetMinimumFeeCapForEventEmitter returns the gas price the event emitter should
// check for when including transactions in a block. This function returns a
// value that is 2% higher than the base fee to provide a buffer for the price
// to increase before the transaction is included in a block.
func GetMinimumFeeCapForEventEmitter(baseFee *big.Int) *big.Int {
	return addPercentage(baseFee, 2)
}

func addPercentage(a *big.Int, percentage int) *big.Int {
	if a == nil {
		return big.NewInt(0)
	}
	res := new(big.Int).Set(a)
	res.Mul(res, big.NewInt(int64(percentage+100)))
	res.Div(res, big.NewInt(100))
	return res
}
