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
	"strings"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
)

// unlockAccounts unlocks any account specifically requested.
func unlockAccounts(ctx *cli.Context, stack *node.Node) error {
	var unlocks []string
	inputs := strings.Split(ctx.GlobalString(flags.UnlockedAccountFlag.Name), ",")
	for _, input := range inputs {
		if trimmed := strings.TrimSpace(input); trimmed != "" {
			unlocks = append(unlocks, trimmed)
		}
	}
	// Short circuit if there is no account to unlock.
	if len(unlocks) == 0 {
		return nil
	}
	// If insecure account unlocking is not allowed if node's APIs are exposed to external.
	// Print warning log to user and skip unlocking.
	if !stack.Config().InsecureUnlockAllowed && stack.Config().ExtRPCEnabled() {
		return fmt.Errorf("account unlock with HTTP access is forbidden")
	}
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	passwords, err := config.MakePasswordList(ctx)
	if err != nil {
		return err
	}
	for i, account := range unlocks {
		if _, _, err := config.UnlockAccount(ks, account, i, passwords); err != nil {
			return err
		}
	}
	return nil
}
