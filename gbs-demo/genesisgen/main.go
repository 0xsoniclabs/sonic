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

// Command genesisgen produces a JSON genesis file for the transaction-priorities
// demo network. The genesis enables the TransactionPriorities upgrade (which
// pre-deploys the priority registry), creates the requested number of fake-net
// validators and pre-funds a set of demo user accounts derived from the
// well-known fake keys.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

// demoAccount describes a pre-funded account that is handed out to a demo user.
type demoAccount struct {
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	Address    string `json:"address"`
}

func main() {
	numValidators := flag.Int("validators", 5, "number of fake-net validators")
	numUsers := flag.Int("users", 10, "number of pre-funded demo user accounts")
	balanceS := flag.Uint64("balance", 1_000_000_000, "balance per demo user account, in whole tokens (S)")
	out := flag.String("out", "genesis.json", "output path for the genesis JSON file")
	accountsOut := flag.String("accounts", "", "optional output path for the demo accounts (private keys + addresses) as JSON")
	flag.Parse()

	if *numValidators < 1 {
		fatal("validators must be >= 1")
	}

	// Brio feature set plus the transaction-priorities feature under test. Enabling
	// TransactionPriorities makes GenerateFakeJsonGenesis pre-deploy the priority
	// registry (proxy + implementation).
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true

	gen := makefakegenesis.GenerateFakeJsonGenesis(
		upgrades,
		makefakegenesis.CreateEqualValidatorStake(*numValidators),
	)

	// Pre-fund demo user accounts. Their keys are the well-known fake keys that
	// follow the validator keys, so they are deterministic and easy to hand out.
	users := make([]demoAccount, 0, *numUsers)
	for i := range *numUsers {
		key := makefakegenesis.FakeKey(idx.ValidatorID(*numValidators + 1 + i))
		addr := crypto.PubkeyToAddress(key.PublicKey)
		name := fmt.Sprintf("demo-user-%d", i+1)
		gen.Accounts = append(gen.Accounts, makefakegenesis.Account{
			Name:    name,
			Address: addr,
			Balance: utils.ToFtmU256(*balanceS),
		})
		users = append(users, demoAccount{
			Name:       name,
			PrivateKey: hexutil.Encode(crypto.FromECDSA(key)),
			Address:    addr.Hex(),
		})
	}

	writeJSON(*out, gen)
	if *accountsOut != "" {
		writeJSON(*accountsOut, users)
	}
}

func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(fmt.Sprintf("failed to marshal %s: %v", path, err))
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatal(fmt.Sprintf("failed to write %s: %v", path, err))
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "genesisgen:", msg)
	os.Exit(1)
}
