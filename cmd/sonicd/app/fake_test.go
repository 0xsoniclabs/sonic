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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/version"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

const (
	ipcAPIs = "abft:1.0 admin:1.0 dag:1.0 debug:1.0 ftm:1.0 net:1.0 personal:1.0 rpc:1.0 trace:1.0 txpool:1.0 web3:1.0"
)

func TestFakeNetFlag_NonValidator(t *testing.T) {
	// Start an sonic console, make sure it's cleaned up and terminate the console
	dataDir := tmpdir(t)
	initFakenetDatadir(dataDir, 3)
	cli := exec(t,
		"--fakenet", "0/3",
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--datadir", dataDir,
		"--exitwhensynced.epoch", "1")

	// Gather all the infos the welcome message needs to contain
	cli.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	cli.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	cli.SetTemplateFunc("gover", runtime.Version)
	cli.SetTemplateFunc("version", func() string { return version.String() })
	cli.SetTemplateFunc("niltime", genesisStart)
	cli.SetTemplateFunc("apis", func() string { return ipcAPIs })
	cli.ExpectExit()

	wantMessages := []string{
		"Unlocked fake validator",
	}
	for _, m := range wantMessages {
		if strings.Contains(cli.StderrText(), m) {
			t.Errorf("stderr text contains %q", m)
		}
	}
}

func TestFakeNetFlag_Validator(t *testing.T) {
	// Start a sonic console, make sure it's cleaned up and terminate the console
	dataDir := tmpdir(t)
	initFakenetDatadir(dataDir, 3)
	cli := exec(t,
		"--fakenet", "3/3",
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--datadir", dataDir,
		"--exitwhensynced.epoch", "1")

	// Gather all the infos the welcome message needs to contain
	va := readFakeValidator("3/3")
	cli.Coinbase = "0x0000000000000000000000000000000000000000"
	cli.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	cli.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	cli.SetTemplateFunc("gover", runtime.Version)
	cli.SetTemplateFunc("version", func() string { return version.String() })
	cli.SetTemplateFunc("niltime", genesisStart)
	cli.SetTemplateFunc("apis", func() string { return ipcAPIs })
	cli.ExpectExit()

	wantMessages := []string{
		"Unlocked validator key",
		"pubkey=" + va.String(),
	}
	for _, m := range wantMessages {
		if !strings.Contains(cli.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestFakeNEtFlag_ValidatorID_NotPresentInGenesis_FailsToStart(t *testing.T) {
	// Start a sonic console, make sure it's cleaned up and terminate the console
	dataDir := tmpdir(t)
	// initialize with a single validator
	initFakenetDatadir(dataDir, 1)
	cli := exec(t,
		"--fakenet", "3/3",
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--datadir", dataDir,
		"--exitwhensynced.epoch", "1")

	// Gather all the infos the welcome message needs to contain
	cli.ExpectExit()

	wantMessages := []string{
		"failed to initialize the node: validator ID 3 is not in the validator set",
	}
	for _, m := range wantMessages {
		if !strings.Contains(cli.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func readFakeValidator(fakenet string) *validatorpk.PubKey {
	n, _, err := config.ParseFakeGen(fakenet)
	if err != nil {
		panic(err)
	}

	if n < 1 {
		return nil
	}

	return &validatorpk.PubKey{
		Raw:  crypto.FromECDSAPub(&makefakegenesis.FakeKey(n).PublicKey),
		Type: validatorpk.Types.Secp256k1,
	}
}

func genesisStart() string {
	return time.Unix(int64(makefakegenesis.FakeGenesisTime.Unix()), 0).Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
}
