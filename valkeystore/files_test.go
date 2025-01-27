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
