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

package bundles

import "github.com/0xsoniclabs/sonic/tests/transaction_properties/core"

// Notes reports the defect below with every run. The shape that provokes it is drawn only where the
// transaction it carries cannot fail, so a run cannot reproduce it -- and without saying so, a run
// would look exactly like a run against a client that had been fixed.
func (d *Domain) Notes() []core.DefectNote {
	return append(d.Inner.Notes(), core.DefectNote{
		Summary: "A bundle whose execution plan is a bare step, and whose transaction fails, keeps " +
			"what that transaction did while the transaction reaches no block.",
		Avoidance: "it leaves state no block accounts for",
		Detail: `
			Found by this domain. Observed directly: the transaction of such a bundle has no receipt
			and is in no block, yet its sender's nonce has advanced and its gas has been charged.

			The plan that provokes it, as the production rendering of an execution plan prints it,
			beside the plan every other test builds -- one letter per transaction, and the only
			difference between them is the group:

			  A                a bare root: nothing snapshots it, so a failing A stays applied
			  AllOf(A)         runAllOfGroup snapshots and reverts, so a failing A leaves nothing

			Every failure of this test prints the plan of each bundle in that same form, so a report
			can be read against the two lines above.

			  evmcore.runTransactionBundleInternal  evmcore/state_processor.go:562
			  bundle.RunBundle                     gossip/blockproc/bundle/bundle_processor.go:32

			runTransactionBundleInternal takes no snapshot of its own. The only reverts are the ones
			runAllOfGroup and runOneOfGroup take around a group, so a plan whose root is a group is
			rolled back cleanly. A plan whose root is a bare step has no group to roll it back: the
			transaction is applied, RunBundle reports failure, and the block processor returns an
			empty list of processed transactions without reverting anything. The block then does not
			account for state the chain has, which is what breaks a replay from block data -- the same
			shape as defect 2, and not fixed by Allegro.

			A bare root is reachable from the public API: sonic_submitBundle drops the group around a
			single step with no flags (api/sonicapi/execution_plan.go:154), and so does the builder
			for bundle.Step(...).Build(). The pool's trial run would refuse a bundle that fails at the
			head state, but a bundle that passes there and fails at execution time, or one proposed by
			a validator that does not pre-check, reaches this path.

			Every existing test wraps its root in AllOf or OneOf, so nothing covered this shape.

			The harness draws a bare root only where the model can promise the transaction cannot
			revert -- a call to an account with no code, ample gas, and a value its sender clearly
			holds -- and turns the plan into a group anywhere else. Once the block processor reverts a
			failed bare root, drop that condition in Domain.build: the shape is worth drawing whatever
			the transaction does.`,
	})
}
