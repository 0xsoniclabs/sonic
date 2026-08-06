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

package config

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0xsoniclabs/sonic/config/flags"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
)

func addFakeValidatorKey(ctx *cli.Context, key *ecdsa.PrivateKey, pubkey validatorpk.PubKey, valKeystore valkeystore.RawKeystoreI) error {
	// add fake validator key
	if key != nil && !valKeystore.Has(pubkey) {
		err := valKeystore.Add(pubkey, crypto.FromECDSA(key), validatorpk.FakePassword)
		if err != nil {
			return fmt.Errorf("failed to add fake validator key: %v", err)
		}
	}
	return nil
}

// makeValidatorPasswordList reads password lines from the file specified by the global --validator.password flag.
func makeValidatorPasswordList(sigCtx context.Context, cliCtx *cli.Context) ([]string, error) {
	if path := cliCtx.GlobalString(flags.ValidatorPasswordFlag.Name); path != "" {
		log.Info("Reading validator password from file...", "path", path)
		text, err := readFileWithContext(sigCtx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read validator password file: %w", err)
		}
		lines := strings.Split(string(text), "\n")
		// Sanitise DOS line endings.
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], "\r")
		}
		return lines, nil
	}
	if cliCtx.GlobalIsSet(FakeNetFlag.Name) {
		return []string{validatorpk.FakePassword}, nil
	}
	return nil, nil
}

func unlockValidatorKey(sigCtx context.Context, cliCtx *cli.Context, pubKey validatorpk.PubKey, valKeystore valkeystore.KeystoreI) error {
	if !valKeystore.Has(pubKey) {
		return valkeystore.ErrNotFound
	}
	var err error
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking validator key %s | Attempt %d/%d", pubKey.String(), trials+1, 3)
		passwordList, err := makeValidatorPasswordList(sigCtx, cliCtx)
		if err != nil {
			return err
		}
		password, err := GetPassPhrase(sigCtx, prompt, false, 0, passwordList)
		if err != nil {
			return err
		}
		err = valKeystore.Unlock(pubKey, password)
		if err == nil {
			log.Info("Unlocked validator key", "pubkey", pubKey.String())
			return nil
		}
		if err.Error() != "could not decrypt key with given password" {
			return err
		}
	}
	// All trials expended to unlock account, bail out
	return err
}

func readFileWithContext(ctx context.Context, path string) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	resultChan := make(chan result, 1)
	go func() {
		f, err := os.Open(path) // blocks on a pipe
		if err != nil {
			resultChan <- result{nil, err}
			return
		}
		defer func() { _ = f.Close() }()

		data, err := io.ReadAll(f) // blocks on a pipe
		resultChan <- result{data, err}
	}()

	select {
	case res := <-resultChan:
		return res.data, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
