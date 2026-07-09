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

package main

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// TestValidatorKey_SameID_IsDeterministic is the property that lets every process
// agree on the validator set: the same ID always derives the same key.
func TestValidatorKey_SameID_IsDeterministic(t *testing.T) {
	for id := 0; id < numValidators; id++ {
		if !bytes.Equal(crypto.FromECDSA(validatorKey(id)), crypto.FromECDSA(validatorKey(id))) {
			t.Fatalf("validatorKey(%d) is not deterministic", id)
		}
	}
}

// TestValidatorKey_DifferentIDs_ProduceDistinctKeys ensures each validator has a
// distinct identity.
func TestValidatorKey_DifferentIDs_ProduceDistinctKeys(t *testing.T) {
	seen := make(map[string]int)
	for id := 0; id < numValidators; id++ {
		key := string(crypto.FromECDSA(validatorKey(id)))
		if prev, ok := seen[key]; ok {
			t.Fatalf("validatorKey(%d) collides with validatorKey(%d)", id, prev)
		}
		seen[key] = id
	}
}

// TestMembers_ReturnsDistinctCompressedKeys checks the member set has one entry
// per validator with a distinct 33-byte compressed public key (the encoding the
// authenticator compares against).
func TestMembers_ReturnsDistinctCompressedKeys(t *testing.T) {
	set := members()
	if len(set) != numValidators {
		t.Fatalf("got %d members, want %d", len(set), numValidators)
	}
	seen := make(map[string]bool)
	for i, member := range set {
		if member.ID != uint32(i) {
			t.Fatalf("member %d has ID %d", i, member.ID)
		}
		if len(member.PublicKey) != 33 {
			t.Fatalf("member %d public key is %d bytes, want 33 (compressed)", i, len(member.PublicKey))
		}
		key := string(member.PublicKey)
		if seen[key] {
			t.Fatalf("member %d has a duplicate public key", i)
		}
		seen[key] = true
	}
}
