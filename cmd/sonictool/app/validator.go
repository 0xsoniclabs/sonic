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

package app

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/urfave/cli.v1"
)

// validatorKeyCreate creates a new validator key into the keystore defined by the CLI flags.
func validatorKeyCreate(ctx *cli.Context) error {
	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	if err := config.SetNodeConfig(ctx, &cfg.Node); err != nil {
		return err
	}

	passwordList, err := config.MakePasswordList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get password list: %w", err)
	}
	password, err := config.GetPassPhrase("Your new validator key is locked with a password. Please give a password. Do not forget this password.", true, 0, passwordList)
	if err != nil {
		return fmt.Errorf("failed to get passphrase: %w", err)
	}

	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	privateKey := crypto.FromECDSA(privateKeyECDSA)
	publicKey := validatorpk.PubKey{
		Raw:  crypto.FromECDSAPub(&privateKeyECDSA.PublicKey),
		Type: validatorpk.Types.Secp256k1,
	}

	keystoreDir, err := cfg.Node.KeyDirConfig()
	if err != nil {
		return fmt.Errorf("failed to setup account config: %w", err)
	}

	valKeystore := valkeystore.NewDefaultFileRawKeystore(path.Join(keystoreDir, "validator"))
	err = valKeystore.Add(publicKey, privateKey, password)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	// Sanity check
	_, err = valKeystore.Get(publicKey, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt the account: %w", err)
	}

	fmt.Printf("\nYour new key was generated\n\n")
	fmt.Printf("Public key:                  %s\n", publicKey.String())
	fmt.Printf("Public address of the key:   %s\n", crypto.PubkeyToAddress(privateKeyECDSA.PublicKey))
	fmt.Printf("Path of the secret key file: %s\n\n", valKeystore.PathOf(publicKey))
	fmt.Printf("- You can share your public key with anyone. Others need it to validate messages from you.\n")
	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your validator!\n")
	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to operate the validator!\n")
	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
	return nil
}

// validatorKeyConvert converts account key to validator key.
func validatorKeyConvert(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		return fmt.Errorf("this command requires 2 arguments")
	}
	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	if err := config.SetNodeConfig(ctx, &cfg.Node); err != nil {
		return err
	}

	keydir, _ := cfg.Node.KeyDirConfig()

	pubkeyStr := ctx.Args().Get(1)
	pubkey, err := validatorpk.FromString(pubkeyStr)
	if err != nil {
		return fmt.Errorf("failed to decode the validator pubkey: %w", err)
	}

	var acckeypath string
	if strings.HasPrefix(ctx.Args().First(), "0x") {
		acckeypath, err = findAccountKeypath(common.HexToAddress(ctx.Args().First()), keydir)
		if err != nil {
			return fmt.Errorf("failed to find the account: %w", err)
		}
	} else {
		acckeypath = ctx.Args().First()
	}

	valkeypath := path.Join(keydir, "validator", common.Bytes2Hex(pubkey.Bytes()))
	err = encryption.MigrateAccountToValidatorKey(acckeypath, valkeypath, pubkey)
	if err != nil {
		return fmt.Errorf("failed to migrate the account key: %w", err)
	}
	fmt.Println("\nYour key was converted and saved to " + valkeypath)
	return nil
}

func findAccountKeypath(addr common.Address, keydir string) (keypath string, err error) {
	addrStr := strings.ToLower(addr.String())[2:]
	// find key path
	err = filepath.Walk(keydir, func(walk string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		_, filename := filepath.Split(walk)
		if strings.Contains(strings.ToLower(filename), addrStr) {
			keypath = walk
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return keypath, err
	}
	if len(keypath) == 0 {
		return keypath, errors.New("account not found")
	}
	return keypath, nil
}
