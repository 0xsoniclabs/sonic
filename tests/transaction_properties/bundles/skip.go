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

package bundles

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
)

// defect3 is why a bare root is only drawn where the transaction cannot fail, and what to delete once
// it is fixed.
//
// A bundle whose execution plan is a bare step, and whose transaction fails, keeps what that
// transaction did while the transaction reaches no block: runTransactionBundleInternal takes no
// snapshot of its own (evmcore/state_processor.go:562), so a plan whose root is a group is rolled back
// and one whose root is a bare step is not. The chain's state then no longer follows from its blocks.
// A bare root is reachable from the public API -- sonic_submitBundle drops the group around a single
// unflagged step (api/sonicapi/execution_plan.go:154) -- and every other test wraps its root in a
// group, so nothing covered this shape.
//
// Domain.build therefore turns a bare root into AllOf(A) unless the model can promise the transaction
// cannot fail. Once the block processor reverts a failed bare root, drop that condition there.
const defect3 = "[3] a bundle whose bare root fails keeps what its transaction did " +
	"(bundles/skip.go, avoided in Domain.build)"

func (d *Domain) Notes() []string {
	if d.cfg.Latest {
		return d.Inner.Notes()
	}
	return append(d.Inner.Notes(), defect3)
}

// Skip asks the domain being wrapped, which answers for an envelope too: a carrier is a call to an
// address with no code and carries no value, so no policy applies to one.
func (d *Domain) Skip(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) string {
	return d.Inner.Skip(spec, sender, baseFee)
}

// dropOffending removes from a bundle's contents whatever the inner domain will not have injected. A
// bundle is a batch like any other, so this happens before the contents are planned: a nonce given to a
// transaction that never reaches a node would leave the rest of them judged against a position nothing
// takes.
//
// In the order given rather than in nonce order, because nothing reorders the inside of a bundle: the
// execution plan runs the contents in the order it references them, so that is the order in which one
// of them takes the balance the next is judged against.
func (d *Domain) dropOffending(contents []core.TxSpec, baseFee *big.Int) []core.TxSpec {
	return core.DropOffendingInGivenOrder(contents, d.inner, baseFee, d.Inner.Skip,
		func(reason string) { d.skipped[reason]++ })
}
