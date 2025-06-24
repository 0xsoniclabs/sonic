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
	"sync"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

type SyncedKeystore struct {
	backend KeystoreI
	mu      sync.Mutex
}

func NewSyncedKeystore(backend KeystoreI) *SyncedKeystore {
	return &SyncedKeystore{
		backend: backend,
	}
}

func (s *SyncedKeystore) Unlocked(pubkey validatorpk.PubKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Unlocked(pubkey)
}

func (s *SyncedKeystore) Has(pubkey validatorpk.PubKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Has(pubkey)
}

func (s *SyncedKeystore) Unlock(pubkey validatorpk.PubKey, auth string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Unlock(pubkey, auth)
}

func (s *SyncedKeystore) GetUnlocked(pubkey validatorpk.PubKey) (*encryption.PrivateKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.GetUnlocked(pubkey)
}

func (s *SyncedKeystore) Add(pubkey validatorpk.PubKey, key []byte, auth string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Add(pubkey, key, auth)
}

func (s *SyncedKeystore) Get(pubkey validatorpk.PubKey, auth string) (*encryption.PrivateKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Get(pubkey, auth)
}
