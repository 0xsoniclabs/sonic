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

package opera

import (
	"math"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/params"
)

// MaxTxExecutionGas returns the transaction gas that still fits into an event
// with the largest possible overhead.
func (r Rules) MaxTxExecutionGas() uint64 {
	maxEmptyEventGas := r.Economy.Gas.EventGas +
		uint64(r.Dag.MaxParents-r.Dag.MaxFreeParents)*r.Economy.Gas.ParentGas +
		uint64(r.Dag.MaxExtraData)*r.Economy.Gas.ExtraDataGas
	if r.Economy.Gas.MaxEventGas < maxEmptyEventGas {
		return 0
	}
	return r.Economy.Gas.MaxEventGas - maxEmptyEventGas
}

// MaxTxGasLimit returns the largest gas limit a transaction can be included
// with. Before Canto, the whole gas limit is execution gas. From Canto on, the
// gas limit above MaxTxExecutionGas funds the EIP-8037 state-gas reservoir, so
// it is only bounded by the gas a single block can hold.
func (r Rules) MaxTxGasLimit() uint64 {
	if !r.Upgrades.Canto {
		return r.MaxTxExecutionGas()
	}
	blockGas := r.Blocks.MaxBlockGas
	if r.Upgrades.SingleProposerBlockFormation {
		blockGas = inter.GetEffectiveGasLimit(math.MaxInt64, r.Economy.ShortGasPower.AllocPerSec, blockGas)
	}
	return min(blockGas, params.MaxTxTotalGas)
}
