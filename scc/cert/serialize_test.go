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
	"github.com/0xsoniclabs/sonic/scc/cert/pb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestCertificate_Serializable_UnserializableStatementReportsError(t *testing.T) {
	cert := Certificate[testStatement]{}
	_, err := cert.Serialize()
	require.ErrorContains(t, err, "unsupported subject type")
}

func TestCertificate_Deserialize_UnserializableStatementReportsError(t *testing.T) {
	cert := Certificate[testStatement]{}
	err := cert.Deserialize([]byte{1, 2, 3})
	require.ErrorContains(t, err, "unsupported subject type")
}

func TestCommitteeCertificate_Serialize_ResultCanBeUnMarshaled(t *testing.T) {
	tests := map[string]Certificate[CommitteeStatement]{
		"default": {},
		"example": getExampleCommitteeCertificate(),
	}

	for name, certificate := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			data, err := certificate.Serialize()
			require.NoError(err)

			var restored Certificate[CommitteeStatement]
			require.NoError(restored.Deserialize(data))

			require.Equal(certificate, restored)
		})
	}
}

func TestCommitteeCertificate_Deserialize_CorruptedDataCanNotBeUnmarshaled(t *testing.T) {
	require.Error(t, new(Certificate[CommitteeStatement]).Deserialize([]byte{1, 2, 3}))
}

func TestCommitteeCertificate_Deserialize_DetectsIssues(t *testing.T) {

	keyLength := len(bls.PublicKey{}.Serialize())
	proofLength := len(bls.Signature{}.Serialize())

	tests := map[string]struct {
		corrupt func(*pb.CommitteeCertificate)
		expect  string
	}{
		"no signature": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Signature = nil
			},
			expect: "no signature",
		},
		"no public key": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].PublicKey = nil
			},
			expect: "invalid public key length",
		},
		"too short public key": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].PublicKey = make([]byte, keyLength-1)
			},
			expect: "invalid public key length",
		},
		"too long public key": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].PublicKey = make([]byte, keyLength+1)
			},
			expect: "invalid public key length",
		},
		"invalid public key": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].PublicKey = make([]byte, keyLength)
			},
			expect: "failed to decode public key",
		},
		"no proof of possession": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].ProofOfPossession = nil
			},
			expect: "invalid proof of possession length",
		},
		"too short proof of possession": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].ProofOfPossession = make([]byte, proofLength-1)
			},
			expect: "invalid proof of possession length",
		},
		"too long proof of possession": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].ProofOfPossession = make([]byte, proofLength+1)
			},
			expect: "invalid proof of possession length",
		},
		"invalid proof of possession": {
			corrupt: func(c *pb.CommitteeCertificate) {
				c.Members[0].ProofOfPossession = make([]byte, proofLength)
			},
			expect: "failed to decode proof of possession",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			certificate := getExampleCommitteeCertificate()
			data, err := certificate.Serialize()
			require.NoError(err)

			var pb pb.CommitteeCertificate
			require.NoError(proto.Unmarshal(data, &pb))

			test.corrupt(&pb)
			data, err = proto.Marshal(&pb)
			require.NoError(err)

			var restored Certificate[CommitteeStatement]
			err = restored.Deserialize(data)
			require.ErrorContains(err, test.expect)
		})
	}
}

func TestBlockCertificate_Serialize_ResultCanBeUnMarshaled(t *testing.T) {
	tests := map[string]Certificate[BlockStatement]{
		"default": {},
		"example": getExampleBlockCertificate(),
	}

	for name, certificate := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			data, err := certificate.Serialize()
			require.NoError(err)

			var restored Certificate[BlockStatement]
			require.NoError(restored.Deserialize(data))

			require.Equal(certificate, restored)
		})
	}
}

func TestBlockCertificate_Deserialize_CorruptedDataCanNotBeUnmarshaled(t *testing.T) {
	require.Error(t, new(Certificate[BlockStatement]).Deserialize([]byte{1, 2, 3}))
}

