package bls

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	gnark "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"
)

// Material:
// - bls spec: https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-bls-signature-04
// - eip2537: https://eips.ethereum.org/EIPS/eip-2537
// - blst lib: https://github.com/supranational/blst
// - gnark lib: https://github.com/Consensys/gnark-crypto

func TestBls_CommonSignature(t *testing.T) {
	key := NewPrivateKeyForTests(1)
	hello := []byte("hello")
	sig1 := blsBlst{}.sign(key, hello)
	sig2 := blsGnark{}.sign(key, hello)

	fmt.Printf("blst:  %v\n", sig1)
	fmt.Printf("gnark: %v\n", sig2)

	require.Equal(t, sig1, sig2)
}

func TestEip2537_CanValidateSignature(t *testing.T) {
	key := NewPrivateKeyForTests(1)
	pubKey := key.PublicKey()
	hello := []byte("hello")
	world := []byte("world")

	tests := map[string]blsLib{
		"blst":  blsBlst{},
		"gnark": blsGnark{},
	}

	for name, lib := range tests {
		t.Run(fmt.Sprintf("sign=%s", name), func(t *testing.T) {
			signature := lib.sign(key, hello)
			for name, lib := range tests {
				t.Run(fmt.Sprintf("verify=%s", name), func(t *testing.T) {
					res, err := lib.verify(pubKey, signature, hello)
					require.NoError(t, err)
					require.True(t, res)
					res, err = lib.verify(pubKey, signature, world)
					require.NoError(t, err)
					require.False(t, res)
				})
			}
		})
	}
}

type blsLib interface {
	sign(PrivateKey, []byte) Signature
	verify(PublicKey, Signature, []byte) (bool, error)
}

type blsBlst struct{}

func (blsBlst) sign(key PrivateKey, message []byte) Signature {
	return key.Sign(message)
}

func (blsBlst) verify(pubKey PublicKey, signature Signature, message []byte) (bool, error) {
	return signature.Verify(pubKey, message), nil
}

type blsGnark struct{}

func (blsGnark) sign(key PrivateKey, message []byte) Signature {
	private := new(big.Int).SetBytes(key.secretKey.ToBEndian())

	msg, err := gnark.HashToG2(message, nil)
	if err != nil {
		panic(fmt.Errorf("failed to hash message: %w", err))
	}

	sig := new(gnark.G2Affine).ScalarMultiplication(&msg, private)
	if !sig.IsInSubGroup() {
		panic("signature is not in the subgroup")
	}
	sigData := sig.Bytes()
	res, err := DeserializeSignature(sigData)
	if err != nil {
		panic(fmt.Errorf("failed to deserialize signature: %w", err))
	}
	return res
}

func (blsGnark) verify(pubKey PublicKey, signature Signature, message []byte) (bool, error) {

	// Convert the public key to the format expected by the gnark library.
	pub, err := publicKeyToGnarkG1Affine(pubKey)
	if err != nil {
		return false, fmt.Errorf("failed to convert public key: %w", err)
	}

	// Convert the signature to the format expected by the gnark library.
	sig, err := signatureToGnarkG2Affine(signature)
	if err != nil {
		return false, fmt.Errorf("failed to convert signature: %w", err)
	}

	// Convert the message to the format expected by the gnark library.
	msg, err := gnark.HashToG2(message, nil)
	if err != nil {
		return false, fmt.Errorf("failed to hash message: %w", err)
	}

	// Check that
	//  e(public,message) == e(generator,signature)
	// by checking that
	//  e(public, message) * e(-generator,signature) == 1
	// where e is the pairing function.
	//
	// Derivation:
	//  e(public,message) == e(generator,signature)
	//  e(public,message) * e(generator,signature)^-1 == e(generator,signature) * e(generator,signature)^-1
	//  e(public,message) * e(generator,signature)^-1 == 1
	// and the inverse of a pairing is a pairing with the same arguments but
	// with one of the arguments negated. Thus
	//  e(public,message) * e(-generator,signature) == 1
	//
	// TODO: verify this derivation;
	gen := getGeneratorG1()
	negGen := *new(gnark.G1Affine).Neg(&gen)
	return gnark.PairingCheck(
		[]gnark.G1Affine{pub, negGen},
		[]gnark.G2Affine{msg, sig},
	)
}






// --- experiments and helpers ---

