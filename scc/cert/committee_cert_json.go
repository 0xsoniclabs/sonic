package cert

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/utils/jsonhex"
)

type committeeCertificateJson struct {
	ChainId   uint64          `json:"chainId"`
	Period    uint64          `json:"period"`
	Members   []scc.Member    `json:"members"`
	Signers   jsonhex.Bytes   `json:"signers"`
	Signature jsonhex.Bytes96 `json:"signature"`
}

func (c committeeCertificateJson) String() string {
	return fmt.Sprintf(`{"chainId":%d,"period":%d,"members":%v,"signers":"%v","signature":"%v"}`,
		c.ChainId, c.Period, c.Members, c.Signers, c.Signature)
}

func (c committeeCertificateJson) ToCommitteeCertificate() (Certificate[CommitteeStatement], error) {
	aggregatedSignature := AggregatedSignature[CommitteeStatement]{}
	aggregatedSignature.signers.mask = c.Signers
	var err error
	aggregatedSignature.signature, err = bls.DeserializeSignature(c.Signature)
	if err != nil {
		return Certificate[CommitteeStatement]{}, fmt.Errorf("failed to deserialize signature: %w", err)
	}

	newCert := NewCertificate(CommitteeStatement{
		statement: statement{
			ChainId: c.ChainId,
		},
		Period:    scc.Period(c.Period),
		Committee: scc.NewCommittee(c.Members...),
	})
	newCert.signature = aggregatedSignature
	return newCert, nil
}

func CommitteeCertificateToJson(c Certificate[CommitteeStatement]) committeeCertificateJson {
	return committeeCertificateJson{
		ChainId:   c.subject.ChainId,
		Period:    uint64(c.subject.Period),
		Members:   c.subject.Committee.Members(),
		Signers:   c.signature.signers.mask,
		Signature: jsonhex.Bytes96(c.signature.signature.Serialize()),
	}
}
