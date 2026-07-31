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
	"flag"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/stretchr/testify/require"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
)

func newTestValidatorKey(t *testing.T) (valkeystore.KeystoreI, validatorpk.PubKey) {
	t.Helper()
	tmpDir := t.TempDir()
	valKeystore := valkeystore.NewLightFileKeystore(path.Join(tmpDir, "validator"))
	key := makefakegenesis.FakeKey(1)
	pubkey := makefakegenesis.GetFakeValidators(1)[0].PubKey
	require.NoError(t, addFakeValidatorKey(nil, key, pubkey, valKeystore))
	return valKeystore, pubkey
}

func newCliContextWithPasswordFile(t *testing.T, path string) *cli.Context {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String(flags.ValidatorPasswordFlag.Name, path, "")
	return cli.NewContext(nil, fs, nil)
}

func TestUnlockValidatorKey_Successful(t *testing.T) {
	valKeystore, pubkey := newTestValidatorKey(t)

	tmpDir := t.TempDir()
	passwordPath := path.Join(tmpDir, "password.txt")
	require.NoError(t, os.WriteFile(passwordPath, []byte(validatorpk.FakePassword+"\n"), 0600))
	cliCtx := newCliContextWithPasswordFile(t, passwordPath)

	require.NoError(t, unlockValidatorKey(context.Background(), cliCtx, pubkey, valKeystore))
	require.True(t, valKeystore.Unlocked(pubkey))
}

func TestUnlockValidatorKey_PasswordPipe(t *testing.T) {
	valKeystore, pubkey := newTestValidatorKey(t)

	tmpDir := t.TempDir()
	fifoPath := path.Join(tmpDir, "password.fifo")
	require.NoError(t, syscall.Mkfifo(fifoPath, 0600))
	cliCtx := newCliContextWithPasswordFile(t, fifoPath)

	go func() { // write password into the pipe
		w, err := os.OpenFile(fifoPath, os.O_WRONLY, 0)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = w.Close() }()
		if _, err := w.Write([]byte(validatorpk.FakePassword)); err != nil {
			t.Error(err)
		}
	}()

	require.NoError(t, unlockValidatorKey(t.Context(), cliCtx, pubkey, valKeystore))
	require.True(t, valKeystore.Unlocked(pubkey))
}

func TestUnlockValidatorKey_PasswordPipeTimeout(t *testing.T) {
	valKeystore, pubkey := newTestValidatorKey(t)

	tmpDir := t.TempDir()
	fifoPath := path.Join(tmpDir, "password.fifo")
	require.NoError(t, syscall.Mkfifo(fifoPath, 0600))
	cliCtx := newCliContextWithPasswordFile(t, fifoPath)

	// nothing written into the pipe - expected to timeout
	sigCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	err := unlockValidatorKey(sigCtx, cliCtx, pubkey, valKeystore)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, valKeystore.Unlocked(pubkey))
}
