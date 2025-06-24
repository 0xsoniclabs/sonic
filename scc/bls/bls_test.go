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

package bls

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Private Keys -----------------------------------------------------------

func TestPrivateKey_KeyGenerationProducesDifferentKeys(t *testing.T) {
	seen := map[PrivateKey]struct{}{}
	for range 10 {
		key := NewPrivateKey()
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate key generated")
		}
		seen[key] = struct{}{}
	}
}

func TestPrivateKey_TestKeyGenerationIsDeterministic(t *testing.T) {
	require := require.New(t)
	for i := range byte(10) {
		key1 := NewPrivateKeyForTests(i)
		key2 := NewPrivateKeyForTests(i)
		require.Equal(key1, key2)
	}
}

func TestPrivateKey_CanBeAssignedAndReAssigned(t *testing.T) {
	require := require.New(t)
	key1 := NewPrivateKey()
	key2 := NewPrivateKey()

	key3 := key1
	require.Equal(key1, key3)
	require.NotEqual(key2, key3)

	key3 = key2
	require.NotEqual(key1, key3)
	require.Equal(key2, key3)
}

func TestPrivateKey_CanBeSerializedAndDeserialized(t *testing.T) {
	require := require.New(t)
	key1 := NewPrivateKey()
	serialized := key1.Serialize()
	key2, err := DeserializePrivateKey(serialized)
	require.NoError(err)
	require.Equal(key1, key2)
}

func TestPrivateKey_DeserializationDetectsInvalidEncoding(t *testing.T) {
	serialized := [32]byte{} // <- invalid encoding
	_, err := DeserializePrivateKey(serialized)
	require.Error(t, err)
}

func TestPrivateKey_String_IsHexEncoded(t *testing.T) {
	regexp := regexp.MustCompile(`^0x[0-9a-f]{64}$`)
	key := NewPrivateKey()
	require.Regexp(t, regexp, key.String())
}

// --- Public Keys ------------------------------------------------------------

func TestPublicKey_DefaultIsInvalid(t *testing.T) {
	require.False(t, PublicKey{}.Validate())
}

func TestPublicKey_GeneratedKeysAreValid(t *testing.T) {
	require := require.New(t)
	for range 10 {
		require.True(NewPrivateKey().PublicKey().Validate())
	}
}

func TestPublicKey_CheckProofOfPossession_CanValidateValidProof(t *testing.T) {
	require := require.New(t)
	private := NewPrivateKey()
	proof := private.GetProofOfPossession()
	public := private.PublicKey()
	require.True(public.CheckProofOfPossession(proof))
}

func TestPublicKey_CheckProofOfPossession_DetectsInvalidProof(t *testing.T) {
	require := require.New(t)
	private := NewPrivateKey()
	public := private.PublicKey()

	test := map[string]Signature{
		"invalid proof": {},
		"wrong key":     NewPrivateKey().GetProofOfPossession(),
		"wrong message": private.Sign([]byte("wrong message")),
	}

	for name, proof := range test {
		t.Run(name, func(t *testing.T) {
			require.False(public.CheckProofOfPossession(proof))
		})
	}
}

func TestPublicKey_Aggregation_EmptyAggregationIsInvalid(t *testing.T) {
	require.False(t, AggregatePublicKeys().Validate())
}

func TestPublicKey_Aggregation_IsAssociative(t *testing.T) {
	key1 := NewPrivateKey().PublicKey()
	key2 := NewPrivateKey().PublicKey()
	key3 := NewPrivateKey().PublicKey()

	require.Equal(
		t,
		AggregatePublicKeys(AggregatePublicKeys(key1, key2), key3),
		AggregatePublicKeys(key1, AggregatePublicKeys(key2, key3)),
	)
}

