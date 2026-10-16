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

// Package oracle suggests gas prices. It holds the gas price oracle and the
// derived price suggestions of the node.
package oracle

import "math/big"

// SuggestedGasPrice is the gas price to suggest for new transactions given the
// base fee: 10% above it, a buffer for short-term price fluctuations. It is the
// same function the node's gaspricelimits package provides; a test keeps the
// two equal.
func SuggestedGasPrice(baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return big.NewInt(0)
	}
	res := new(big.Int).Set(baseFee)
	res.Mul(res, big.NewInt(110))
	res.Div(res, big.NewInt(100))
	return res
}
