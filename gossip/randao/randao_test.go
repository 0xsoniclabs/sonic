package randao_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
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

func TestPrevRandAO_CanBeProduceAPrevRandAOFromVerifiableSource(t *testing.T) {
	previous := common.Hash{}
	replayProtection := big.NewInt(0)

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	source, err := randao.NewRandaoSource(previous, replayProtection, publicKey, signer)
	require.NoError(t, err)

	_, ok := source.GetRandAo(previous, replayProtection, publicKey)
	require.True(t, ok)
}

func TestRandAO_NewPrevRandAo_FailsWithInvalidKey(t *testing.T) {

	previous := common.Hash{}
	replayProtection := big.NewInt(0)

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)

	_, err := randao.NewRandaoSource(previous, replayProtection, validatorpk.PubKey{}, signer)
	require.ErrorContains(t, err, "not supported key type")
}

func TestPrevRandAO_VerificationDependsOnKnownPublicValues(t *testing.T) {
	previous := common.Hash{}
	replayProtection := big.NewInt(0)

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	_, differentPublicKey := generateKeyPair(t)

	source, err := randao.NewRandaoSource(previous, replayProtection, publicKey, signer)
	require.NoError(t, err)

	tests := map[string]struct {
		previous          common.Hash
		proposerPublicKey validatorpk.PubKey
	}{
		"different previous prevRandAo": {
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
			_, ok := source.GetRandAo(test.previous, replayProtection, test.proposerPublicKey)
			require.False(t, ok)
		})
	}
}

func TestPrevRandAO_GetRandAo_InvalidSourceShallFailVerification(t *testing.T) {
	previous := common.Hash{}
	replayProtection := big.NewInt(0)

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	signer := valkeystore.NewSigner(mockBackend)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil)

	source, err := randao.NewRandaoSource(previous, replayProtection, publicKey, signer)
	require.NoError(t, err)

	for i := range len(source) {
		// modify the signature somehow
		modifiedSignature := randao.RandaoSource(make([]byte, len(source)))
		copy(modifiedSignature[:], source[:])
		modifiedSignature[i] = modifiedSignature[i] + 1

		_, ok := modifiedSignature.GetRandAo(previous, big.NewInt(0), publicKey)
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
