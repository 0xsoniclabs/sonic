// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package app

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/urfave/cli.v1"
)

func accountList(ctx *cli.Context) error {
	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	stack, err := config.MakeNetworkStack(ctx, &cfg.Node)
	if err != nil {
		return fmt.Errorf("failed to create the protocol stack: %w", err)
	}
	var index int
	for _, wallet := range stack.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			fmt.Printf("Account #%d: {%x} %s\n", index, account.Address, &account.URL)
			index++
		}
	}
	return nil
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) error {
	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	if err := config.SetNodeConfig(ctx, &cfg.Node); err != nil {
		return err
	}
	keydir, err := cfg.Node.KeyDirConfig()
	if err != nil {
		return fmt.Errorf("failed to read configuration: %w", err)
	}

	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if cfg.Node.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	passwordList, err := config.MakePasswordList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get password list: %w", err)
	}
	password, err := config.GetPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, passwordList)
	if err != nil {
		return fmt.Errorf("failed to get passphrase: %w", err)
	}

	account, err := keystore.StoreKey(keydir, password, scryptN, scryptP)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	fmt.Printf("\nYour new key was generated\n\n")
	fmt.Printf("Public address of the key:   %s\n", account.Address.Hex())
	fmt.Printf("Path of the secret key file: %s\n\n", account.URL.Path)
	fmt.Printf("- You can share your public address with anyone. Others need it to interact with you.\n")
	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your funds!\n")
	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to access account funds!\n")
	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
	return nil
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return fmt.Errorf("no accounts specified to update")
	}

	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	stack, err := config.MakeNetworkStack(ctx, &cfg.Node)
	if err != nil {
		return fmt.Errorf("failed to create the protocol stack: %w", err)
	}
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	for _, addr := range ctx.Args() {
		account, oldPassword, err := config.UnlockAccount(ks, addr, 0, nil)
		if err != nil {
			return err
		}
		newPassword, err := config.GetPassPhrase("Please give a new password. Do not forget this password.", true, 0, nil)
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %w", err)
		}
		if err := ks.Update(account, oldPassword, newPassword); err != nil {
			return fmt.Errorf("could not update the account: %w", err)
		}
	}
	return nil
}

func accountImport(ctx *cli.Context) error {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		return fmt.Errorf("keyfile must be given as argument")
	}
	key, err := crypto.LoadECDSA(keyfile)
	if err != nil {
		return fmt.Errorf("failed to load the private key: %v", err)
	}

	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	stack, err := config.MakeNetworkStack(ctx, &cfg.Node)
	if err != nil {
		return fmt.Errorf("failed to create the protocol stack: %w", err)
	}
	passwordList, err := config.MakePasswordList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get password list: %w", err)
	}
	passphrase, err := config.GetPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, passwordList)
	if err != nil {
		return fmt.Errorf("failed to get passphrase: %w", err)
	}

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportECDSA(key, passphrase)
	if err != nil {
		return fmt.Errorf("could not create the account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}
