package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/tests/contracts/blsContracts"
	gnark "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBlsVerificationOnChain(t *testing.T) {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		FeatureSet: opera.AllegroFeatures,
	})
	defer net.Stop()

	// Deploy contract with transaction options
	blsContract, _, err := DeployContract(net, blsContracts.DeployBLS)
	require.NoError(t, err, "failed to deploy contract; %v", err)

	// Test different bls libraries verification
	tests := []struct {
		name     string
		dataFunc func(pk bls.PrivateKey, message []byte) ([]byte, []byte, error)
	}{
		{"gnark", getGnarkData},
		{"bls", getBlsData},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runTest(t, net, blsContract, test.dataFunc)
		})
	}

	t.Run("aggregate", func(t *testing.T) {
		// Get test data
		pubKeys, signature, msg := getBlsAggregateData()

		tests := []struct {
			name      string
			pubkeys   []bls.PublicKey
			signature bls.Signature
			message   []byte
			ok        bool
		}{
			{"ok", pubKeys, signature, msg, true},
			{"message not ok", pubKeys, signature, []byte("message not ok"), false},
			{"public key not ok", []bls.PublicKey{bls.NewPrivateKey().PublicKey()}, signature, msg, false},
			{"signature not ok", pubKeys, bls.NewPrivateKey().Sign([]byte("some message")), msg, false},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {

				pubKeys, signature, msg, err := parseInputData(test.pubkeys, test.signature, test.message)
				require.NoError(t, err, "failed to parse test data; %v", err)

				ok, err := blsContract.CheckAgregatedSignature(nil, pubKeys, signature, msg)
				require.NoError(t, err, "failed to call CheckSignature; %v", err)
				require.Equal(t, test.ok, ok, "result has to be has to be %v", test.ok)
			})
		}

		t.Run("update signature", func(t *testing.T) {
			pubKeysData, sig, message, err := parseInputData(pubKeys, signature, msg)
			require.NoError(t, err, "failed to parse test data; %v", err)

			receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
				return blsContract.CheckAndUpdateAgregatedSignature(ops, pubKeysData, sig, message)
			})
			require.NoError(t, err, "failed to get receipt; %v", err)
			t.Logf("gas used for updating signature: %v", receipt.GasUsed)

			updatedSignature, err := blsContract.Signature(nil)
			require.NoError(t, err, "failed to get updated signature; %v", err)
			require.Equal(t, sig, updatedSignature, "signature has to be updated")
		})
	})
}

func runTest(
	t *testing.T,
	net *IntegrationTestNet,
	blsContract *blsContracts.BLS,
	f func(pk bls.PrivateKey, message []byte) ([]byte, []byte, error),
) {

	pubKey, signature, message, err := getDataForVerification(f)
	require.NoError(t, err, "failed to get test data; %v", err)

	checkOk, err := blsContract.CheckSignature(nil, pubKey, signature, message)
	require.NoError(t, err, "failed to call CheckSignature; %v", err)
	require.True(t, checkOk, "signature has to be valid")

	checkNotOk, err := blsContract.CheckSignature(nil, pubKey, signature, []byte("hello world"))
	require.NoError(t, err, "failed to call CheckSignature; %v", err)
	require.False(t, checkNotOk, "signature has to be invalid")

	txOpts, err := net.GetTransactOptions(&net.account)
	require.NoError(t, err, "failed to get transaction options; %v", err)

	updateTransaction, err := blsContract.CheckAndUpdate(txOpts, pubKey, signature, message)
	require.NoError(t, err, "failed to update contract signature; %v", err)

	receipt, err := net.GetReceipt(updateTransaction.Hash())
	require.NoError(t, err, "failed to get receipt; %v", err)
	t.Logf("gas used for updating signature: %v", receipt.GasUsed)
}

// getDataForVerification returns data for verification
// pubKey, signature, message, error
func getDataForVerification(
	f func(pk bls.PrivateKey, message []byte) ([]byte, []byte, error),
) ([]byte, []byte, []byte, error) {

	message := []byte("hello")
	pk := bls.NewPrivateKeyForTests(1)

	pubKey, signature, err := f(pk, message)
	if err != nil {
		return nil, nil, nil, err
	}
	return pubKey, signature, message, err
}

