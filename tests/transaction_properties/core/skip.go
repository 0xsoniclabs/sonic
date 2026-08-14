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

// THIS IS WHERE OFFENDING TRANSACTIONS ARE DROPPED.
//
// A transaction that provokes a defect the client has not fixed yet is not injected at all, so a run
// stays green and says in one line what it stepped around. Every policy deciding that lives in the
// skip.go of the domain it belongs to:
//
//	regular/skip.go     a contract creation whose value its sender may not be able to transfer
//	subsidies/skip.go   a sponsorship request whose value does not fit in 256 bits
//	bundles/skip.go     the inner domain's policies, applied to a bundle's contents too
//
// To stop skipping something once the client is fixed, delete its case there.

package core

import "math/big"

// dropOffending removes from a drawn batch whatever the domain will not have injected, tallying the
// reason so the run reports how often it stepped around what.
//
// It runs before anything is planned: a nonce assigned to a transaction that never reaches a node
// would leave the rest of that sender's run judged against a sequence position the chain never gets to.
func (r *Runner) dropOffending(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
) []TxSpec {

	kept := make([]TxSpec, 0, len(specs))
	for _, spec := range specs {
		if reason := r.Domain.Skip(spec, senders[SenderOf(spec, senders)], baseFee); reason != "" {
			r.skipped[reason]++
			continue
		}
		kept = append(kept, spec)
	}
	return kept
}
