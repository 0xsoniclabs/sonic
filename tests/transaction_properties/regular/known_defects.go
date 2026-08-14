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

// This file holds the defect this domain found and now has to steer around, and why steering around
// it is all that can be done.

package regular

import "github.com/0xsoniclabs/sonic/tests/transaction_properties/core"

// Notes reports the defect below with every run. Skip keeps the transaction that provokes it out of
// every batch, so a run cannot reproduce it -- and without saying so, a run would look exactly like a
// run against a client that had been fixed.
func (d Domain) Notes() []core.DefectNote {
	return []core.DefectNote{{
		Summary: "A contract creation whose value its sender cannot transfer executes and gets a " +
			"receipt without spending its sender's nonce.",
		Avoidance: "it leaves the sender's nonce behind the transactions that executed",
		Detail: `
			Found by this domain, and observed directly: the creation has a receipt with a failed
			status and its gas has been charged, yet the nonce it was built with is still the one the
			sender's next transaction has to carry.

			  core.stateTransition.execute   core/state_transition.go:503  (clause 6)
			  vm.EVM.create                  core/vm/evm.go:530

			Sonic sets vm.Config.InsufficientBalanceIsNotAnError, so clause 6 -- the check that the
			sender can hand over the value at all -- is skipped and an insufficient balance becomes a
			revert rather than a consensus error. That is deliberate, and for a call it is consistent:
			the state processor advances the nonce itself before entering the EVM, so the transaction
			pays for its gas and spends its nonce like any other failure.

			A creation does not go through that path. Its nonce is advanced by EVM.create, which
			checks CanTransfer first and returns ErrInsufficientBalance before reaching the SetNonce
			below it -- so the only place that would have spent the nonce is the place that gave up.
			The accounting oracle sees an account whose nonce moved by less than the number of its
			transactions that executed, and the next transaction of that sender is judged against a
			sequence position the chain never left.

			The harness therefore keeps out of every batch a creation whose value its sender may not
			hold once its gas is bought -- see Domain.Skip. Once the creation gate spends the nonce it
			was given, drop that condition: the value is worth drawing against the balance for a
			creation exactly as it is for a call.`,
	}}
}
