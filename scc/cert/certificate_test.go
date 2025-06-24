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
