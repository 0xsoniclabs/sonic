package cert

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCommitteeCertificate_Serialize_ResultCanBeUnMarshaled(t *testing.T) {
	tests := map[string]Certificate[CommitteeStatement]{
		"default": {},
		"example": getExampleCommitteeCertificate(),
	}

	for name, certificate := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			data, err := SerializeCommitteeCertificate(certificate)
			require.NoError(err)

			restored, err := DeserializeCommitteeCertificate(data)
			require.NoError(err)

			require.Equal(certificate, restored)
		})
	}
}

func TestMarshalBlockCertificate_ResultCanBeUnMarshaled(t *testing.T) {
	tests := map[string]Certificate[BlockStatement]{
		"default": {},
		"example": getExampleBlockCertificate(),
	}

	for name, certificate := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			data, err := SerializeBlockCertificate(certificate)
			require.NoError(err)

			restored, err := DeserializeBlockCertificate(data)
			require.NoError(err)

			require.Equal(certificate, restored)
		})
	}
}

func BenchmarkBlockCertificate_Marshaling(b *testing.B) {
	certificate := getExampleBlockCertificate()
	b.ResetTimer()
	for range b.N {
		_, err := SerializeBlockCertificate(certificate)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockCertificate_Unmarshaling(b *testing.B) {
	certificate := getExampleBlockCertificate()
	data, err := SerializeBlockCertificate(certificate)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		_, err := DeserializeBlockCertificate(data)
		if err != nil {
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
		certificate.Add(i, sig)
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
		certificate.Add(i, sig)
	}
	return certificate
}
