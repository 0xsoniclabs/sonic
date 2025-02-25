package cert

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
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

// func (c CommitteeCertificateJson) ToCommitteeCertificate() (Certificate[CommitteeStatement], error) {
// 	members := make([]scc.Member, len(c.Members))
// 	for i, member := range c.Members {
// 		m, err := member.ToMember()
// 		if err != nil {
// 			return Certificate[CommitteeStatement]{}, err
// 		}
// 		members[i] = m
// 	}
// 	aggregatedSignature, err := c.AggregatedSignature.ToAggregatedSignature()
// 	if err != nil {
// 		return Certificate[CommitteeStatement]{}, err
// 	}
// 	certificate := NewCertificate(CommitteeStatement{
// 		statement: statement{
// 			ChainId: 123,
// 		},
// 		Period:    456,
// 		Committee: scc.NewCommittee(members...),
// 	})
// 	return NewCertificate(CommitteeStatement{
// 		statement: statement{
// 			ChainId: c.ChainId,
// 		},
// 		Period:    c.Period,
// 		Committee: scc.NewCommittee(members...),
// 	}), nil
// }
