package valkeystore

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestSignerAuthority_PublicKeyCanBeRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)

	_, publicKey := generateKeyPair(t)
	signer := NewSignerAuthority(store, publicKey)

	retrievedPublicKey := signer.PublicKey()
	require.Equal(t, publicKey, retrievedPublicKey)
}

func TestSignerAuthority_CanSign(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)

	privateKey, publicKey := generateKeyPair(t)
	signer := NewSignerAuthority(store, publicKey)
	store.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	var digest common.Hash
	copy(digest[:], []byte("digest"))

	signed, err := signer.Sign(digest)
	require.NoError(t, err)

	require.True(t, VerifySignature(digest, signed, publicKey))
}

func TestSignerAuthority_SignatureMayFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)
	_, publicKey := generateKeyPair(t)

	t.Run("with invalid key format", func(t *testing.T) {
		testKey := publicKey.Copy()
		testKey.Type = validatorpk.Types.Secp256k1 + 1

		signer := NewSignerAuthority(store, testKey)

		var digest common.Hash
		copy(digest[:], []byte("digest"))

		_, err := signer.Sign(digest)
		require.Error(t, err)
	})

	t.Run("with corrupted key", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		key.D = big.NewInt(0) // corrupt the key by setting D to zero
		corruptedKey := encryption.PrivateKey{Decoded: key}

		signer := NewSignerAuthority(store, publicKey)
		store.EXPECT().GetUnlocked(publicKey).Return(&corruptedKey, nil)

		var digest common.Hash
		copy(digest[:], []byte("digest"))

		_, err = signer.Sign(digest)
		require.Error(t, err)
	})

	t.Run("with missing key", func(t *testing.T) {
		signer := NewSignerAuthority(store, publicKey)
		store.EXPECT().GetUnlocked(publicKey).Return(nil, fmt.Errorf("key not found"))

		var digest common.Hash
		copy(digest[:], []byte("digest"))

		_, err := signer.Sign(digest)
		require.Error(t, err)
	})
}

func TestSignerAuthority_VerificationFailsWithDifferentDigest(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)

	privateKey, publicKey := generateKeyPair(t)
	signer := NewSignerAuthority(store, publicKey)
	store.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	var digest common.Hash
	copy(digest[:], []byte("digest"))

	signed, err := signer.Sign(digest)
	require.NoError(t, err)

	// Modify the digest for verification
	var differentDigest common.Hash
	copy(differentDigest[:], []byte("different_digest"))

	valid := VerifySignature(differentDigest, signed, publicKey)
	require.False(t, valid)
}

func TestSignerAuthority_VerificationFailsWithDifferentKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)

	privateKey, publicKey := generateKeyPair(t)
	signer := NewSignerAuthority(store, publicKey)
	store.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	var digest common.Hash
	copy(digest[:], []byte("digest"))

	signed, err := signer.Sign(digest)
	require.NoError(t, err)

	// Generate a different key pair for verification
	_, differentPublicKey := generateKeyPair(t)

	// Verify with a different public key
	valid := VerifySignature(digest, signed, differentPublicKey)
	require.False(t, valid)
}

func TestSignerAuthority_VerificationFailsWithInvalidKeType(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockKeystoreI(ctrl)

	privateKey, publicKey := generateKeyPair(t)
	signer := NewSignerAuthority(store, publicKey)
	store.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	var digest common.Hash
	copy(digest[:], []byte("digest"))

	signed, err := signer.Sign(digest)
	require.NoError(t, err)

	// Create a public key with an unsupported type
	invalidPublicKey := publicKey.Copy()
	invalidPublicKey.Type = validatorpk.Types.Secp256k1 + 1

	// Verify with the invalid public key type
	valid := VerifySignature(digest, signed, invalidPublicKey)
	require.False(t, valid)
}

func generateKeyPair(t testing.TB) (*encryption.PrivateKey, validatorpk.PubKey) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	publicKey := validatorpk.PubKey{
		Raw:  crypto.FromECDSAPub(&privateKeyECDSA.PublicKey),
		Type: validatorpk.Types.Secp256k1,
	}
	privateKey := &encryption.PrivateKey{
		Type:    validatorpk.Types.Secp256k1,
		Decoded: privateKeyECDSA,
	}

	return privateKey, publicKey
}
