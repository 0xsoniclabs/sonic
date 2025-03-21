package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/tests/contracts/blsContracts"
	gnark "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/stretchr/testify/require"
)

func TestBlsOnChain(t *testing.T) {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		FeatureSet: opera.AllegroFeatures,
	})
	defer net.Stop()

	// Deploy contract
	txOptions, err := net.GetTransactOptions(&net.account)
	require.NoError(t, err, "failed to get transact options; %v", err)
	txOptions.Nonce = nil
	txOptions.GasLimit = 10000000

	contract, _, err := DeployContractWithOpts(net, blsContracts.DeployBLS, txOptions)
	require.NoError(t, err, "failed to deploy contract; %v", err)

	tests := []struct {
		name     string
		detaFunc func(pk bls.PrivateKey, message []byte) ([]byte, []byte, error)
	}{
		{"gnark", getGnarkData},
		{"bls", getBlsData},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runTest(t, net, contract, test.detaFunc)
		})
	}
}

func runTest(
	t *testing.T,
	net *IntegrationTestNet,
	blsContract *blsContracts.BLS,
	f func(pk bls.PrivateKey, message []byte) ([]byte, []byte, error),
) {

	notok := make([]byte, 32)
	ok := make([]byte, 32)
	ok[31] = 1

	pubKey, signature, message, err := getDataForVerification(f)
	require.NoError(t, err, "failed to get test data; %v", err)

	checkOk, err := blsContract.CheckSignature(nil, pubKey, signature, message)
	require.NoError(t, err, "failed to call CheckSignature; %v", err)
	require.Equal(t, ok[:], checkOk, "unexpected identity value")

	checkNotOk, err := blsContract.CheckSignature(nil, pubKey, signature, []byte("hello world"))
	require.NoError(t, err, "failed to call CheckSignature; %v", err)
	require.Equal(t, notok[:], checkNotOk, "unexpected identity value")

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
