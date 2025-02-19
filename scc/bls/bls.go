// This package provides convenience wrappers around the blst library to
// generate BLS12-381 keys, sign messages, and verify signatures. The package
// is designed to be simple to use and to provide a high-level API to work with
// BLS12-381 keys and signatures.
//
// The blst (pronounced "Blast") library is a high-performance BLS12-381 library
// that provides a low-level API to work with BLS12-381 keys and signatures. The
// library is written in C and provides Go bindings to use the library from Go
// code. The blst library is designed to be fast and secure, and is suitable for
// production use.
//
// When compiling the Go bindings of the blst library, native C code is
// implicitly compiled using Go's CGO tool. On most systems, this should work
// transparent. However, on older CPU architectures, certain optimizations may
// cause the compiled code to crash with a SIGILL signal. This is a known issue
// with the blst library, as reported here:
//
// Issue https://github.com/bnb-chain/bsc/issues/1521
//
// If you encounter an error like the this
//
//	> Caught SIGILL in blst_cgo_init, consult <blst>/bindinds/go/README.md
//
// you can work around the issue by setting the CGO_CFLAGS environment variable
// to disable the optimizations that cause the crash.
//
//	> export CGO_CFLAGS="-O -D__BLST_PORTABLE__"
//	> export CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__"
//
// Example uses of the API are demonstrated in the unit tests of the package.
package bls

import (
	"crypto/rand"
	"fmt"

	blst "github.com/supranational/blst/bindings/go"
)

// PrivateKey represents a BLS12-381 private key.
type PrivateKey struct {
	secretKey blst.SecretKey
}

// PublicKey represents a BLS12-381 public key.
type PublicKey struct {
	publicKey blst.P1Affine
}

// Signature represents a BLS12-381 signature.
type Signature struct {
	sign blst.P2Affine
}

// --- Private Keys -----------------------------------------------------------

// NewPrivateKey creates a new BLS12-381 private key. The resulting keys are
// cryptographically secure and can be used for production purposes.
func NewPrivateKey() PrivateKey {
	var inputKeyMaterial [32]byte
	// crypto/rand.Read is cryptographically secure, guaranteed to never fail
	_, _ = rand.Read(inputKeyMaterial[:])
	return newPrivateKey(inputKeyMaterial)
}

// NewPrivateKeyForTests creates a new BLS12-381 private key using the provided
// key material. The first 32 bytes of the material is used, the rest is ignored.
// The security of the resulting key depends on the quality of the key material.
// This function may be used in tests to generate deterministic keys. Use
// NewPrivateKey() for production purposes.
func NewPrivateKeyForTests(keyMaterial ...byte) PrivateKey {
	var material [32]byte
	copy(material[:], keyMaterial)
	return newPrivateKey(material)
}

// newPrivateKey is a helper function that creates a new private key using the
// provided key material. The security of the resulting key depends on the
// quality of the key material.
func newPrivateKey(keyMaterial [32]byte) PrivateKey {
	res := PrivateKey{}
	res.secretKey = *blst.KeyGen(keyMaterial[:])
	return res
}

// PublicKey returns the public key corresponding to the private key.
func (k PrivateKey) PublicKey() PublicKey {
	res := PublicKey{}
	res.publicKey = *res.publicKey.From(&k.secretKey)
	return res
}

// Sign signs the provided message using the private key and returns the
// resulting signature.
func (k PrivateKey) Sign(message []byte) Signature {
	res := Signature{}
	res.sign.Sign(&k.secretKey, message, nil)
	return res
}

// GetProofOfPossession returns a proof of possession for the private key. The
// proof of possession can be used to prove that the signer knew the private key
// corresponding to the public key at some point in time.
func (k PrivateKey) GetProofOfPossession() Signature {
	msg := k.PublicKey().Serialize()
	return k.Sign(msg[:])
}

// Serialize exports the private key into a 32-byte array. This format can be
// used to serialize the key to disk or to transmit it over the network.
func (k PrivateKey) Serialize() [32]byte {
	return [32]byte(k.secretKey.Serialize())
}

// DeserializePrivateKey deserializes a private key from the provided serialized
// representation.If the provided serialized representation is invalid, an error
// is returned.
func DeserializePrivateKey(serialized [32]byte) (PrivateKey, error) {
	res := PrivateKey{}
	if key := res.secretKey.Deserialize(serialized[:]); key == nil {
		return PrivateKey{}, fmt.Errorf("failed to deserialize private key")
	}
	return res, nil
}

