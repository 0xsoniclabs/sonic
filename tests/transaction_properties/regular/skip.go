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

// defect1 is why the creation below is not injected, and what to delete once it is fixed.
//
// A contract creation whose value its sender cannot transfer executes and gets a receipt without
// spending its sender's nonce, so the accounting sees a nonce behind the transactions that ran.
// Sonic sets vm.Config.InsufficientBalanceIsNotAnError, which turns a shortfall into a revert. For a
// call that is consistent, since the state processor advances the nonce before entering the EVM; a
// creation's nonce is advanced by EVM.create (core/vm/evm.go:530), which tests CanTransfer first and
// gives up before reaching its own SetNonce.
const defect1 = "[1] a contract creation whose value its sender cannot transfer keeps its nonce " +
	"(regular/skip.go)"

func (d Domain) Notes() []string {
	return []string{defect1}
}

// Skip drops a contract creation whose value its sender may not be able to hand over -- see defect1.
// Nothing else needs it: a call that cannot cover its value reverts, spends its nonce and pays for its
// gas, which is what the model expects.
func (d Domain) Skip(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) string {
	if !spec.IsCreate() {
		return ""
	}

	// What is left of the balance once the gas is bought, priced at the top of the band the base fee
	// may have moved into: a creation that clearly holds its value never reaches the defect.
	left := new(big.Int).Sub(sender.Balance, core.Affordability(spec, core.Scale(baseFee, 4, 1)))
	if left.Sign() > 0 && spec.Amount().Cmp(left) <= 0 {
		return ""
	}
	return defect1
}