func TestBlockCertificate_Deserialize_DetectsIssues(t *testing.T) {

	tests := map[string]struct {
		corrupt func(*pb.BlockCertificate)
		expect  string
	}{
		"no signature": {
			corrupt: func(c *pb.BlockCertificate) {
				c.Signature = nil
			},
			expect: "no signature",
		},
		"no hash": {
			corrupt: func(c *pb.BlockCertificate) {
				c.Hash = nil
			},
			expect: "invalid hash length",
		},
		"too short hash": {
			corrupt: func(c *pb.BlockCertificate) {
				c.Hash = make([]byte, 31)
			},
			expect: "invalid hash length",
		},
		"too long hash": {
			corrupt: func(c *pb.BlockCertificate) {
				c.Hash = make([]byte, 33)
			},
			expect: "invalid hash length",
		},
		"no state root": {
			corrupt: func(c *pb.BlockCertificate) {
				c.StateRoot = nil
			},
			expect: "invalid state root length",
		},
		"too short state root": {
			corrupt: func(c *pb.BlockCertificate) {
				c.StateRoot = make([]byte, 31)
			},
			expect: "invalid state root length",
		},
		"too long state root": {
			corrupt: func(c *pb.BlockCertificate) {
				c.StateRoot = make([]byte, 33)
			},
			expect: "invalid state root length",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			certificate := getExampleBlockCertificate()
			data, err := certificate.Serialize()
			require.NoError(err)

			var pb pb.BlockCertificate
			require.NoError(proto.Unmarshal(data, &pb))

			test.corrupt(&pb)
			data, err = proto.Marshal(&pb)
			require.NoError(err)

			var restored Certificate[BlockStatement]
			err = restored.Deserialize(data)
			require.ErrorContains(err, test.expect)
		})
	}
}

func TestAggregatedSignature_UnmarshalDetectsIssues(t *testing.T) {

	sigLength := len(bls.Signature{}.Serialize())

	tests := map[string]struct {
		corrupt func(*pb.AggregatedSignature)
		expect  string
	}{
		"no signature": {
			corrupt: func(s *pb.AggregatedSignature) {
				s.Signature = nil
			},
			expect: "invalid signature length",
		},
		"too short signature": {
			corrupt: func(s *pb.AggregatedSignature) {
				s.Signature = make([]byte, sigLength-1)
			},
			expect: "invalid signature length",
		},
		"too long signature": {
			corrupt: func(s *pb.AggregatedSignature) {
				s.Signature = make([]byte, sigLength+1)
			},
			expect: "invalid signature length",
		},
		"corrupted signature": {
			corrupt: func(s *pb.AggregatedSignature) {
				s.Signature = make([]byte, sigLength)
			},
			expect: "failed to decode signature",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			signature := bls.Signature{}.Serialize()
			sig := &pb.AggregatedSignature{
				Signature: signature[:],
			}
			_, err := fromProtoSignature[CommitteeStatement](sig)
			require.NoError(err)

			test.corrupt(sig)
			_, err = fromProtoSignature[CommitteeStatement](sig)
			require.ErrorContains(err, test.expect)
		})
	}

}

func BenchmarkBlockCertificate_Marshaling(b *testing.B) {
	certificate := getExampleBlockCertificate()
	b.ResetTimer()
	for range b.N {
		_, err := certificate.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockCertificate_Unmarshaling(b *testing.B) {
	certificate := getExampleBlockCertificate()
	data, err := certificate.Serialize()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		var cert Certificate[BlockStatement]
		if err := cert.Deserialize(data); err != nil {
			b.Fatal(err)
		}
	}
}

func getExampleCommitteeCertificate() Certificate[CommitteeStatement] {

	members := make([]scc.Member, 50)
	for i := range members {
		members[i] = newMember(bls.NewPrivateKeyForTests(byte(i)), 10)
	}

	certificate := NewCertificate(CommitteeStatement{
		statement: statement{
			ChainId: 123,
		},
		Period:    456,
		Committee: scc.NewCommittee(members...),
	})

	sig := Sign(certificate.subject, bls.NewPrivateKey())
	for i := scc.MemberId(1); i <= 256; i *= 2 {
		if err := certificate.Add(i, sig); err != nil {
			panic("failed to add signature")
		}
	}
	return certificate
}

func getExampleBlockCertificate() Certificate[BlockStatement] {
	certificate := NewCertificate(BlockStatement{
		statement: statement{
			ChainId: 123,
		},
		Number:    45678,
		Hash:      common.Hash{1, 2, 3},
		StateRoot: common.Hash{4, 5, 6},
	})
	sig := Sign(certificate.subject, bls.NewPrivateKey())
	for i := scc.MemberId(1); i <= 256; i *= 2 {
		if err := certificate.Add(i, sig); err != nil {
			panic("failed to add signature")
		}
	}
	return certificate
}