// String returns the private key as a hexadecimal string prefixed with "0x".
func (k PrivateKey) String() string {
	return fmt.Sprintf("0x%x", k.Serialize())
}

// --- Public Keys ------------------------------------------------------------

// Validate returns true if the public key is valid, false otherwise. A public
// key is considered valid if it is on the BLS12-381 curve. Invalid public keys
// must not be used in cryptographic operations.
func (k PublicKey) Validate() bool {
	return k.publicKey.KeyValidate()
}

// CheckProofOfPossession returns true if the provided signature is a valid proof
// of possession for the public key, false otherwise. A valid proof of possession
// guarantees that the signer knew at some point in time the private key
// corresponding to the public key.
func (k PublicKey) CheckProofOfPossession(signature Signature) bool {
	msg := k.Serialize()
	return signature.Verify(k, msg[:])
}

// AggregatePublicKeys aggregates the provided keys into a single key. The
// provided keys must be valid. Caller must ensure that the provided keys are
// valid, otherwise the resulting key may be invalid.
func AggregatePublicKeys(keys ...PublicKey) PublicKey {
	agg := blst.P1Aggregate{}
	for _, key := range keys {
		agg.Add(&key.publicKey, false)
	}
	return PublicKey{publicKey: *agg.ToAffine()}
}

// DeserializePublicKey deserializes a public key from the provided serialized
// representation. If the provided serialized representation is invalid, an
// error is returned.
func DeserializePublicKey(serialized [48]byte) (PublicKey, error) {
	res := PublicKey{}
	if key := res.publicKey.Uncompress(serialized[:]); key == nil {
		return PublicKey{}, fmt.Errorf("failed to deserialize public key")
	}
	return res, nil
}

// Serialize exports the public key into a 48-byte array. This format can be
// used to serialize the key to disk or to transmit it over the network.
func (k PublicKey) Serialize() [48]byte {
	return [48]byte(k.publicKey.Compress())
}

// String returns the public key as a hexadecimal string prefixed with "0x".
func (k PublicKey) String() string {
	return fmt.Sprintf("0x%x", k.Serialize())
}

// --- Signatures -------------------------------------------------------------

// Validate returns true if the signature is valid, false otherwise. A signature
// is considered valid if it is on the BLS12-381 curve. Invalid signatures must
// not be used in cryptographic operations.
func (s Signature) Validate() bool {
	return s.sign.SigValidate(true)
}

// Verify returns true if the signature is valid for the provided message and
// public key, false otherwise. A signature is considered valid if it was
// produced by the corresponding private key and the message has not been
// tampered with.
func (s Signature) Verify(publicKey PublicKey, message []byte) bool {
	return s.sign.Verify(true, &publicKey.publicKey, true, message, nil)
}

// VerifyAll returns true if the signature is the aggregation of all signature
// produced by the owners of the all the given public keys for the given
// message, false otherwise. Using this function is more efficient than
// aggregating the public keys first and then verifying the signature against
// the aggregated public key.
func (s Signature) VerifyAll(publicKeys []PublicKey, message []byte) bool {
	keys := make([]*blst.P1Affine, len(publicKeys))
	msgs := make([][]byte, len(publicKeys))
	for i, key := range publicKeys {
		keys[i] = &key.publicKey
		msgs[i] = message
	}
	return s.sign.AggregateVerify(false, keys, false, msgs, nil)
}

// AggregateSignatures aggregates the provided signatures into a single
// signature. The provided signatures must be valid.
func AggregateSignatures(signatures ...Signature) Signature {
	agg := blst.P2Aggregate{}
	for _, sigs := range signatures {
		agg.Add(&sigs.sign, false)
	}
	return Signature{sign: *agg.ToAffine()}
}

// DeserializeSignature deserializes a signature from the provided serialized
// representation. If the provided serialized representation is invalid, an
// error is returned.
func DeserializeSignature(serialized [96]byte) (Signature, error) {
	res := Signature{}
	if key := res.sign.Uncompress(serialized[:]); key == nil {
		return Signature{}, fmt.Errorf("failed to deserialize signature")
	}
	return res, nil
}

// Serialize exports the signature into a 96-byte array. This format can be used
// to serialize the signature to disk or to transmit it over the network.
func (k Signature) Serialize() [96]byte {
	return [96]byte(k.sign.Compress())
}

// String returns the signature as a hexadecimal string prefixed with "0x".
func (k Signature) String() string {
	return fmt.Sprintf("0x%x", k.Serialize())
}
