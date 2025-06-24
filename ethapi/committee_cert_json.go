// Copyright 2025 Sonic Operations Ltd
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

package ethapi

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

// CommitteeCertificate is a JSON representation of a committee certificate
// as returned by the Sonic API. This type provides a conversion between the
// internal certificate representation and the JSON representation provided to
// the API clients. The external API is expected to be stable over time and
// should only be updated in a backward compatible way.
type CommitteeCertificate struct {
	ChainId   uint64                    `json:"chainId"`
	Period    uint64                    `json:"period"`
	Members   []scc.Member              `json:"members"`
	Signers   cert.BitSet[scc.MemberId] `json:"signers"`
	Signature bls.Signature             `json:"signature"`
}

func (c CommitteeCertificate) ToCertificate() cert.CommitteeCertificate {
	aggregatedSignature := cert.NewAggregatedSignature[cert.CommitteeStatement](c.Signers, c.Signature)

	newCert := cert.NewCertificateWithSignature(
		cert.NewCommitteeStatement(
			c.ChainId,
			scc.Period(c.Period),
			scc.NewCommittee(c.Members...)),
		aggregatedSignature)

	return newCert
}

func toJsonCommitteeCertificate(c cert.CommitteeCertificate) CommitteeCertificate {
	sub := c.Subject()
	agg := c.Signature()
	return CommitteeCertificate{
		ChainId:   sub.ChainId,
		Period:    uint64(sub.Period),
		Members:   sub.Committee.Members(),
		Signers:   agg.Signers(),
		Signature: agg.Signature(),
	}
}
