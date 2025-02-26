package ethapi

import (
	"encoding/json"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/stretchr/testify/require"
)

func TestCommitteeCertificateJson_ToCommitteeCertificate_ConvertsToHealthyCertificate(t *testing.T) {
	require := require.New(t)
	bitSet := cert.BitSet[scc.MemberId]{}
	bitSet.Add(1)
	sig := bls.Signature{}

	c := committeeCertificateJson{
		ChainId:   123,
		Period:    456,
		Members:   []scc.Member{},
		Signers:   bitSet,
		Signature: sig,
	}

	got := c.ToCommitteeCertificate()
	aggregatedSignature := cert.NewAggregatedSignature[cert.CommitteeStatement](
		c.Signers,
		c.Signature,
	)
	want := cert.NewCertificateWithSignature(cert.NewCommitteeStatement(
		c.ChainId,
		scc.Period(c.Period),
		scc.NewCommittee(c.Members...)),
		aggregatedSignature)

	require.Equal(want, got)
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
	agg := cert.AggregatedSignature[cert.CommitteeStatement]{}
	agg.Add(1, cert.Signature[cert.CommitteeStatement]{Signature: private.Sign([]byte{1})})
	c := cert.NewCertificateWithSignature(
		cert.NewCommitteeStatement(123, 456, scc.NewCommittee(member)),
		agg,
	)

	// encode
	certJson := committeeCertificateToJson(c)
	data, err := json.Marshal(certJson)
	require.NoError(err)

	// decode
	var certJson2 committeeCertificateJson
	err = json.Unmarshal(data, &certJson2)
	require.NoError(err)

	// check
	cert := certJson2.ToCommitteeCertificate()
	require.Equal(c, cert)
}

func TestCommitteeCertificateToJson(t *testing.T) {
	require := require.New(t)
	bitSet := cert.BitSet[scc.MemberId]{}
	bitSet.Add(1)
	sig := bls.Signature{}

	aggregatedSignature := cert.NewAggregatedSignature[cert.CommitteeStatement](
		bitSet,
		sig,
	)
	c := cert.NewCertificateWithSignature(
		cert.NewCommitteeStatement(123, 456, scc.NewCommittee()),
		aggregatedSignature,
	)

	want := committeeCertificateJson{
		ChainId:   123,
		Period:    456,
		Members:   []scc.Member(nil),
		Signers:   bitSet,
		Signature: sig,
	}
	got := committeeCertificateToJson(c)
	require.Equal(want, got)
}
