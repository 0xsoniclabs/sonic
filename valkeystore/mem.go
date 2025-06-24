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

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

type MemKeystore struct {
	mem  map[string]*encryption.PrivateKey
	auth map[string]string
}

func NewMemKeystore() *MemKeystore {
	return &MemKeystore{
		mem:  make(map[string]*encryption.PrivateKey),
		auth: make(map[string]string),
	}
}

func (m *MemKeystore) Has(pubkey validatorpk.PubKey) bool {
	_, ok := m.mem[m.idxOf(pubkey)]
	return ok
}

func (m *MemKeystore) Add(pubkey validatorpk.PubKey, key []byte, auth string) error {
	if m.Has(pubkey) {
		return ErrAlreadyExists
	}
	if pubkey.Type != validatorpk.Types.Secp256k1 {
		return encryption.ErrNotSupportedType
	}
	decoded, err := crypto.ToECDSA(key)
	if err != nil {
		return err
	}
	m.mem[m.idxOf(pubkey)] = &encryption.PrivateKey{
		Type:    pubkey.Type,
		Bytes:   key,
		Decoded: decoded,
	}
	m.auth[m.idxOf(pubkey)] = auth
	return nil
}

func (m *MemKeystore) Get(pubkey validatorpk.PubKey, auth string) (*encryption.PrivateKey, error) {
	if !m.Has(pubkey) {
		return nil, ErrNotFound
	}
	if m.auth[m.idxOf(pubkey)] != auth {
		return nil, errors.New("could not decrypt key with given password")
	}
	return m.mem[m.idxOf(pubkey)], nil
}

func (m *MemKeystore) idxOf(pubkey validatorpk.PubKey) string {
	return string(pubkey.Bytes())
}
