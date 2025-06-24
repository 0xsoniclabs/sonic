// Copyright 2025 Sonic Operations Ltd
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

package randao

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRandao_RandaoReveal_CanBeConstructedAndVerified(t *testing.T) {
	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)
	signerAuth := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	source, err := generateNextRandaoReveal(previous, signerAuth)
	require.NoError(t, err)

	_, ok := source.VerifyAndGetRandao(previous, publicKey)
	require.True(t, ok)
}

func TestRandao_NewRandaoReveal_ConstructionFailsWithInvalidKey(t *testing.T) {

	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)

	publicKey := validatorpk.PubKey{}
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	_, err := generateNextRandaoReveal(previous, signer)
	require.ErrorContains(t, err, "not supported key type")
}

func TestRandao_RandaoReveal_VerificationDependsOnKnownPublicValues(t *testing.T) {
	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	_, differentPublicKey := generateKeyPair(t)

	source, err := generateNextRandaoReveal(previous, signer)
	require.NoError(t, err)

	tests := map[string]struct {
		previous          common.Hash
		proposerPublicKey validatorpk.PubKey
	}{
		"different previous prevRandao": {
			previous:          common.Hash{0x01},
			proposerPublicKey: publicKey,
		},
		"different proposerAddress": {
			previous:          common.Hash{},
			proposerPublicKey: differentPublicKey,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, ok := source.VerifyAndGetRandao(test.previous, test.proposerPublicKey)
			require.False(t, ok)
		})
	}
}

func TestRandao_RandaoReveal_InvalidRandaoRevealShallFailVerification(t *testing.T) {
	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	source, err := generateNextRandaoReveal(previous, signer)
	require.NoError(t, err)

	for i := range len(source) {
		// modify the signature somehow
		modifiedSignature := RandaoReveal(make([]byte, len(source)))
		copy(modifiedSignature[:], source[:])
		modifiedSignature[i] = modifiedSignature[i] + 1

		_, ok := modifiedSignature.VerifyAndGetRandao(previous, publicKey)
		require.False(t, ok, "modified signature shall not be valid")
	}
}

// generateKeyPair is a helper function that creates a new ECDSA key pair
// and packs it in the data structures used by the gossip package.
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

func TestRandao_NewRandaoReveal_IsDeterministic(t *testing.T) {

	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	reveals := make([]RandaoReveal, 10)
	for i := range 10 {
		source, err := generateNextRandaoReveal(previous, signer)
		require.NoError(t, err)
		reveals[i] = source
	}

	// check that all randao reveals are equal
	for _, a := range reveals {
		for _, b := range reveals {
			require.Equal(t, a, b)
		}
	}
}
func TestRandao_GetRandao_IsDeterministic(t *testing.T) {

	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	reveals := make([]RandaoReveal, 10)
	for i := range 10 {
		source, err := generateNextRandaoReveal(previous, signer)
		require.NoError(t, err)
		reveals[i] = source
	}

	randaoValues := make([]common.Hash, 10)
	for i := range 10 {
		randaoValue, ok := reveals[i].VerifyAndGetRandao(previous, publicKey)
		require.True(t, ok)
		randaoValues[i] = randaoValue
	}

	// check that all randao values are equal
	for _, a := range randaoValues {
		for _, b := range randaoValues {
			require.Equal(t, a, b)
		}
	}
}

func TestRandaoReveal_EntropyTest(t *testing.T) {
	// The following test measures the generation entropy by creating
	// a large number of randao generations and 10 different keys.

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)

	type keyPair struct {
		privateKey *encryption.PrivateKey
		publicKey  validatorpk.PubKey
	}
	keys := make([]keyPair, 10)

	for i := range len(keys) {
		privateKey, publicKey := generateKeyPair(t)

		keys[i] = keyPair{
			privateKey: privateKey,
			publicKey:  publicKey,
		}
	}
	// The storage of keys would be easier using a map from public to private keys,
	// but because the gossip interfaces use pubKey by value indexing turns a little more
	// complicated.
	// This test uses a slice of keypairs sorted by public key, and binary search to
	// preserve arguments by value.
	slices.SortFunc(keys, func(a, b keyPair) int {
		return strings.Compare(a.publicKey.String(), b.publicKey.String())
	})
	mockBackend.EXPECT().GetUnlocked(gomock.Any()).DoAndReturn(func(key validatorpk.PubKey) (*encryption.PrivateKey, error) {
		idx, ok := slices.BinarySearchFunc(keys, key, func(element keyPair, key validatorpk.PubKey) int {
			return strings.Compare(element.publicKey.String(), key.String())
		})
		if ok {
			return keys[idx].privateKey, nil
		}
		return nil, fmt.Errorf("key not found, malformed test")
	}).AnyTimes()

	byteStream := make([]byte, 0)
	for range 10 {

		lastRandao := common.Hash{}
		_, err := rand.Read(lastRandao[:])
		require.NoError(t, err)
		for i := range 10_000 {

			keyIdx := i % 10
			publicKey := keys[keyIdx].publicKey
			signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

			source, err := generateNextRandaoReveal(lastRandao, signer)
			require.NoError(t, err)
			randao, ok := source.VerifyAndGetRandao(lastRandao, publicKey)
			require.True(t, ok)
			byteStream = append(byteStream, randao[:]...)
			lastRandao = randao
		}

		entropy := calculate_normalized_shannon_entropy(byteStream)
		require.Greater(t, entropy, 0.9999, "Entropy should be greater than 0.9999")
	}
}

func TestRandaoMixer_CanProduceAVerifiableRandaoReveal(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	randaoMixer := NewRandaoMixerAdapter(signer)

	previousRandao := common.Hash{}
	reveal, randao, err := randaoMixer.MixRandao(previousRandao)
	require.NoError(t, err)

	// Verify the reveal
	verifiedRandao, ok := reveal.VerifyAndGetRandao(previousRandao, publicKey)
	require.True(t, ok)
	require.Equal(t, randao, verifiedRandao)
}

func TestRandaoMixer_ReturnsErrorIfRandaoGenerationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	signer := valkeystore.NewMockSignerAuthority(ctrl)

	signer.EXPECT().Sign(gomock.Any()).Return(nil, fmt.Errorf("nop"))
	randaoMixer := NewRandaoMixerAdapter(signer)
	previousRandao := common.Hash{}

	_, _, err := randaoMixer.MixRandao(previousRandao)
	require.ErrorContains(t, err, "failed to generate next randao reveal")
}

func TestRandaoMixer_ReturnsErrorIfRandaoVerificationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	signer := valkeystore.NewMockSignerAuthority(ctrl)

	signer.EXPECT().Sign(gomock.Any()).Return(bytes.Repeat([]byte{0x42}, 64), nil)
	signer.EXPECT().PublicKey()
	randaoMixer := NewRandaoMixerAdapter(signer)
	previousRandao := common.Hash{}

	_, _, err := randaoMixer.MixRandao(previousRandao)
	require.ErrorContains(t, err, "failed to generate next randao reveal, randao reveal verification failed")
}

func calculate_normalized_shannon_entropy(data []byte) float64 {
	// Create a map to store the frequency of each byte value
	frequency := make(map[byte]int)
	for _, b := range data {
		frequency[b]++
	}

	// Calculate the total number of bytes
	total := len(data)

	// Calculate the Shannon entropy
	entropy := 0.0
	for _, count := range frequency {
		probability := float64(count) / float64(total)
		entropy -= probability * math.Log2(probability)
	}

	// normalize the result
	return entropy / 8 // log2(256) = 8
}