func TestPublicKey_Aggregation_IsCommutative(t *testing.T) {
	key1 := NewPrivateKey().PublicKey()
	key2 := NewPrivateKey().PublicKey()
	key3 := NewPrivateKey().PublicKey()

	ref := AggregatePublicKeys(key1, key2, key3)
	require.Equal(t, ref, AggregatePublicKeys(key1, key2, key3))
	require.Equal(t, ref, AggregatePublicKeys(key1, key3, key2))
	require.Equal(t, ref, AggregatePublicKeys(key2, key1, key3))
	require.Equal(t, ref, AggregatePublicKeys(key2, key3, key1))
	require.Equal(t, ref, AggregatePublicKeys(key3, key1, key2))
	require.Equal(t, ref, AggregatePublicKeys(key3, key2, key1))
}

func TestPublicKey_Aggregation_IsNotIdempotent(t *testing.T) {
	key1 := NewPrivateKey().PublicKey()

	require.Equal(t, key1, AggregatePublicKeys(key1))
	require.NotEqual(t, key1, AggregatePublicKeys(key1, key1))
	require.NotEqual(t, key1, AggregatePublicKeys(key1, key1, key1))
}

func TestPublicKey_CanBeSerializedAndDeserialized(t *testing.T) {
	require := require.New(t)
	key1 := PublicKey{}
	serialized := key1.Serialize()
	key2, err := DeserializePublicKey(serialized)
	require.NoError(err)
	require.Equal(key1, key2)
}

func TestPublicKey_DeserializationDetectsInvalidEncoding(t *testing.T) {
	serialized := [48]byte{} // <- invalid encoding
	_, err := DeserializePublicKey(serialized)
	require.Error(t, err)
}

func TestPublicKey_String_IsHexEncoded(t *testing.T) {
	regexp := regexp.MustCompile(`^0x[0-9a-f]{96}$`)
	key := NewPrivateKey().PublicKey()
	require.Regexp(t, regexp, key.String())
}

func TestPublicKey_JsonEncodingUsesHexFormat(t *testing.T) {
	regexp := regexp.MustCompile(`^"0x[0-9a-f]{96}"$`)
	key := NewPrivateKey().PublicKey()
	data, err := json.Marshal(key)
	require.NoError(t, err)
	require.Regexp(t, regexp, string(data))
}

func TestPublicKey_CanBeMarshalAndUnmarshal(t *testing.T) {
	require := require.New(t)
	key := PublicKey{}
	data, err := json.Marshal(key)
	require.NoError(err)

	var key2 PublicKey
	err = json.Unmarshal(data, &key2)
	require.NoError(err)
	require.Equal(key, key2)
}

func TestPublicKey_UnmarshalFailsForInvalidData(t *testing.T) {
	data := []byte(`"0xg"`)
	var key PublicKey
	err := json.Unmarshal(data, &key)
	require.Error(t, err)
	invalid := [96]byte{}
	data = []byte(invalid[:])
	err = json.Unmarshal(data, &key)
	require.Error(t, err)
}

// --- Signatures -------------------------------------------------------------

func TestSignature_DefaultIsInvalid(t *testing.T) {
	require.False(t, Signature{}.Validate())
}

func TestSignature_GeneratedSignaturesAreValid(t *testing.T) {
	require := require.New(t)
	for i := range byte(10) {
		key := NewPrivateKey()
		require.True(key.Sign([]byte{i}).Validate())
	}
}

func TestSignature_Verify_AcceptsValidSignatures(t *testing.T) {
	require := require.New(t)
	private := NewPrivateKey()
	public := private.PublicKey()
	message := []byte("Hello, world!")
	signature := private.Sign(message)
	require.True(signature.Verify(public, message))
}

func TestSignature_Verify_DetectsInvalidSignatures(t *testing.T) {
	require := require.New(t)
	private := NewPrivateKey()
	public := private.PublicKey()
	message := []byte("Hello, world!")
	signature := private.Sign(message)

	test := map[string]struct {
		message []byte
		public  PublicKey
	}{
		"wrong message": {[]byte("wrong message"), public},
		"wrong key":     {message, NewPrivateKey().PublicKey()},
		"invalid key":   {message, PublicKey{}},
	}

	for name, data := range test {
		t.Run(name, func(t *testing.T) {
			require.False(signature.Verify(data.public, data.message))
		})
	}
}

