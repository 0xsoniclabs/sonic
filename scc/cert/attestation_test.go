package cert

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc/bls"
)

func TestAttestation_Attest_ProducesValidAttestation(t *testing.T) {
	key := bls.NewPrivateKey()
	statement := testStatement(1)

	attest := Attest(statement, key)
	if !attest.Verify(key.PublicKey()) {
		t.Error("attestation is not valid")
	}
}

func TestAttestation_Verify_RejectsInvalidSignature(t *testing.T) {
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
			attest := Attest(testStatement(1), key)
			attest.Subject = test.stmt
			if attest.Verify(test.key) {
				t.Error("invalid attestation is reported to be valid")
			}
		})
	}
}
