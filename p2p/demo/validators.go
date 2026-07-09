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
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/p2p/networks"
)

// numValidators is the size of the demo's hard-coded validator set. Valid
// validator IDs are 0..numValidators-1. Every process derives the identical set,
// so nodes agree on who the validators are without any shared configuration.
const numValidators = 4

// demoEpoch is the fixed consensus epoch the demo runs in. The validator set
// never changes, so a single constant epoch is enough for the handshake and the
// address directory.
const demoEpoch = 1

// validatorKey deterministically derives validator id's secp256k1 consensus key.
// The derivation is a pure function of the ID, so any process computes the same
// key for the same ID — the basis for a shared validator set with no coordination.
func validatorKey(id int) *ecdsa.PrivateKey {
	seed := crypto.Keccak256([]byte(fmt.Sprintf("sonic/p2p/demo/validator/%d", id)))
	for {
		key, err := crypto.ToECDSA(seed)
		if err == nil {
			return key
		}
		// A hashed seed is almost always a valid scalar; on the rare miss, hash
		// again so the result stays deterministic.
		seed = crypto.Keccak256(seed)
	}
}

// members returns the full demo validator set: each validator's ID and its
// compressed secp256k1 public key. The key is compressed to match the
// authenticator, whose Signer.PublicKey also returns the compressed form.
func members() []networks.Member {
	set := make([]networks.Member, 0, numValidators)
	for id := 0; id < numValidators; id++ {
		key := validatorKey(id)
		set = append(set, networks.Member{
			ID:        uint32(id),
			PublicKey: crypto.CompressPubkey(&key.PublicKey),
		})
	}
	return set
}

// staticMembership is the demo's fixed Membership: a constant validator set and
// epoch that never change, so OnChange never fires.
type staticMembership struct{}

func (staticMembership) Members() []networks.Member { return members() }

func (staticMembership) Epoch() uint64 { return demoEpoch }

func (staticMembership) OnChange(func()) (cancel func()) { return func() {} }