func TestSignature_Aggregation_EmptyAggregationIsInvalid(t *testing.T) {
	require.False(t, AggregateSignatures().Validate())
}

func TestSignature_Aggregation_IsAssociative(t *testing.T) {
	msg := []byte("msg")
	sig1 := NewPrivateKey().Sign(msg)
	sig2 := NewPrivateKey().Sign(msg)
	sig3 := NewPrivateKey().Sign(msg)

	require.Equal(
		t,
		AggregateSignatures(AggregateSignatures(sig1, sig2), sig3),
		AggregateSignatures(sig1, AggregateSignatures(sig2, sig3)),
	)
}

func TestSignature_Aggregation_IsCommutative(t *testing.T) {
	msg := []byte("msg")
	sig1 := NewPrivateKey().Sign(msg)
	sig2 := NewPrivateKey().Sign(msg)
	sig3 := NewPrivateKey().Sign(msg)

	ref := AggregateSignatures(sig1, sig2, sig3)
	require.Equal(t, ref, AggregateSignatures(sig1, sig2, sig3))
	require.Equal(t, ref, AggregateSignatures(sig1, sig3, sig2))
	require.Equal(t, ref, AggregateSignatures(sig2, sig1, sig3))
	require.Equal(t, ref, AggregateSignatures(sig2, sig3, sig1))
	require.Equal(t, ref, AggregateSignatures(sig3, sig1, sig2))
	require.Equal(t, ref, AggregateSignatures(sig3, sig2, sig1))
}

func TestSignature_Aggregation_IsNotIdempotent(t *testing.T) {
	msg := []byte("msg")
	sig := NewPrivateKey().Sign(msg)

	require.Equal(t, sig, AggregateSignatures(sig))
	require.NotEqual(t, sig, AggregateSignatures(sig, sig))
	require.NotEqual(t, sig, AggregateSignatures(sig, sig, sig))
}

func TestSignature_CanBeSerializedAndDeserialized(t *testing.T) {
	require := require.New(t)
	signature1 := Signature{}
	serialized := signature1.Serialize()
	signature2, err := DeserializeSignature(serialized)
	require.NoError(err)
	require.Equal(signature1, signature2)
}

func TestSignature_DeserializationDetectsInvalidEncoding(t *testing.T) {
	serialized := [96]byte{} // <- invalid encoding
	_, err := DeserializeSignature(serialized)
	require.Error(t, err)
}

func TestSignature_String_IsHexEncoded(t *testing.T) {
	regexp := regexp.MustCompile(`^0x[0-9a-f]{192}$`)
	require.Regexp(t, regexp, Signature{}.String())
}

func TestSignature_JsonEncodingUsesHexFormat(t *testing.T) {
	regexp := regexp.MustCompile(`^"0x[0-9a-f]{192}"$`)
	signature := NewPrivateKey().Sign([]byte("Hello, world!"))
	data, err := json.Marshal(signature)
	require.NoError(t, err)
	require.Regexp(t, regexp, string(data))
}

func TestSignature_CanBeMarshalAndUnmarshal(t *testing.T) {
	require := require.New(t)
	signature := Signature{}
	data, err := json.Marshal(signature)
	require.NoError(err)

	var signature2 Signature
	err = json.Unmarshal(data, &signature2)
	require.NoError(err)
	require.Equal(signature, signature2)
}

func TestSignature_UnmarshalFailsForInvalidData(t *testing.T) {
	data := []byte(`"0xg"`)
	var signature Signature
	err := json.Unmarshal(data, &signature)
	require.Error(t, err)
	invalid := [192]byte{}
	data = []byte(invalid[:])
	err = json.Unmarshal(data, &signature)
	require.Error(t, err)
}

// --- Signature Protocol -----------------------------------------------------

func TestBsl_CanSignAndVerifySignature(t *testing.T) {
	private := NewPrivateKey()
	public := private.PublicKey()

	message := "Hello, world!"
	signature := private.Sign([]byte(message))

	if !signature.Verify(public, []byte(message)) {
		t.Fatalf("failed to verify signature")
	}
}

