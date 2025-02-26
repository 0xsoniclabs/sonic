package ethapi

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

type committeeCertificateJson struct {
	ChainId   uint64                    `json:"chainId"`
	Period    uint64                    `json:"period"`
	Members   []scc.Member              `json:"members"`
	Signers   cert.BitSet[scc.MemberId] `json:"signers"`
	Signature bls.Signature             `json:"signature"`
}

func (c committeeCertificateJson) ToCommitteeCertificate() cert.Certificate[cert.CommitteeStatement] {
	aggregatedSignature := cert.NewAggregatedSignature[cert.CommitteeStatement](c.Signers, c.Signature)

	newCert := cert.NewCertificateWithSignature(
		cert.NewCommitteeStatement(
			c.ChainId,
			scc.Period(c.Period),
			scc.NewCommittee(c.Members...)),
		aggregatedSignature)

	return newCert
}

func CommitteeCertificateToJson(c cert.Certificate[cert.CommitteeStatement]) committeeCertificateJson {
	sub := c.Subject()
	agg := c.Signature()
	return committeeCertificateJson{
		ChainId:   sub.ChainId,
		Period:    uint64(sub.Period),
		Members:   sub.Committee.Members(),
		Signers:   agg.Signers(),
		Signature: agg.Signature(),
	}
}
