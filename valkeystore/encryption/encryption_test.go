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

package encryption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

func TestNew(t *testing.T) {
	ks := New(2, 1)
	if ks == nil {
		t.Fatal("expected non-nil Keystore")
	}
	if ks.scryptN != 2 {
		t.Fatalf("expected scryptN 2, got %d", ks.scryptN)
	}
	if ks.scryptP != 1 {
		t.Fatalf("expected scryptP 1, got %d", ks.scryptP)
	}
}

func TestEncryptAndDecrypt(t *testing.T) {
	ks := New(2, 1) // low scrypt params for fast testing

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
	pubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  pubBytes,
	}
	keyBytes := crypto.FromECDSA(privKey)

	password := "testpassword"

	keyjson, err := ks.EncryptKey(pubKey, keyBytes, password)
	if err != nil {
		t.Fatalf("failed to encrypt key: %v", err)
	}
	if len(keyjson) == 0 {
		t.Fatal("expected non-empty encrypted JSON")
	}

	decrypted, err := DecryptKey(keyjson, password)
	if err != nil {
		t.Fatalf("failed to decrypt key: %v", err)
	}

	if decrypted.Type != validatorpk.Types.Secp256k1 {
		t.Fatal("unexpected key type")
	}
	if len(decrypted.Bytes) == 0 {
		t.Fatal("expected non-empty decrypted bytes")
	}
	if decrypted.Decoded == nil {
		t.Fatal("expected non-nil decoded key")
	}
}

func TestEncryptKey_UnsupportedType(t *testing.T) {
	ks := New(2, 1)
	pubKey := validatorpk.PubKey{
		Type: 99, // unsupported
		Raw:  make([]byte, 33),
	}

	_, err := ks.EncryptKey(pubKey, []byte("key"), "pass")
	if err != ErrNotSupportedType {
		t.Fatalf("expected ErrNotSupportedType, got %v", err)
	}
}

func TestDecryptKey_WrongPassword(t *testing.T) {
	ks := New(2, 1)

	privKey, _ := crypto.GenerateKey()
	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
	pubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  pubBytes,
	}

	keyjson, err := ks.EncryptKey(pubKey, crypto.FromECDSA(privKey), "correctpassword")
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	_, err = DecryptKey(keyjson, "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestDecryptKey_InvalidJSON(t *testing.T) {
	_, err := DecryptKey([]byte("not json"), "pass")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecryptKey_UnsupportedType(t *testing.T) {
	json := `{"type":99,"pubkey":"aabb","crypto":{}}`
	_, err := DecryptKey([]byte(json), "pass")
	if err != ErrNotSupportedType {
		t.Fatalf("expected ErrNotSupportedType, got %v", err)
	}
}

func TestStoreAndReadKey(t *testing.T) {
	dir := t.TempDir()
	ks := New(2, 1)

	privKey, _ := crypto.GenerateKey()
	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
	pubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  pubBytes,
	}
	keyBytes := crypto.FromECDSA(privKey)

	filename := filepath.Join(dir, "testkey.json")
	err := ks.StoreKey(filename, pubKey, keyBytes, "testpass")
	if err != nil {
		t.Fatalf("failed to store key: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("key file should exist")
	}

	// Read it back
	decoded, err := ks.ReadKey(pubKey, filename, "testpass")
	if err != nil {
		t.Fatalf("failed to read key: %v", err)
	}
	if decoded.Type != validatorpk.Types.Secp256k1 {
		t.Fatal("unexpected key type")
	}
}

func TestReadKey_FileNotFound(t *testing.T) {
	ks := New(2, 1)
	pubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  make([]byte, 33),
	}

	_, err := ks.ReadKey(pubKey, "/nonexistent/path/key.json", "pass")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadKey_WrongPubkey(t *testing.T) {
	dir := t.TempDir()
	ks := New(2, 1)

	privKey, _ := crypto.GenerateKey()
	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
	correctPubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  pubBytes,
	}

	filename := filepath.Join(dir, "testkey.json")
	err := ks.StoreKey(filename, correctPubKey, crypto.FromECDSA(privKey), "pass")
	if err != nil {
		t.Fatalf("failed to store key: %v", err)
	}

	// Try reading with different pubkey
	wrongPubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  make([]byte, 65), // wrong pubkey
	}
	_, err = ks.ReadKey(wrongPubKey, filename, "pass")
	if err == nil {
		t.Fatal("expected error for wrong pubkey")
	}
}
