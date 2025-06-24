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
	"errors"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

var (
	ErrNotFound      = errors.New("key is not found")
	ErrAlreadyExists = errors.New("key already exists")
)

type FileKeystore struct {
	enc *encryption.Keystore
	dir string
}

func NewFileKeystore(dir string, enc *encryption.Keystore) *FileKeystore {
	return &FileKeystore{
		enc: enc,
		dir: dir,
	}
}

func (f *FileKeystore) Has(pubkey validatorpk.PubKey) bool {
	return fileExists(f.PathOf(pubkey))
}

func (f *FileKeystore) Add(pubkey validatorpk.PubKey, key []byte, auth string) error {
	if f.Has(pubkey) {
		return ErrAlreadyExists
	}
	return f.enc.StoreKey(f.PathOf(pubkey), pubkey, key, auth)
}

func (f *FileKeystore) Get(pubkey validatorpk.PubKey, auth string) (*encryption.PrivateKey, error) {
	if !f.Has(pubkey) {
		return nil, ErrNotFound
	}
	return f.enc.ReadKey(pubkey, f.PathOf(pubkey), auth)
}

func (f *FileKeystore) PathOf(pubkey validatorpk.PubKey) string {
	return path.Join(f.dir, common.Bytes2Hex(pubkey.Bytes()))
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
