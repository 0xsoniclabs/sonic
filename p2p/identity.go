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

package p2p

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// loadOrCreateHostKey returns the libp2p host private key to use for this node.
//
// When path is empty the key is generated fresh in memory and never written to
// disk: this is the default and yields a new peer identity on every start,
// which is more private and adequate for validators (authenticated via a
// separate binding proof) and observers. When path is non-empty the key is
// loaded from that file if it exists, or generated and persisted there
// otherwise, providing a stable peer ID across restarts as needed by archives.
func loadOrCreateHostKey(path string) (crypto.PrivKey, error) {
	if path == "" {
		return generateHostKey()
	}

	if data, err := os.ReadFile(path); err == nil {
		key, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key at %s: %w", path, err)
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read host key at %s: %w", path, err)
	}

	key, err := generateHostKey()
	if err != nil {
		return nil, err
	}
	data, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to encode host key: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to persist host key at %s: %w", path, err)
	}
	return key, nil
}

// generateHostKey creates a fresh Ed25519 host key. Ed25519 is used for the
// network identity because it is fast and independent of the secp256k1
// consensus keys, keeping the two identity systems decoupled.
func generateHostKey() (crypto.PrivKey, error) {
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}
	return key, nil
}
