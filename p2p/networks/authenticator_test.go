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
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestVerifyBindingProof_ValidProof_Accepts(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()
	nonce := newNonce(t)

	proof, err := CreateBindingProof(signer, self, 7, 42, nonce)
	require.NoError(t, err, "CreateBindingProof failed")
	require.NoError(t, VerifyBindingProof(verifier, proof, self, 42, memberOf(publicKey)), "expected valid proof to verify")
}

func TestVerifyBindingProof_ReplayedOntoOtherPeer_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	attacker := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	require.NoError(t, err, "CreateBindingProof failed")
	// The attacker presents the victim's proof on its own connection.
	err = VerifyBindingProof(verifier, proof, attacker, 42, memberOf(publicKey))
	require.ErrorIs(t, err, ErrHandshakePeerMismatch)
}

func TestVerifyBindingProof_WrongEpoch_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	require.NoError(t, err, "CreateBindingProof failed")
	require.ErrorIs(t, VerifyBindingProof(verifier, proof, self, 43, memberOf(publicKey)), ErrHandshakeWrongEpoch)
}

func TestVerifyBindingProof_NotInValidatorSet_Rejected(t *testing.T) {
	signer, _ := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	require.NoError(t, err, "CreateBindingProof failed")
	rejectAll := func([]byte) bool { return false }
	require.ErrorIs(t, VerifyBindingProof(verifier, proof, self, 42, rejectAll), ErrHandshakeNotValidator)
}

func TestVerifyBindingProof_TamperedSignature_Rejected(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	self := newTestPeerID(t)
	verifier := NewSecp256k1Verifier()

	proof, err := CreateBindingProof(signer, self, 7, 42, newNonce(t))
	require.NoError(t, err, "CreateBindingProof failed")
	proof.Signature[0] ^= 0xff
	require.ErrorIs(t, VerifyBindingProof(verifier, proof, self, 42, memberOf(publicKey)), ErrHandshakeBadSignature)
}

// --- test helpers ---

func newTestSigner(t *testing.T) (Signer, []byte) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err, "failed to generate key")
	signer := NewSecp256k1Signer(key)
	return signer, signer.PublicKey()
}

func newTestPeerID(t *testing.T) peer.ID {
	t.Helper()
	_, public, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err, "failed to generate peer key")
	id, err := peer.IDFromPublicKey(public)
	require.NoError(t, err, "failed to derive peer ID")
	return id
}

func newNonce(t *testing.T) []byte {
	t.Helper()
	nonce := make([]byte, 16)
	_, err := rand.Read(nonce)
	require.NoError(t, err, "failed to read nonce")
	return nonce
}

func memberOf(publicKey []byte) func([]byte) bool {
	return func(candidate []byte) bool {
		return bytes.Equal(candidate, publicKey)
	}
}
