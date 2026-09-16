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

package subsidies

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
)

// defect4 is why such a request is never drawn, and what to delete once it is fixed.
//
// A sponsorship request whose value does not fit in 256 bits panics the proposing node while it forms
// the block: createChooseFundInput writes the value with tx.Value().FillBytes(make([]byte, 32))
// (gossip/blockproc/subsidies/subsidies.go:454), which panics on anything wider, while the fee beside
// it is range-checked. A legacy, access-list or dynamic-fee transaction holds its value as a big.Int,
// so the field can carry one, and the block formation filter asks the registry who would pay before
// ValidateTxStatic drops it.
const defect4 = "[4] a sponsorship request whose value exceeds 256 bits panics the node " +
	"(subsidies/skip.go)"

func (d *Domain) Notes() []string {
	if d.cfg.Latest {
		return d.Inner.Notes()
	}
	return append(d.Inner.Notes(), defect4)
}

// Skip asks the domain being wrapped: sponsorship changes who pays for a transaction, not which
// transactions the client cannot survive. What must not be *sponsored* is kept out by unsponsorable
// below instead, because such a transaction is still worth injecting on its own account.
func (d *Domain) Skip(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) string {
	return d.Inner.Skip(spec, sender, baseFee)
}

// unsponsorable reports what this domain will not ask to have sponsored. Whatever it names keeps the
// price it was drawn with, and is dropped from the batch if that price made a request of it anyway.
func (d *Domain) unsponsorable(spec core.TxSpec) bool {
	if spec.Amount().BitLen() > 256 && !d.cfg.Latest {
		return true // defect4, left to bring the node down on the newest fork
	}

	// A registry that sponsors every sender sponsors a stranger too, so a signature recovering to an
	// address holding nothing executes rather than being refused for want of gas. It then moves an
	// account nobody here claimed, which is beyond what this harness observes: the accounting watches
	// the pooled sender the spec names, and the nonce that transaction spends is the stranger's. A
	// fund-backed registry has no such case -- no fund covers a stranger -- so it keeps them.
	return d.Mode != ModeFundBacked && spec.SigningMode().RecoversToStranger()
}
