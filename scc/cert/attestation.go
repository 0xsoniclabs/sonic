package cert

import (
	"github.com/0xsoniclabs/sonic/scc/bls"
)

// Attestation is a statement attested by a signature. Unlike a Certificate,
// an Attestation is not signed by a full committee, but by a single member.
type Attestation[S Statement] struct {
	Subject   S
	Signature Signature[S]
}

// Attest creates an attestation for the given statement using the given key.
func Attest[S Statement](subject S, key bls.PrivateKey) Attestation[S] {
	return Attestation[S]{
		Subject:   subject,
		Signature: Sign(subject, key),
	}
}

// Verify checks if the attestation is valid for the given key.
func (a Attestation[S]) Verify(key bls.PublicKey) bool {
	return a.Signature.Verify(key, a.Subject)
}