func getGnarkData(pk bls.PrivateKey, message []byte) ([]byte, []byte, error) {
	// hash message and get G2 point
	msg, err := gnark.EncodeToG2(message, nil)
	if err != nil {
		return nil, nil, err
	}

	// private key as scalar
	pkBytes := pk.Serialize()
	pkScalar := big.NewInt(0).SetBytes(pkBytes[:])

	// calculate signature as pk * msg, G2 points
	sig := new(gnark.G2Affine).ScalarMultiplication(&msg, pkScalar)
	signature := encodePointG2(sig)

	// calculate public key as pk * G1, G1 points
	_, _, g1, _ := gnark.Generators()
	pubKeyG1 := new(gnark.G1Affine).ScalarMultiplication(&g1, pkScalar)
	pubKey := encodePointG1(pubKeyG1)

	return pubKey, signature, nil
}

func getBlsData(pk bls.PrivateKey, message []byte) ([]byte, []byte, error) {

	// convert bls public key to G2 point bytes
	pubKeyBls := pk.PublicKey()
	pubKeyG1, err := publicKeyToGnarkG1Affine(pubKeyBls)
	if err != nil {
		return nil, nil, err
	}
	pubKey := encodePointG1(&pubKeyG1)

	// convert bls signature to G2 point bytes
	sigBlsLib := pk.Sign(message)
	p, err := signatureToGnarkG2Affine(sigBlsLib)
	if err != nil {
		return nil, nil, err
	}
	signature := encodePointG2(&p)
	return pubKey, signature, nil
}

func publicKeyToGnarkG1Affine(key bls.PublicKey) (gnark.G1Affine, error) {
	data := key.Serialize()
	var res gnark.G1Affine
	_, err := res.SetBytes(data[:])
	if err != nil {
		return gnark.G1Affine{}, err
	}
	return res, nil
}

func signatureToGnarkG2Affine(sig bls.Signature) (gnark.G2Affine, error) {
	data := sig.Serialize()
	var res gnark.G2Affine
	_, err := res.SetBytes(data[:])
	if err != nil {
		return gnark.G2Affine{}, err
	}
	return res, nil
}

// encodePointG1 encodes a point into 128 bytes.
func encodePointG1(p *gnark.G1Affine) []byte {
	out := make([]byte, 128)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:]), p.X)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[64+16:]), p.Y)
	return out
}

// encodePointG2 encodes a point into 256 bytes.
func encodePointG2(p *gnark.G2Affine) []byte {
	out := make([]byte, 256)
	// encode x
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:16+48]), p.X.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[80:80+48]), p.X.A1)
	// encode y
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[144:144+48]), p.Y.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[208:208+48]), p.Y.A1)
	return out
}

func getBlsAggregateData() ([]bls.PublicKey, bls.Signature, []byte) {
	msg := []byte("Test message")
	pk1 := bls.NewPrivateKey()
	pk2 := bls.NewPrivateKey()
	pk3 := bls.NewPrivateKey()
	sig1 := pk1.Sign(msg)
	sig2 := pk2.Sign(msg)
	sig3 := pk3.Sign(msg)

	pubKeys := []bls.PublicKey{pk1.PublicKey(), pk2.PublicKey(), pk3.PublicKey()}
	sigAggregate := bls.AggregateSignatures(sig1, sig2, sig3)

	return pubKeys, sigAggregate, msg
}

func parseInputData(pubKeys []bls.PublicKey, signature bls.Signature, msg []byte) ([]byte, []byte, []byte, error) {
	pubKeysData := make([]byte, 0, len(pubKeys)*128)
	for _, pk := range pubKeys {
		pubG1, err := publicKeyToGnarkG1Affine(pk)
		if err != nil {
			return nil, nil, nil, err
		}
		pubKeysData = append(pubKeysData, encodePointG1(&pubG1)...)
	}
	signatureG2, err := signatureToGnarkG2Affine(signature)
	if err != nil {
		return nil, nil, nil, err
	}
	signatureData := encodePointG2(&signatureG2)
	return pubKeysData, signatureData, msg, nil
}
