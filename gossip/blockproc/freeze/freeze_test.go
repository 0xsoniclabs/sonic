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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	priorities "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	subsidies "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
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

func TestRegistryChecker_DisabledReportsNothing(t *testing.T) {
	account := common.Address{0xf}
	db := registryState{registry.GetFrozenSlot(account): {31: 1}, registry.GetOptedOutSlot(account): {31: 1}}
	tests := map[string]struct {
		enabled bool
	}{
		"enabled":  {enabled: true},
		"disabled": {enabled: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			checker := NewRegistryChecker(test.enabled, db)
			require.Equal(t, test.enabled, checker.IsEnabled())
			require.Equal(t, test.enabled, checker.IsFrozen(account))
			require.Equal(t, test.enabled, checker.IsOptedOut(account))
		})
	}
}

func TestRegistryChecker_IsFrozen_ReadsRegistry(t *testing.T) {
	frozen, other := common.Address{0xf}, common.Address{0xa}
	for _, value := range listedValues {
		db := registryState{registry.GetFrozenSlot(frozen): value}
		require.True(t, NewRegistryChecker(true, db).IsFrozen(frozen), "%v", value)
		require.False(t, NewRegistryChecker(true, db).IsFrozen(other), "%v", value)
	}
}

func TestRegistryChecker_IsFrozen_ExemptsZeroAddressSystemAddressAndSystemContracts(t *testing.T) {
	for _, addr := range []common.Address{
		{},
		params.SystemAddress,
		registry.GetAddress(),
		sfc.ContractAddress,
		driver.ContractAddress,
		driverauth.ContractAddress,
		evmwriter.ContractAddress,
		subsidies.GetAddress(),
		priorities.GetAddress(),
	} {
		db := registryState{registry.GetFrozenSlot(addr): common.Hash{31: 1}}
		require.False(t, NewRegistryChecker(true, db).IsFrozen(addr), "%v", addr)
	}
}

func TestRegistryChecker_IsOptedOut_ReadsRegistry(t *testing.T) {
	token, other := common.Address{0x71}, common.Address{0x70}
	for _, value := range listedValues {
		db := registryState{registry.GetOptedOutSlot(token): value}
		require.True(t, NewRegistryChecker(true, db).IsOptedOut(token), "%v", value)
		require.False(t, NewRegistryChecker(true, db).IsOptedOut(other), "%v", value)
	}
}

func TestRegistryChecker_ReadsWhatTheRegistryContractWrites(t *testing.T) {
	db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	implementation := registry.GetImplementationAddress()
	db.SetCode(registry.GetAddress(), proxy.GetCode(), tracing.CodeChangeUnspecified)
	db.SetState(registry.GetAddress(), proxy.GetSlotForImplementation(), common.BytesToHash(implementation.Bytes()))
	db.SetCode(implementation, registry.GetCode(), tracing.CodeChangeUnspecified)

	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer: func(vm.StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		BlockNumber: big.NewInt(1),
		Time:        1,
		Random:      &common.Hash{},
	}, db, params.MergedTestChainConfig, vm.Config{})
	abi, err := registry.RegistryMetaData.GetAbi()
	require.NoError(t, err)
	call := func(method string, addr common.Address, value bool) {
		t.Helper()
		input, err := abi.Pack(method, addr, value)
		require.NoError(t, err)
		_, _, err = evm.Call(common.Address{}, registry.GetAddress(), input, vm.NewGasBudget(1_000_000, 0), new(uint256.Int))
		require.NoError(t, err)
	}

	account, token := common.Address{0xa}, common.Address{0x7}
	checker := NewRegistryChecker(true, db)
	require.False(t, checker.IsFrozen(account))
	require.False(t, checker.IsOptedOut(token))

	call("setFrozen", account, true)
	call("setOptedOut", token, true)
	require.True(t, checker.IsFrozen(account))
	require.True(t, checker.IsOptedOut(token))
	require.False(t, checker.IsFrozen(token))
	require.False(t, checker.IsOptedOut(account))

	call("setFrozen", account, false)
	call("setOptedOut", token, false)
	require.False(t, checker.IsFrozen(account))
	require.False(t, checker.IsOptedOut(token))
}
