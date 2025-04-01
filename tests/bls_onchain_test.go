package tests

import (
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

	testVariants := []struct {
		name         string
		signersCount int
		checkFunc    func(opts *bind.CallOpts, pubKey []byte, signature []byte, message []byte) (bool, error)
		updateFunc   func(opts *bind.TransactOpts, pubKeys []byte, signature []byte, message []byte) (*types.Transaction, error)
	}{
		{"verify single", 1, blsContract.CheckSignature, blsContract.CheckAndUpdate},
		{"verify aggregate", 25, blsContract.CheckAggregatedSignature, blsContract.CheckAndUpdateAggregatedSignature},
	}

	for _, testVariant := range testVariants {
		t.Run(testVariant.name, func(t *testing.T) {

			pubKeys, signature, msg := getBlsData(testVariant.signersCount)
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
					pubKeysBytes, signatureBytes, msgBytes, err := parseInputData(test.pubkeys, test.signature, test.message)
					require.NoError(t, err, "failed to parse test data; %v", err)

					ok, err := testVariant.checkFunc(nil, pubKeysBytes, signatureBytes, msgBytes)
					require.NoError(t, err, "failed to check signature; %v", err)
					require.Equal(t, test.ok, ok, "signature has to be %v", test.ok)
				})
			}

			t.Run("update signature", func(t *testing.T) {
				pubKeysBytes, signatureBytes, msgBytes, err := parseInputData(pubKeys, signature, msg)
				require.NoError(t, err, "failed to parse test data; %v", err)

				receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
					ops.GasLimit = 10000000
					return testVariant.updateFunc(ops, pubKeysBytes, signatureBytes, msgBytes)
				})
				require.NoError(t, err, "failed to get receipt; %v", err)
				t.Logf("gas used for updating signature: %v", receipt.GasUsed)

				updatedSignature, err := blsContract.Signature(nil)
				require.NoError(t, err, "failed to get updated signature; %v", err)
				require.Equal(t, signatureBytes, updatedSignature, "signature has to be updated")
			})
		})
	}
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

func getBlsData(signersCount int) ([]bls.PublicKey, bls.Signature, []byte) {
	msg := []byte("Test message")
	pubKeys := make([]bls.PublicKey, signersCount)
	signatures := make([]bls.Signature, signersCount)

	for i := 0; i < signersCount; i++ {
		pk := bls.NewPrivateKey()
		pubKeys[i] = pk.PublicKey()
		signatures[i] = pk.Sign(msg)
	}

	var signature bls.Signature
	if signersCount == 1 {
		signature = signatures[0]
	} else {
		signature = bls.AggregateSignatures(signatures...)
	}

	return pubKeys, signature, msg
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