func getGeneratorG1() gnark.G1Affine {
	res := gnark.G1Affine{}
	_, err1 := res.X.SetString("0x17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb")
	_, err2 := res.Y.SetString("0x08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1")
	if err := errors.Join(err1, err2); err != nil {
		panic(fmt.Errorf("failed to set generator G1: %w", err))
	}
	return res
}

func TestGnark_SignatureCheck(t *testing.T) {
	require := require.New(t)

	// Create a generator point for G1.
	generator := getGeneratorG1()
	require.True(generator.IsOnCurve())

	// Define a private key and compute its public key.
	private := big.NewInt(12)
	public := new(gnark.G1Affine).ScalarMultiplication(&generator, private)
	require.True(public.IsOnCurve())
	fmt.Printf("Public key: %v\n", public)

	// Define a message.
	msg1, err := gnark.HashToG2([]byte("hello"), nil)
	require.NoError(err)
	fmt.Printf("Message: %v\n", &msg1)

	msg2, err := gnark.HashToG2([]byte("world"), nil)
	require.NoError(err)
	fmt.Printf("Message: %v\n", &msg2)

	// Define a signature.
	signature := new(gnark.G2Affine).ScalarMultiplication(&msg1, private)
	require.True(signature.IsOnCurve())
	fmt.Printf("Signature: %v\n", signature)

	// Check preconditions for pairing check.
	require.True(generator.IsInSubGroup())
	require.True(public.IsInSubGroup())
	require.True(msg1.IsInSubGroup())
	require.True(msg2.IsInSubGroup())
	require.True(signature.IsInSubGroup())

	// Verify the signature
	negGenerator := *new(gnark.G1Affine).Neg(&generator)
	require.True(negGenerator.IsInSubGroup())

	valid, err := gnark.PairingCheck(
		[]gnark.G1Affine{*public, negGenerator},
		[]gnark.G2Affine{msg1, *signature},
	)
	require.NoError(err)
	require.True(valid)

	// Make sure that a different message can not be verified.
	valid, err = gnark.PairingCheck(
		[]gnark.G1Affine{*public, negGenerator},
		[]gnark.G2Affine{msg2, *signature},
	)
	require.NoError(err)
	require.False(valid)
}

func TestBlsPublicKeyToGnarkConversion(t *testing.T) {
	for range 100 {
		require := require.New(t)
		key := NewPrivateKey().PublicKey()
		point, err := publicKeyToGnarkG1Affine(key)
		require.NoError(err)
		require.True(point.IsOnCurve())
	}
}

func publicKeyToGnarkG1Affine(key PublicKey) (gnark.G1Affine, error) {
	// See the blst library serialization format:
	// https://github.com/supranational/blst?tab=readme-ov-file#serialization-format
	data := key.Serialize()
	var res gnark.G1Affine
	_, err := res.SetBytes(data[:])
	if err != nil {
		return gnark.G1Affine{}, err
	}
	return res, nil
}

func TestBlsSignatureToGnarkConversion(t *testing.T) {
	for range 100 {
		require := require.New(t)
		key := NewPrivateKey()
		signature := key.Sign([]byte("hello"))
		point, err := signatureToGnarkG2Affine(signature)
		require.NoError(err)
		require.True(point.IsOnCurve())
	}
}

func signatureToGnarkG2Affine(sig Signature) (gnark.G2Affine, error) {
	// See the blst library serialization format:
	// https://github.com/supranational/blst?tab=readme-ov-file#serialization-format
	data := sig.Serialize()
	var res gnark.G2Affine
	_, err := res.SetBytes(data[:])
	if err != nil {
		return gnark.G2Affine{}, err
	}
	return res, nil
}

func TestMessageToG2Mapping_BlstVsGnark_PerformSameMapping(t *testing.T) {
	msgs := []string{"", "hello", "world"}
	for _, msg := range msgs {
		t.Run(msg, func(t *testing.T) {
			require := require.New(t)
			point, err := gnark.EncodeToG2([]byte(msg), nil)
			require.NoError(err)
			require.True(point.IsOnCurve())

			point2 := blst.EncodeToG2([]byte(msg), nil)
			data := point2.Serialize()
			var res gnark.G2Affine
			_, err = res.SetBytes(data[:])
			require.NoError(err)
			require.True(res.IsOnCurve())

			require.True(point.Equal(&res))
		})
	}
}
