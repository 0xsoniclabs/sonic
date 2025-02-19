package cert

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/stretchr/testify/require"
)

func TestSignature_Sign_ProducesValidSignature(t *testing.T) {
	key := bls.NewPrivateKey()
	statement := testStatement(1)

	sig := Sign(statement, key)
	require.Equal(t, key.Sign([]byte{1}), sig.Signature)

	if !sig.Verify(key.PublicKey(), statement) {
		t.Error("signature is not valid")
	}
}

func TestSignature_Verify_RejectsInvalidSignature(t *testing.T) {
	key := bls.NewPrivateKey()
	stmt := testStatement(1)
	tests := map[string]struct {
		key  bls.PublicKey
		stmt testStatement
	}{
		"wrong key": {
			key:  bls.NewPrivateKey().PublicKey(),
			stmt: stmt,
		},
		"wrong statement": {
			key:  key.PublicKey(),
			stmt: stmt + 1,
		},
		"both wrong": {
			key:  bls.NewPrivateKey().PublicKey(),
			stmt: stmt + 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sig := Sign(testStatement(1), key)
			if sig.Verify(test.key, test.stmt) {
				t.Error("invalid signature is reported to be valid")
			}
		})
	}
}

type testStatement byte

func (s testStatement) GetDataToSign() []byte {
	return []byte{byte(s)}
}
