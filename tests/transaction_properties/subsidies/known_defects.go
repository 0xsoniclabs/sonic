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

package subsidies

import "github.com/0xsoniclabs/sonic/tests/transaction_properties/core"

// Notes reports the defect below with every run, alongside the ones the domain being wrapped had to
// avoid. The transaction that provokes it is never asked to be sponsored, so a run cannot reproduce
// it -- and without saying so, a run would look exactly like a run against a client that had been
// fixed.
func (d *Domain) Notes() []core.DefectNote {
	return append(d.Inner.Notes(), core.DefectNote{
		Summary: "A sponsorship request whose value does not fit in 256 bits panics the node while " +
			"the block is being formed.",
		Avoidance: "it takes the node down rather than producing a wrong answer",
		Detail: `
			Found by this domain.

			  subsidies.createChooseFundInput   gossip/blockproc/subsidies/subsidies.go:454

			The call is assembled by hand, one 32-byte word at a time, and the value is written with
			tx.Value().FillBytes(make([]byte, 32)) -- which panics on a value too wide to fit, unlike
			every other word of the input. The fee beside it is range-checked; the value is not.

			A transaction whose value is wider than 256 bits is not exotic here: a legacy, access-list
			or dynamic-fee transaction holds its value as a big.Int, so the field can carry one, and
			ValidateTxStatic drops such a transaction from Brio onwards -- but only after the block
			formation filter has already asked the registry who would pay for it. The panic therefore
			happens on the proposing node, while it is building a block, for a transaction nothing
			would have executed.

			The harness keeps such a transaction from asking to be sponsored, rather than out of the
			batch: it still exercises the value range against everything else, it just never reaches
			chooseFund -- see Domain.unsponsorable. Once the value is range-checked like the fee, drop
			that condition: a request carrying an impossible value is exactly the shape worth drawing.`,
	})
}
