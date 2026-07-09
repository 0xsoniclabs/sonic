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

package tests

import (
	"crypto/rand"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/blsContracts"
	gnark "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"
)

func TestBlsVerificationOnChain(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())
	t.Parallel()

	// Deploy contract with transaction options
	blsContract, _, err := DeployContract(session, blsContracts.DeployBLS)
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
				pubkeys   []blsPublicKey
				signature blsSignature
				message   []byte
				ok        bool
			}{
				{"ok", pubKeys, signature, msg, true},
				{"message not ok", pubKeys, signature, []byte("message not ok"), false},
				{"public key not ok", []blsPublicKey{blsNewPrivateKey().PublicKey()}, signature, msg, false},
				{"signature not ok", pubKeys, blsNewPrivateKey().Sign([]byte("some message")), msg, false},
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

				receipt, err := session.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
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

func publicKeyToGnarkG1Affine(key blsPublicKey) (gnark.G1Affine, error) {
	data := key.Serialize()
	var res gnark.G1Affine
	_, err := res.SetBytes(data[:])
	if err != nil {
		return gnark.G1Affine{}, err
	}
	return res, nil
}

func signatureToGnarkG2Affine(sig blsSignature) (gnark.G2Affine, error) {
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

func getBlsData(signersCount int) ([]blsPublicKey, blsSignature, []byte) {
	msg := []byte("Test message")
	pubKeys := make([]blsPublicKey, signersCount)
	signatures := make([]blsSignature, signersCount)

	for i := 0; i < signersCount; i++ {
		pk := blsNewPrivateKey()
		pubKeys[i] = pk.PublicKey()
		signatures[i] = pk.Sign(msg)
	}

	var signature blsSignature
	if signersCount == 1 {
		signature = signatures[0]
	} else {
		signature = blsAggregateSignatures(signatures...)
	}

	return pubKeys, signature, msg
}

func parseInputData(pubKeys []blsPublicKey, signature blsSignature, msg []byte) ([]byte, []byte, []byte, error) {
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

// --- a blst based reference implementation of the BLS signature scheme, used for testing purposes only ---

// blsPrivateKey represents a BLS12-381 private key.
type blsPrivateKey struct {
	secretKey blst.SecretKey
}

// blsNewPrivateKey creates a new BLS12-381 private key. The resulting keys are
// cryptographically secure and can be used for production purposes.
func blsNewPrivateKey() blsPrivateKey {
	var inputKeyMaterial [32]byte
	// crypto/rand.Read is cryptographically secure, guaranteed to never fail
	_, _ = rand.Read(inputKeyMaterial[:])
	res := blsPrivateKey{}
	res.secretKey = *blst.KeyGen(inputKeyMaterial[:])
	return res
}

// blsPublicKey returns the public key corresponding to the private key.
func (k blsPrivateKey) PublicKey() blsPublicKey {
	res := blsPublicKey{}
	res.publicKey = *res.publicKey.From(&k.secretKey)
	return res
}

// Sign signs the provided message using the private key and returns the
// resulting signature.
func (k blsPrivateKey) Sign(message []byte) blsSignature {
	res := blsSignature{}
	res.sign.Sign(&k.secretKey, message, nil, false)
	return res
}

// blsPublicKey represents a BLS12-381 public key.
type blsPublicKey struct {
	publicKey blst.P1Affine
}

// Serialize exports the public key into a 48-byte array. This format can be
// used to serialize the key to disk or to transmit it over the network.
func (k blsPublicKey) Serialize() [48]byte {
	return [48]byte(k.publicKey.Compress())
}

// blsSignature represents a BLS12-381 signature.
type blsSignature struct {
	sign blst.P2Affine
}

// Serialize exports the signature into a 96-byte array. This format can be used
// to serialize the signature to disk or to transmit it over the network.
func (s blsSignature) Serialize() [96]byte {
	return [96]byte(s.sign.Compress())
}

// AggregateSignatures aggregates the provided signatures into a single
// signature. The provided signatures must be valid.
func blsAggregateSignatures(signatures ...blsSignature) blsSignature {
	agg := blst.P2Aggregate{}
	for _, sigs := range signatures {
		agg.Add(&sigs.sign, false)
	}
	return blsSignature{sign: *agg.ToAffine()}
}
