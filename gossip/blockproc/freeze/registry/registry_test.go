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

package registry

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// The client reads the registry's storage directly, so this pins the slots it
// computes to the layout of the compiled contract behind the proxy.
func TestRegistry_SlotsMatchContractLayoutBehindProxy(t *testing.T) {
	r := newRegistryBehindProxy(t)

	addr := common.Address{0: 0xa, 19: 0xb}
	for method, slot := range map[string]common.Hash{
		"setFrozen":   GetFrozenSlot(addr),
		"setOptedOut": GetOptedOutSlot(addr),
	} {
		r.call(method, addr, true)
		require.NotEqual(t, common.Hash{}, r.db.GetState(GetAddress(), slot), method)
		require.Equal(t, common.Hash{}, r.db.GetState(r.implementation, slot), method)
		r.call(method, addr, false)
		require.Equal(t, common.Hash{}, r.db.GetState(GetAddress(), slot), method)
	}
}

// Writing the slots the client reads must flip the contract's own views, and
// only the matching one. Unlike the setters, the views are the same in any
// registry, so this check carries over to a replacement implementation.
func TestRegistry_ViewsAgreeWithSlotsReadByClient(t *testing.T) {
	r := newRegistryBehindProxy(t)

	addr := common.Address{0: 0xa, 19: 0xb}
	slots := map[string]common.Hash{
		"isFrozen":   GetFrozenSlot(addr),
		"isOptedOut": GetOptedOutSlot(addr),
	}
	for set, slot := range slots {
		r.db.SetState(GetAddress(), slot, common.Hash{31: 1})
		for view := range slots {
			require.Equal(t, view == set, r.view(view, addr), "%s after setting the %s slot", view, set)
		}
		r.db.SetState(GetAddress(), slot, common.Hash{})
		require.False(t, r.view(set, addr), "%s after clearing its slot", set)
	}
}

type registryBehindProxy struct {
	t              *testing.T
	db             *state.StateDB
	evm            *vm.EVM
	abi            *abi.ABI
	implementation common.Address
}

func newRegistryBehindProxy(t *testing.T) *registryBehindProxy {
	db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)

	implementation := GetImplementationAddress()
	db.SetCode(GetAddress(), proxy.GetCode(), tracing.CodeChangeUnspecified)
	db.SetState(GetAddress(), proxy.GetSlotForImplementation(), common.BytesToHash(implementation.Bytes()))
	db.SetCode(implementation, GetCode(), tracing.CodeChangeUnspecified)

	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer: func(vm.StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		BlockNumber: big.NewInt(1),
		Time:        1,
		Random:      &common.Hash{},
	}, db, params.MergedTestChainConfig, vm.Config{})
	abi, err := RegistryMetaData.GetAbi()
	require.NoError(t, err)
	return &registryBehindProxy{t: t, db: db, evm: evm, abi: abi, implementation: implementation}
}

func (r *registryBehindProxy) call(method string, args ...any) {
	input, err := r.abi.Pack(method, args...)
	require.NoError(r.t, err)
	_, _, err = r.evm.Call(common.Address{}, GetAddress(), input, vm.NewGasBudget(1_000_000, 0), new(uint256.Int))
	require.NoError(r.t, err)
}

func (r *registryBehindProxy) view(method string, addr common.Address) bool {
	input, err := r.abi.Pack(method, addr)
	require.NoError(r.t, err)
	output, _, err := r.evm.StaticCall(common.Address{}, GetAddress(), input, vm.NewGasBudget(1_000_000, 0))
	require.NoError(r.t, err)
	result, err := r.abi.Unpack(method, output)
	require.NoError(r.t, err)
	return result[0].(bool)
}
