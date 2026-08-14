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
	"pgregory.net/rapid"
)

// The dimensions an envelope is drawn over. A bundle that executes is what the checks are for, so the
// shapes that keep one from executing are the minority -- and, since rapid's index draw favours the
// front of a list, the order is what makes that land. See the note on the shared generators in
// core/generators.go.
var (
	// bundledWeight is how often a batch carries a bundle at all: two times in three. An ordinary
	// batch is already covered by the domain being wrapped.
	bundledWeight = rapid.SampledFrom([]bool{true, true, false})

	// A range or a gas limit that keeps the bundle from running is worth a handful of draws, no more:
	// each is one gate, and a run spent on gates is a run that never reaches the block a bundle has to
	// land in whole.
	ranges = rapid.SampledFrom([]RangeChoice{
		RangeCovering, RangeCovering, RangeCovering, RangeCovering,
		RangeCovering, RangeCovering, RangeCovering, RangeCovering,
		RangeBefore, RangeAfter,
	})

	// roots offer the bare single step as often as the group, because it is the shape nothing else
	// generates and the one the block processor handles without a snapshot of its own.
	roots = rapid.SampledFrom([]RootChoice{RootAllOf, RootAllOf, RootSingle, RootSingle})

	declarations = rapid.SampledFrom([]GasChoice{
		GasExact, GasExact, GasExact, GasExact,
		GasExact, GasExact, GasExact, GasExact,
		GasTooLow, GasTooHigh,
	})

	// prices are what an envelope offers for its gas, which matters in only one of the three regimes:
	// where bundles are disabled or unknown, an envelope is an ordinary call and has to pay its way like
	// one. Where they are enabled nothing charges it, and a price wide enough to overflow its cost would
	// only get the carrier deleted before anything is opened -- which the ordinary domain covers
	// already, and which would cost a bundle here for a reason that has nothing to do with bundles.
	prices = rapid.SampledFrom([]*big.Int{
		core.MinimumViableFeeCap,
		core.MinimumViableFeeCap,
		new(big.Int).Mul(core.MinimumViableFeeCap, big.NewInt(10)),
		big.NewInt(1),
	})

	// counts is how many of the drawn contents a bundle keeps. A group tolerates no failure, so every
	// transaction it carries is another chance for the bundle not to run at all -- and a bundle that
	// never runs checks far less than one that does. One leads for that reason; two is the smallest
	// number for which all-or-nothing means anything, so it is next.
	counts = rapid.SampledFrom([]int{1, 1, 2, 2, 3})
)

// Bundling replaces one of the transactions another generator drew with an envelope carrying a batch
// of its own, drawn from that same generator: a bundle is a batch inside a batch, and nothing about a
// batch has to change to be one.
//
// The transaction it replaces is not wasted -- its envelope capability becomes the envelope's own, so
// who sends the bundle, where it sits in that sender's sequence and how it is signed are still drawn
// rather than fixed. At most one envelope is drawn per batch, so that two bundles never compete for
// one nonce; the contents' senders live in their own window of the claimed accounts, so they never
// compete with the batch around them either.
func Bundling(inner *rapid.Generator[[]core.TxSpec]) *rapid.Generator[[]core.TxSpec] {
	return rapid.Custom(func(t *rapid.T) []core.TxSpec {
		specs := inner.Draw(t, "batch")
		if !bundledWeight.Draw(t, "bundled") {
			return specs
		}

		carrier := rapid.IntRange(0, len(specs)-1).Draw(t, "carrier")
		root := roots.Draw(t, "root")
		contents := inner.Draw(t, "contents")
		keep := counts.Draw(t, "contentCount")
		if root == RootSingle {
			keep = 1 // a bare step carries one transaction, since there is no group to hold more
		}
		contents = contents[:min(keep, len(contents))]

		out := make([]core.TxSpec, len(specs))
		copy(out, specs)
		out[carrier] = &Envelope{
			Envelope:    envelopeOf(specs[carrier]),
			SinglePrice: core.SinglePrice{GasPrice: prices.Draw(t, "envelopePrice")},
			Contents:    contents,
			Range:       ranges.Draw(t, "range"),
			Root:        root,
			Declared:    declarations.Draw(t, "declaredGas"),
		}
		return out
	})
}

// envelopeOf takes over the drawn envelope of the transaction being replaced, apart from two fields.
// The gas limit, because an envelope's is dictated by its contents and Envelope.Gas answers from what
// was built. And the nonce, which an envelope never spends -- nothing charges a carrier, so its nonce
// is never checked and never advances. The one place it would matter is the static check that refuses
// the maximum uint64, and taking a bundle down for that would cost a draw for a reason that has
// nothing to do with bundles. The ordinary domain covers that rule.
func envelopeOf(spec core.TxSpec) core.Envelope {
	return core.Envelope{
		SenderIdx: spec.Sender(),
		Nonce:     core.NonceCorrect,
		Signing:   spec.SigningMode(),
	}
}

// envelopesIn lists the envelopes of a batch, which is at most one but reads better as a loop than as
// a special case everywhere.
func envelopesIn(specs []core.TxSpec) []*Envelope {
	out := make([]*Envelope, 0, 1)
	for _, spec := range specs {
		if envelope, ok := spec.(*Envelope); ok {
			out = append(out, envelope)
		}
	}
	return out
}