func TestBsl_DeserializedKeysAndSignaturesCanBeVerified(t *testing.T) {
	private := NewPrivateKey()
	public := private.PublicKey()

	message := "Hello, world!"
	signature := private.Sign([]byte(message))

	recoveredKey, err := DeserializePublicKey(public.Serialize())
	require.NoError(t, err)
	recoveredSignature, err := DeserializeSignature(signature.Serialize())
	require.NoError(t, err)

	if !recoveredSignature.Verify(recoveredKey, []byte(message)) {
		t.Fatalf("failed to verify signature")
	}
}

func TestBsl_AggregatedSignaturesCanBeVerified(t *testing.T) {
	private1 := NewPrivateKey()
	private2 := NewPrivateKey()
	public1 := private1.PublicKey()
	public2 := private2.PublicKey()

	message := []byte("Hello, world!")
	signature1 := private1.Sign(message)
	signature2 := private2.Sign(message)

	aggSignature := AggregateSignatures(signature1, signature2)
	aggPublic := AggregatePublicKeys(public1, public2)

	if !aggSignature.Verify(aggPublic, message) {
		t.Fatalf("failed to verify aggregated signature")
	}
}

func TestBsl_AggregatedSignaturesCanBeVerifiedAgainstPublicKeys(t *testing.T) {
	private1 := NewPrivateKey()
	private2 := NewPrivateKey()
	public1 := private1.PublicKey()
	public2 := private2.PublicKey()

	message := []byte("Hello, world!")
	signature1 := private1.Sign(message)
	signature2 := private2.Sign(message)

	aggSignature := AggregateSignatures(signature1, signature2)

	if !aggSignature.VerifyAll([]PublicKey{public1, public2}, message) {
		t.Fatalf("failed to verify aggregated signature")
	}
}

// TODO: test known signatures

// --- Benchmarks -------------------------------------------------------------

func BenchmarkKey_Generation(b *testing.B) {
	for range b.N {
		NewPrivateKey()
	}
}

func BenchmarkSignature_Signing(b *testing.B) {
	message := [32]byte{}
	private := NewPrivateKey()
	b.ResetTimer()
	for range b.N {
		private.Sign(message[:])
	}
}

func BenchmarkSignature_Verification(b *testing.B) {
	message := [32]byte{}
	private := NewPrivateKey()
	signature := private.Sign(message[:])
	public := private.PublicKey()
	b.ResetTimer()
	for range b.N {
		if !signature.Verify(public, message[:]) {
			b.Fail()
		}
	}
}

func BenchmarkSignature_VerificationAggregatedWithKeyAggregation(b *testing.B) {
	message := []byte("hello")
	private1 := NewPrivateKey()
	private2 := NewPrivateKey()

	signature := AggregateSignatures(
		private1.Sign(message),
		private2.Sign(message),
	)

	public1 := private1.PublicKey()
	public2 := private2.PublicKey()

	b.ResetTimer()
	for range b.N {
		public := AggregatePublicKeys(public1, public2)
		if !signature.Verify(public, message[:]) {
			b.Fail()
		}
	}
}
func BenchmarkSignature_VerificationAggregatedWithoutKeyAggregation(b *testing.B) {
	message := []byte("hello")
	private1 := NewPrivateKey()
	private2 := NewPrivateKey()

	signature := AggregateSignatures(
		private1.Sign(message),
		private2.Sign(message),
	)

	public1 := private1.PublicKey()
	public2 := private2.PublicKey()

	b.ResetTimer()
	for range b.N {
		if !signature.VerifyAll([]PublicKey{public1, public2}, message[:]) {
			b.Fail()
		}
	}
}

func BenchmarkSignature_Aggregation(b *testing.B) {
	message := [32]byte{}
	private := NewPrivateKey()
	signature := private.Sign(message[:])
	b.ResetTimer()
	for range b.N {
		AggregateSignatures(signature, signature)
	}
}
