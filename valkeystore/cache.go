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

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

var (
	ErrAlreadyUnlocked = errors.New("already unlocked")
	ErrLocked          = errors.New("key is locked")
)

type CachedKeystore struct {
	backend RawKeystoreI
	cache   map[string]*encryption.PrivateKey
}

func NewCachedKeystore(backend RawKeystoreI) *CachedKeystore {
	return &CachedKeystore{
		backend: backend,
		cache:   make(map[string]*encryption.PrivateKey),
	}
}

func (c *CachedKeystore) Unlocked(pubkey validatorpk.PubKey) bool {
	_, ok := c.cache[c.idxOf(pubkey)]
	return ok
}

func (c *CachedKeystore) Has(pubkey validatorpk.PubKey) bool {
	if c.Unlocked(pubkey) {
		return true
	}
	return c.backend.Has(pubkey)
}

func (c *CachedKeystore) Unlock(pubkey validatorpk.PubKey, auth string) error {
	if c.Unlocked(pubkey) {
		return ErrAlreadyUnlocked
	}
	key, err := c.backend.Get(pubkey, auth)
	if err != nil {
		return err
	}
	c.cache[c.idxOf(pubkey)] = key
	return nil
}

func (c *CachedKeystore) GetUnlocked(pubkey validatorpk.PubKey) (*encryption.PrivateKey, error) {
	if !c.Unlocked(pubkey) {
		return nil, ErrLocked
	}
	return c.cache[c.idxOf(pubkey)], nil
}

func (c *CachedKeystore) idxOf(pubkey validatorpk.PubKey) string {
	return string(pubkey.Bytes())
}

func (c *CachedKeystore) Add(pubkey validatorpk.PubKey, key []byte, auth string) error {
	return c.backend.Add(pubkey, key, auth)
}

func (c *CachedKeystore) Get(pubkey validatorpk.PubKey, auth string) (*encryption.PrivateKey, error) {
	return c.backend.Get(pubkey, auth)
}
