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

package freeze

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	frozenAccount = common.Address{0xf}
	otherAccount  = common.Address{0xa}
	token         = common.Address{0x70}
	optedOutToken = common.Address{0x71}
)

// fakeState holds the registry storage.
type fakeState struct {
	slots map[common.Hash]common.Hash
}

func newFakeState(frozen ...common.Address) *fakeState {
	s := &fakeState{slots: map[common.Hash]common.Hash{}}
	for _, a := range frozen {
		s.slots[registry.GetFrozenSlot(a)] = common.Hash{31: 1}
	}
	return s
}

func (s *fakeState) GetState(addr common.Address, key common.Hash) common.Hash {
	if addr != registry.GetAddress() {
		return common.Hash{}
	}
	return s.slots[key]
}

func TestIsFrozen_ReadsRegistryAndExemptsSystemContracts(t *testing.T) {
	db := newFakeState(frozenAccount, sfc.ContractAddress, registry.GetAddress())

	require.True(t, IsFrozen(db, frozenAccount))
	require.False(t, IsFrozen(db, otherAccount))
	require.False(t, IsFrozen(db, sfc.ContractAddress))
	require.False(t, IsFrozen(db, registry.GetAddress()))
}

func TestIsOptedOut_ReadsRegistry(t *testing.T) {
	db := newFakeState()
	db.slots[registry.GetOptedOutSlot(optedOutToken)] = common.Hash{31: 1}
	require.True(t, IsOptedOut(db, optedOutToken))
	require.False(t, IsOptedOut(db, token))
}
