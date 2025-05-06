package randao_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/randao"
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
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	source, err := randao.NewRandaoReveal(previous, publicKey, signer)
	require.NoError(t, err)

	_, ok := source.GetRandao(previous, publicKey)
	require.True(t, ok)
}

func TestRandao_NewRandaoReveal_ConstructionFailsWithInvalidKey(t *testing.T) {

	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)

	_, err := randao.NewRandaoReveal(previous, validatorpk.PubKey{}, signer)
	require.ErrorContains(t, err, "not supported key type")
}

func TestRandao_RandaoReveal_VerificationDependsOnKnownPublicValues(t *testing.T) {
	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	_, differentPublicKey := generateKeyPair(t)

	source, err := randao.NewRandaoReveal(previous, publicKey, signer)
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
			_, ok := source.GetRandao(test.previous, test.proposerPublicKey)
			require.False(t, ok)
		})
	}
}

func TestRandao_RandaoReveal_InvalidRandaoRevealShallFailVerification(t *testing.T) {
	previous := common.Hash{}

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	source, err := randao.NewRandaoReveal(previous, publicKey, signer)
	require.NoError(t, err)

	for i := range len(source) {
		// modify the signature somehow
		modifiedSignature := randao.RandaoReveal(make([]byte, len(source)))
		copy(modifiedSignature[:], source[:])
		modifiedSignature[i] = modifiedSignature[i] + 1

		_, ok := modifiedSignature.GetRandao(previous, publicKey)
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
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()

	reveals := make([]randao.RandaoReveal, 10)
	for i := range 10 {
		source, err := randao.NewRandaoReveal(previous, publicKey, signer)
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
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()

	reveals := make([]randao.RandaoReveal, 10)
	for i := range 10 {
		source, err := randao.NewRandaoReveal(previous, publicKey, signer)
		require.NoError(t, err)
		reveals[i] = source
	}

	randaoValues := make([]common.Hash, 10)
	for i := range 10 {
		randaoValue, ok := reveals[i].GetRandao(previous, publicKey)
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
	signer := valkeystore.NewSigner(mockBackend)

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

			source, err := randao.NewRandaoReveal(lastRandao, publicKey, signer)
			require.NoError(t, err)
			randao, ok := source.GetRandao(lastRandao, publicKey)
			require.True(t, ok)
			byteStream = append(byteStream, randao[:]...)
			lastRandao = randao
		}

		entropy := calculate_normalized_shannon_entropy(byteStream)
		require.Greater(t, entropy, 0.9999, "Entropy should be greater than 0.9999")
	}
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
