package cert

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/stretchr/testify/require"
)

func TestCertificate_NewCertificate_ContainsStatement(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)
	cert := NewCertificate(stmt)
	require.Equal(stmt, cert.Subject())
	require.NotNil(cert.Signature())
}

func TestCertificate_NewCertificateWithSignature_ContainsStatementAndSignature(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)
	sig := bls.Signature{}
	AggregatedSignature := NewAggregatedSignature[testStatement](BitSet[scc.MemberId]{}, sig)
	cert := NewCertificateWithSignature(stmt, AggregatedSignature)
	require.Equal(stmt, cert.Subject())
	require.Equal(AggregatedSignature, cert.Signature())
}

func TestCertificate_NewCertificateWithSignature_DeepCopiesInput(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)
	sig := bls.Signature{}
	AggregatedSignature := NewAggregatedSignature[testStatement](BitSet[scc.MemberId]{}, sig)
	cert := NewCertificateWithSignature(stmt, AggregatedSignature)
	err := AggregatedSignature.Add(0, Signature[testStatement]{Signature: bls.Signature{}})
	require.NoError(err)
	require.NotEqual(AggregatedSignature, cert.Signature())
}

func TestCertificate_Signature_ReturnsIndependentCopy(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)
	cert := NewCertificate(stmt)
	sig := cert.Signature()
	err := sig.Add(0, Signature[testStatement]{Signature: bls.Signature{}})
	require.NoError(err)
	require.NotEqual(sig, cert.Signature())
}

func TestCertificate_Statement_ReturnsIndependentCopy(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)
	cert := NewCertificate(stmt)
	stmt = cert.Subject()
	err := cert.Add(1, Signature[testStatement]{
		Signature: bls.NewPrivateKey().Sign([]byte{1}),
	})
	require.NoError(err)
	require.NotEqual(stmt, cert.Subject())
}

func TestCertificate_CanBeGraduallyAccumulatedAndVerified(t *testing.T) {
	requires := require.New(t)
	keys := make([]bls.PrivateKey, 6)
	members := make([]scc.Member, 6)
	for i := range keys {
		keys[i] = bls.NewPrivateKey()
		members[i] = newMember(keys[i], 1)
	}
	committee := scc.NewCommittee(members...)
	requires.NoError(committee.Validate())

	stmt := testStatement(1)

	cert := NewCertificate(stmt)
	for i, key := range keys {
		sig := Sign(stmt, key)
		requires.NoError(cert.Add(scc.MemberId(i), sig))

		err := cert.Verify(committee)
		if i < 4 { // needs more than 2/3 to be valid
			requires.Error(err)
		} else {
			requires.NoError(err)
		}
	}
}

func TestCertificate_VerifyAuthority_AcceptsDifferentAuthority(t *testing.T) {
	require := require.New(t)
	key0 := bls.NewPrivateKey()
	key1 := bls.NewPrivateKey()

	stmt := testStatement(1)

	producer := scc.NewCommittee(newMember(key0, 1), newMember(key1, 2))
	require.NoError(producer.Validate())

	authority := scc.NewCommittee(newMember(key0, 1))
	require.NoError(producer.Validate())

	// Initially the certificate is for nobody valid.
	cert := NewCertificate(stmt)
	require.Error(cert.VerifyAuthority(producer, producer))
	require.Error(cert.VerifyAuthority(authority, producer))

	// A signature of key 0 makes it valid for the authority.
	require.NoError(cert.Add(0, Sign(stmt, key0)))
	require.Error(cert.VerifyAuthority(producer, producer))
	require.NoError(cert.VerifyAuthority(authority, producer))

	// A signature of key 1 makes it valid for the producer as well.
	require.NoError(cert.Add(1, Sign(stmt, key1)))
	require.NoError(cert.VerifyAuthority(producer, producer))
	require.NoError(cert.VerifyAuthority(authority, producer))
}
