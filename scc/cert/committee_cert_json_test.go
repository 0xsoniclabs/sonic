package cert

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/utils/jsonhex"
	"github.com/stretchr/testify/require"
)

func TestCommitteeCertificateJson_String(t *testing.T) {
	var c committeeCertificateJson
	want := fmt.Sprintf(`{"chainId":0,"period":0,"members":[],"signers":"%v","signature":"%v"}`,
		jsonhex.Bytes{}, jsonhex.Bytes96{})
	require.Equal(t, want, c.String())
}

func TestCommitteeCertificateJson_ToCommitteeCertificate_ConvertsToHealthyCertificate(t *testing.T) {
	require := require.New(t)
	bitSet := BitSet[scc.MemberId]{}
	bitSet.Add(1)
	sig := bls.Signature{}

	c := committeeCertificateJson{
		ChainId:   123,
		Period:    456,
		Members:   []scc.Member{},
		Signers:   jsonhex.Bytes(bitSet.mask),
		Signature: jsonhex.Bytes96(sig.Serialize()),
	}

	got, err := c.ToCommitteeCertificate()
	require.NoError(err)
	want := NewCertificate(CommitteeStatement{
		statement: statement{
			ChainId: c.ChainId,
		},
		Period:    scc.Period(c.Period),
		Committee: scc.NewCommittee(c.Members...),
	})
	aggregatedSignature := AggregatedSignature[CommitteeStatement]{}
	aggregatedSignature.signers.mask = c.Signers
	aggregatedSignature.signature, err = bls.DeserializeSignature(c.Signature)
	want.signature = aggregatedSignature
	require.Equal(want, got)
}

func TestCommitteeCertificateJson_ToCommitteeCertificate_FailsWithInvalidAggregatedSignature(t *testing.T) {
	require := require.New(t)

	c := committeeCertificateJson{
		ChainId:   123,
		Period:    456,
		Members:   []scc.Member{},
		Signers:   jsonhex.Bytes{},
		Signature: jsonhex.Bytes96{},
	}

	got, err := c.ToCommitteeCertificate()
	require.ErrorContains(err, "failed to deserialize signature")
	require.Equal(Certificate[CommitteeStatement]{}, got)
}

func TestCommitteeCertificateJson_CanBeJsonEncodedAndDecoded(t *testing.T) {
	require := require.New(t)

	// setup
	private := bls.NewPrivateKey()
	member := scc.Member{
		PublicKey:         private.PublicKey(),
		ProofOfPossession: private.GetProofOfPossession(),
		VotingPower:       1,
	}
	c := NewCertificate(CommitteeStatement{
		statement: statement{
			ChainId: 123,
		},
		Period:    456,
		Committee: scc.NewCommittee(member),
	})
	agg := AggregatedSignature[CommitteeStatement]{}
	agg.Add(1, Signature[CommitteeStatement]{Signature: private.Sign(c.subject.GetDataToSign())})
	c.signature = agg

	// encode
	certJson := CommitteeCertificateToJson(c)
	data, err := json.Marshal(certJson)
	require.NoError(err)

	// decode
	var certJson2 committeeCertificateJson
	err = json.Unmarshal(data, &certJson2)
	require.NoError(err)

	// check
	cert, err := certJson2.ToCommitteeCertificate()
	require.NoError(err)
	require.Equal(c, cert)
}

func TestCommitteeCertificateToJson(t *testing.T) {
	require := require.New(t)
	bitSet := BitSet[scc.MemberId]{}
	bitSet.Add(1)
	sig := bls.Signature{}

	c := NewCertificate(CommitteeStatement{
		statement: statement{
			ChainId: 123,
		},
		Period:    456,
		Committee: scc.NewCommittee(),
	})
	aggregatedSignature := AggregatedSignature[CommitteeStatement]{}
	aggregatedSignature.signers.mask = bitSet.mask
	aggregatedSignature.signature = sig
	c.signature = aggregatedSignature

	want := committeeCertificateJson{
		ChainId:   123,
		Period:    456,
		Members:   []scc.Member(nil),
		Signers:   jsonhex.Bytes(bitSet.mask),
		Signature: jsonhex.Bytes96(sig.Serialize()),
	}
	got := CommitteeCertificateToJson(c)
	require.Equal(want, got)
}
