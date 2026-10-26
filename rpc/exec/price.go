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

package exec

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
)

// BaseFee is the base fee of the head block, nil before London.
func BaseFee(head Head) *big.Int {
	return head.Block.BaseFee
}

// SuggestedGasPrice is the price suggested for a new transaction: the base
// fee of the head with the margin the pool and the emitter apply too.
func SuggestedGasPrice(head Head) *big.Int {
	return gaspricelimits.GetSuggestedGasPriceForNewTransactions(BaseFee(head))
}
