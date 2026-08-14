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

// Package subsidies layers gas subsidies over another domain's transactions: it takes that domain's
// generator and turns drawn transactions into sponsorship requests, takes its pricing rules and
// replaces them where a request pays from a fund rather than from its sender's balance, arranges the
// funding each iteration needs, and checks the post-execution transactions only this domain knows
// about. Nothing here is known to core or to the domain being wrapped.
package subsidies

import (
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"pgregory.net/rapid"
)

// sponsorshipWeight is how often a drawn transaction is turned into a sponsorship request: three
// times in four. Requests are the point of this domain, but a batch that never mixes them with
// ordinary transactions would never exercise the two sharing a sender's nonce sequence.
var sponsorshipWeight = rapid.SampledFrom([]bool{true, true, true, false})

// Sponsoring turns some of the transactions another generator drew into sponsorship requests, which
// is all a sponsorship request is: an ordinary transaction offering nothing for its gas. The prices
// are zeroed on the drawn spec rather than hidden behind a wrapper, so the spec a builder assembles
// and the spec the model judges stay the same object -- a wrapper overriding FeeCap would have to
// override the optional capabilities too, and a static type either has them all or none.
//
// A contract creation is left priced as drawn even when it is picked: production refuses to sponsor
// one, so zeroing it would only produce a transaction nothing can pay for, which the ordinary rules
// already predict without any of this. Whatever else the domain declares unsponsorable is left alone
// the same way, and dropped from the batch if it asked to be sponsored regardless.
func Sponsoring(
	inner *rapid.Generator[[]core.TxSpec],
	unsponsorable func(core.TxSpec) bool,
) *rapid.Generator[[]core.TxSpec] {

	return rapid.Custom(func(t *rapid.T) []core.TxSpec {
		specs := inner.Draw(t, "batch")
		for i, spec := range specs {
			if spec.IsCreate() || unsponsorable(spec) ||
				!sponsorshipWeight.Draw(t, label("sponsored", i)) {
				continue
			}
			spec.(core.ZeroablePrice).ZeroPrices()
		}

		// Leaving a transaction unsponsored is not enough to keep it out of the sponsored set: the
		// ordinary fee caps include zero, so one can ask to be sponsored without this ever touching
		// it. What must not be sponsored is therefore dropped from the batch instead.
		kept := make([]core.TxSpec, 0, len(specs))
		for _, spec := range specs {
			if !IsSponsorshipRequest(spec) || !unsponsorable(spec) {
				kept = append(kept, spec)
			}
		}
		return kept
	})
}

// IsSponsorshipRequest reports whether a spec asks to be sponsored, mirroring
// subsidies.IsSponsorshipRequest: a transaction with a maximum gas price of zero and a recipient.
// It is written from the specification rather than by calling that function, which would make the
// model assert nothing about it.
func IsSponsorshipRequest(spec core.TxSpec) bool {
	return spec.FeeCap().Sign() == 0 && !spec.IsCreate()
}

// requests lists the transactions of a batch that ask to be sponsored.
func requests(specs []core.TxSpec) []core.TxSpec {
	out := make([]core.TxSpec, 0, len(specs))
	for _, spec := range specs {
		if IsSponsorshipRequest(spec) {
			out = append(out, spec)
		}
	}
	return out
}
