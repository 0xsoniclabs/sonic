package encryption

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

func TestMigrateAccountToValidatorKey(t *testing.T) {
	dir := t.TempDir()

	// Create a fake account key (simulating go-ethereum keystore format)
	privKey, _ := crypto.GenerateKey()
	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)

	ks := New(2, 1)
	// First, create a validator key to get the crypto JSON, then rebuild as account key
	keyjson, err := ks.EncryptKey(
		validatorpk.PubKey{Type: validatorpk.Types.Secp256k1, Raw: pubBytes},
		crypto.FromECDSA(privKey),
		"password",
	)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Parse the encrypted key and re-create as an account key JSON
	var encKey EncryptedKeyJSON
	if err := json.Unmarshal(keyjson, &encKey); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	accKey := encryptedAccountKeyJSONV3{
		Address: "0000000000000000000000000000000000000001",
		Crypto:  encKey.Crypto,
		Id:      "test-id",
		Version: 3,
	}

	accKeyJSON, _ := json.Marshal(accKey)
	accKeyPath := filepath.Join(dir, "acckey.json")
	if err := os.WriteFile(accKeyPath, accKeyJSON, 0600); err != nil {
		t.Fatalf("failed to write account key file: %v", err)
	}

	// Migrate
	valKeyPath := filepath.Join(dir, "valkey.json")
	pubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  pubBytes,
	}

	err = MigrateAccountToValidatorKey(accKeyPath, valKeyPath, pubKey)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Verify the validator key file exists
	if _, err := os.Stat(valKeyPath); os.IsNotExist(err) {
		t.Fatal("validator key file should exist")
	}

	// Verify the validator key can be decrypted
	valKeyJSON, _ := os.ReadFile(valKeyPath)
	decrypted, err := DecryptKey(valKeyJSON, "password")
	if err != nil {
		t.Fatalf("failed to decrypt migrated key: %v", err)
	}
	if decrypted.Type != validatorpk.Types.Secp256k1 {
		t.Fatal("unexpected key type")
	}
}

func TestMigrateAccountToValidatorKey_FileNotFound(t *testing.T) {
	err := MigrateAccountToValidatorKey(
		"/nonexistent/path",
		"/tmp/out.json",
		validatorpk.PubKey{},
	)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMigrateAccountToValidatorKey_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	accKeyPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(accKeyPath, []byte("not json"), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	err := MigrateAccountToValidatorKey(
		accKeyPath,
		filepath.Join(dir, "out.json"),
		validatorpk.PubKey{},
	)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
