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
	priorities "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidies "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
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

// Any non-zero slot lists the address, not only Solidity's `true`.
var listedValues = []common.Hash{{31: 1}, {0: 1}}

func TestRegistryChecker_IsFrozen_ReadsRegistry(t *testing.T) {
	frozen, other := common.Address{0xf}, common.Address{0xa}
	for _, value := range listedValues {
		db := registryState{registry.GetFrozenSlot(frozen): value}
		require.True(t, NewRegistryChecker(db).IsFrozen(frozen), "%v", value)
		require.False(t, NewRegistryChecker(db).IsFrozen(other), "%v", value)
	}
}

func TestRegistryChecker_IsFrozen_ExemptsSystemContracts(t *testing.T) {
	for _, addr := range []common.Address{
		registry.GetAddress(),
		sfc.ContractAddress,
		driver.ContractAddress,
		driverauth.ContractAddress,
		evmwriter.ContractAddress,
		subsidies.GetAddress(),
		priorities.GetAddress(),
	} {
		db := registryState{registry.GetFrozenSlot(addr): common.Hash{31: 1}}
		require.False(t, NewRegistryChecker(db).IsFrozen(addr), "%v", addr)
	}
}

func TestRegistryChecker_IsOptedOut_ReadsRegistry(t *testing.T) {
	token, other := common.Address{0x71}, common.Address{0x70}
	for _, value := range listedValues {
		db := registryState{registry.GetOptedOutSlot(token): value}
		require.True(t, NewRegistryChecker(db).IsOptedOut(token), "%v", value)
		require.False(t, NewRegistryChecker(db).IsOptedOut(other), "%v", value)
	}
}
