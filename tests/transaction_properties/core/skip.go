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

import (
	"cmp"
	"math/big"
	"slices"
)

// dropOffending removes from a drawn batch whatever the domain will not have injected, tallying the
// reason so the run reports how often it stepped around what.
func (r *Runner) dropOffending(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
) []TxSpec {

	return DropOffending(specs, senders, baseFee, r.Domain.Skip, func(reason string) {
		r.skipped[reason]++
	})
}

// DropOffending removes from a batch whatever a policy will not have injected, judging each
// transaction against what the ones running before it leave and tallying every reason it gives.
//
// The batch is taken to be reorderable, which an injected one is: the scrambler puts one sender's
// transactions in nonce order whatever order they arrived in, so a creation drawn ahead of the
// transfer that drains its sender still runs behind it. Use DropOffendingInGivenOrder for a batch
// nothing reorders.
//
// It runs before anything is planned: a nonce assigned to a transaction that never reaches a node
// would leave the rest of that sender's run judged against a sequence position the chain never gets to.
func DropOffending(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	skip func(TxSpec, SenderState, *big.Int) string,
	tally func(reason string),
) []TxSpec {

	return dropOffending(specs, senders, baseFee, skip, tally, true)
}

// DropOffendingInGivenOrder is DropOffending for a batch nothing reorders, where the transactions run
// in the order the batch itself gives. A bundle's contents are that: the execution plan runs them in
// the order it references them.
func DropOffendingInGivenOrder(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	skip func(TxSpec, SenderState, *big.Int) string,
	tally func(reason string),
) []TxSpec {

	return dropOffending(specs, senders, baseFee, skip, tally, false)
}

func dropOffending(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	skip func(TxSpec, SenderState, *big.Int) string,
	tally func(reason string),
	reorderable bool,
) []TxSpec {

	// Dropping a transaction changes the nonces the ones behind it are given, and so the order the rest
	// of the batch runs in, which is what firstOffender below reads. So a refusal ends the walk and the whole
	// thing starts again over what is left, rather than carrying on with an order that no longer holds.
	// A batch is a handful of transactions and every pass drops one, so this converges at once.
	kept := slices.Clone(specs)
	for {
		offender, reason := firstOffender(kept, senders, baseFee, skip, reorderable)
		if offender < 0 {
			return kept
		}
		tally(reason)
		kept = slices.Delete(kept, offender, offender+1)
	}
}

// firstOffender asks the policy about every transaction in the order the chain will run them, and
// reports the index of the first it refuses along with why, or -1 when it refuses none.
func firstOffender(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	skip func(TxSpec, SenderState, *big.Int) string,
	reorderable bool,
) (int, string) {

	// What each sender may still hold by the time a transaction of the batch reaches the EVM. A policy
	// is asked what a transaction's sender can cover, and a batch is several transactions from few
	// senders, so the answer has to follow the money: a creation running behind a transfer that hands
	// most of the balance away has to be judged against what that transfer leaves, not against what the
	// sender held before any of it ran. Judging it against the opening balance is how a creation that
	// does provoke defect 1 gets injected anyway.
	//
	// Every transaction is charged the most it could take -- its whole value, and its gas at the top of
	// the band the base fee may move into -- because which of them execute is not settled until the
	// chain has run them. That errs towards skipping, which is the safe direction: a policy exists to
	// keep a defect out of the run, and one skip too many costs coverage while one too few costs a red
	// build over a defect already known.
	left := make([]*big.Int, len(senders))
	for i, sender := range senders {
		left[i] = new(big.Int).Set(sender.Balance)
	}

	for _, index := range executionOrder(specs, senders, reorderable) {
		spec := specs[index]
		i := SenderOf(spec, senders)
		sender := SenderState{Nonce: senders[i].Nonce, Balance: left[i]}
		if reason := skip(spec, sender, baseFee); reason != "" {
			return index, reason
		}

		takes := new(big.Int).Add(spec.Amount(), GasCost(spec, Scale(baseFee, 4, 1)))
		if left[i] = new(big.Int).Sub(left[i], takes); left[i].Sign() < 0 {
			left[i] = new(big.Int)
		}
	}
	return -1, ""
}

// executionOrder indexes a batch in the order the chain will run it. Where the order across senders
// falls is arbitrary -- the running balance is per sender, so only each sender's own sequence says
// anything -- and grouping by sender is what keeps the comparison a total order.
//
// The nonces come from AssignNonces, which depends only on the drawn specs and the senders' on-chain
// nonces and never on what the model expects, so reading them here plans nothing.
func executionOrder(specs []TxSpec, senders []SenderState, reorderable bool) []int {
	order := make([]int, len(specs))
	for i := range order {
		order[i] = i
	}
	if !reorderable {
		return order
	}

	nonces := AssignNonces(specs, senders)
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(SenderOf(specs[a], senders), SenderOf(specs[b], senders)),
			cmp.Compare(nonces[a], nonces[b]),
			cmp.Compare(a, b),
		)
	})
	return order
}
