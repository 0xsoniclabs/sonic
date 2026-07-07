// Copyright 2026 Sonic Operations Ltd
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

package networks

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// Errors returned when verifying a validator binding proof.
var (
	ErrHandshakePeerMismatch = errors.New("networks: handshake peer ID does not match connection")
	ErrHandshakeWrongEpoch   = errors.New("networks: handshake epoch does not match current epoch")
	ErrHandshakeNotValidator = errors.New("networks: handshake public key is not in the validator set")
	ErrHandshakeBadSignature = errors.New("networks: handshake signature is invalid")
)

// Signer signs a digest with the local validator consensus key. It mirrors the
// subset of valkeystore.SignerAuthority the mesh needs, using raw bytes to keep
// the package decoupled from the keystore.
type Signer interface {
	// Sign returns a 64-byte R||S signature over the 32-byte digest.
	Sign(digest []byte) ([]byte, error)
	// PublicKey returns the validator's secp256k1 public key bytes.
	PublicKey() []byte
}

// Verifier verifies a signature produced by a Signer.
type Verifier interface {
	// Verify reports whether signature is a valid secp256k1 signature over
	// digest for publicKey.
	Verify(publicKey, digest, signature []byte) bool
}

// CreateBindingProof builds a validator handshake proving that the local
// validator (identity validatorID / publicKey via signer) operates the libp2p
// peer self, for the given epoch. The signature covers the peer ID, epoch, and
// nonce, so the proof cannot be replayed onto a different peer or epoch.
func CreateBindingProof(signer Signer, self peer.ID, validatorID uint32, epoch uint64, nonce []byte) (*pb.ValidatorHandshake, error) {
	digest := bindingDigest(self, epoch, nonce)
	signature, err := signer.Sign(digest[:])
	if err != nil {
		return nil, err
	}
	return &pb.ValidatorHandshake{
		ValidatorPublicKey: signer.PublicKey(),
		ValidatorId:        validatorID,
		Epoch:              epoch,
		PeerId:             []byte(self),
		Nonce:              nonce,
		Signature:          signature,
	}, nil
}

// VerifyBindingProof checks a received handshake against the expected remote
// peer and current epoch. isValidator reports whether the presented public key
// belongs to the current validator set. It defends against replay by requiring
// the proof to bind to the actual connection's remote peer and the live epoch.
func VerifyBindingProof(
	verifier Verifier,
	proof *pb.ValidatorHandshake,
	expectedPeer peer.ID,
	epoch uint64,
	isValidator func(publicKey []byte) bool,
) error {
	if string(proof.PeerId) != string(expectedPeer) {
		return ErrHandshakePeerMismatch
	}
	if proof.Epoch != epoch {
		return ErrHandshakeWrongEpoch
	}
	if !isValidator(proof.ValidatorPublicKey) {
		return ErrHandshakeNotValidator
	}
	digest := bindingDigest(expectedPeer, epoch, proof.Nonce)
	if !verifier.Verify(proof.ValidatorPublicKey, digest[:], proof.Signature) {
		return ErrHandshakeBadSignature
	}
	return nil
}

// handshakeDomain domain-separates the mesh handshake signature from every other
// use of the consensus key (e.g. validator-directory advertisements), so a
// signature produced in one context can never be valid in another.
const handshakeDomain = "sonic/p2p/validator-handshake/v1\x00"

// bindingDigest computes the 32-byte digest signed by a binding proof over the
// domain tag, peer ID, epoch, and nonce.
func bindingDigest(self peer.ID, epoch uint64, nonce []byte) [32]byte {
	hasher := sha256.New()
	hasher.Write([]byte(handshakeDomain))
	hasher.Write([]byte(self))
	var epochBytes [8]byte
	binary.BigEndian.PutUint64(epochBytes[:], epoch)
	hasher.Write(epochBytes[:])
	hasher.Write(nonce)
	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))
	return digest
}

// --- default secp256k1 implementations ---

// secp256k1Signer is a Signer backed by an ECDSA private key. It is primarily a
// convenience for tests and the future keystore wiring.
type secp256k1Signer struct {
	key *ecdsa.PrivateKey
}

// NewSecp256k1Signer returns a Signer backed by the given secp256k1 key.
func NewSecp256k1Signer(key *ecdsa.PrivateKey) Signer {
	return &secp256k1Signer{key: key}
}

func (s *secp256k1Signer) Sign(digest []byte) ([]byte, error) {
	signature, err := crypto.Sign(digest, s.key)
	if err != nil {
		return nil, err
	}
	return signature[:64], nil
}

func (s *secp256k1Signer) PublicKey() []byte {
	return crypto.CompressPubkey(&s.key.PublicKey)
}

// secp256k1Verifier verifies secp256k1 signatures using go-ethereum crypto.
type secp256k1Verifier struct{}

// NewSecp256k1Verifier returns the default secp256k1 signature verifier.
func NewSecp256k1Verifier() Verifier {
	return secp256k1Verifier{}
}

func (secp256k1Verifier) Verify(publicKey, digest, signature []byte) bool {
	return crypto.VerifySignature(publicKey, digest, signature)
}
