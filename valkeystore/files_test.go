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

package valkeystore

import (
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

func TestFileKeystoreAdd(t *testing.T) {
	dir := t.TempDir()

	require := require.New(t)
	keystore := NewFileKeystore(dir, encryption.New(keystore.LightScryptN, keystore.LightScryptP))

	key, err := keystore.Get(pubkey1, "auth1")
	require.EqualError(err, ErrNotFound.Error())
	require.Nil(key)

	err = keystore.Add(pubkey1, key1, "auth1")
	require.NoError(err)
	_, err = os.Stat(path.Join(dir, name1))
	require.NoError(err)

	testGet(t, keystore, pubkey1, key1, "auth1")

	err = keystore.Add(pubkey2, key2, "auth2")
	require.NoError(err)
	_, err = os.Stat(path.Join(dir, name2))
	require.NoError(err)

	testGet(t, keystore, pubkey1, key1, "auth1")
	testGet(t, keystore, pubkey2, key2, "auth2")

	err = keystore.Add(pubkey2, key2, "auth1")
	require.Error(err, ErrAlreadyExists.Error())

	testGet(t, keystore, pubkey2, key2, "auth2")
}

func TestFileKeystoreRead(t *testing.T) {
	dir := t.TempDir()
	require := require.New(t)
	keystore := NewFileKeystore(dir, encryption.New(keystore.LightScryptN, keystore.LightScryptP))

	fd, err := os.Create(path.Join(dir, name1))
	require.NoError(err)
	_, err = fd.Write(file1)
	require.NoError(err)

	testGet(t, keystore, pubkey1, key1, "auth1")

	fd, err = os.Create(path.Join(dir, name2))
	require.NoError(err)
	_, err = fd.Write(file2)
	require.NoError(err)

	testGet(t, keystore, pubkey1, key1, "auth1")
	testGet(t, keystore, pubkey2, key2, "auth2")
}
