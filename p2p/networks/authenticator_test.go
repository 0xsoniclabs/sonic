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
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestVerifyBindingProof_ValidProof_Accepts(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()
	nonce := newNonce(t)

	proof, err := CreateBindingProof(signer, self, 7, 42, nonce)
	if err != nil {
		t.Fatalf("CreateBindingProof failed: %v", err)
	}
	if err := VerifyBindingProof(verifier, proof, self, 42, memberOf(publicKey)); err != nil {
		t.Fatalf("expected valid proof to verify, got %v", err)
	}
}

func TestVerifyBindingProof_ReplayedOntoOtherPeer_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	attacker := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	if err != nil {
		t.Fatalf("CreateBindingProof failed: %v", err)
	}
	// The attacker presents the victim's proof on its own connection.
	err = VerifyBindingProof(verifier, proof, attacker, 42, memberOf(publicKey))
	if !errors.Is(err, ErrHandshakePeerMismatch) {
		t.Fatalf("expected ErrHandshakePeerMismatch, got %v", err)
	}
}

func TestVerifyBindingProof_WrongEpoch_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	if err != nil {
		t.Fatalf("CreateBindingProof failed: %v", err)
	}
	if err := VerifyBindingProof(verifier, proof, self, 43, memberOf(publicKey)); !errors.Is(err, ErrHandshakeWrongEpoch) {
		t.Fatalf("expected ErrHandshakeWrongEpoch, got %v", err)
	}
}

func TestVerifyBindingProof_NotInValidatorSet_Rejected(t *testing.T) {
	signer, _ := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	if err != nil {
		t.Fatalf("CreateBindingProof failed: %v", err)
	}
	rejectAll := func([]byte) bool { return false }
	if err := VerifyBindingProof(verifier, proof, self, 42, rejectAll); !errors.Is(err, ErrHandshakeNotValidator) {
		t.Fatalf("expected ErrHandshakeNotValidator, got %v", err)
	}
}

func TestVerifyBindingProof_TamperedSignature_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	if err != nil {
		t.Fatalf("CreateBindingProof failed: %v", err)
	}
	proof.Signature[0] ^= 0xff
	if err := VerifyBindingProof(verifier, proof, self, 42, memberOf(publicKey)); !errors.Is(err, ErrHandshakeBadSignature) {
		t.Fatalf("expected ErrHandshakeBadSignature, got %v", err)
	}
}

// --- test helpers ---

func newTestSigner(t *testing.T) (Signer, []byte) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	signer := NewSecp256k1Signer(key)
	return signer, signer.PublicKey()
}

func newTestPeerID(t *testing.T) peer.ID {
	t.Helper()
	_, public, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate peer key: %v", err)
	}
	id, err := peer.IDFromPublicKey(public)
	if err != nil {
		t.Fatalf("failed to derive peer ID: %v", err)
	}
	return id
}

func newNonce(t *testing.T) []byte {
	t.Helper()
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("failed to read nonce: %v", err)
	}
	return nonce
}

func memberOf(publicKey []byte) func([]byte) bool {
	return func(candidate []byte) bool {
		return bytes.Equal(candidate, publicKey)
	}
}
