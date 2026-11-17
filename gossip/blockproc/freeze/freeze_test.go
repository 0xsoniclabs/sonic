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

// registryState is a StateReader holding only the registry's storage.
type registryState map[common.Hash]common.Hash

func (s registryState) GetState(addr common.Address, key common.Hash) common.Hash {
	if addr != registry.GetAddress() {
		return common.Hash{}
	}
	return s[key]
}

func TestIsFrozen_ReadsRegistryAndExemptsSystemContracts(t *testing.T) {
	frozen, other := common.Address{0xf}, common.Address{0xa}
	db := registryState{}
	for _, a := range []common.Address{frozen, sfc.ContractAddress, registry.GetAddress()} {
		db[registry.GetFrozenSlot(a)] = common.Hash{31: 1}
	}

	require.True(t, IsFrozen(db, frozen))
	require.False(t, IsFrozen(db, other))
	require.False(t, IsFrozen(db, sfc.ContractAddress))
	require.False(t, IsFrozen(db, registry.GetAddress()))
}

func TestIsOptedOut_ReadsRegistry(t *testing.T) {
	optedOut, other := common.Address{0x71}, common.Address{0x70}
	db := registryState{registry.GetOptedOutSlot(optedOut): common.Hash{31: 1}}

	require.True(t, IsOptedOut(db, optedOut))
	require.False(t, IsOptedOut(db, other))
}
