package cert

import (
	"github.com/0xsoniclabs/sonic/scc/bls"
)

// Signature is a BLS signature of a statement.
type Signature[S Statement] struct {
	Signature bls.Signature
}

// Sign creates a signature for the given statement using the given key.
func Sign[S Statement](subject S, key bls.PrivateKey) Signature[S] {
	data := subject.GetDataToSign()
	return Signature[S]{Signature: key.Sign(data)}
}

// Verify checks if the signature is valid for the given key and statement.
func (s Signature[S]) Verify(key bls.PublicKey, statement S) bool {
	data := statement.GetDataToSign()
	return s.Signature.Verify(key, data)
}
