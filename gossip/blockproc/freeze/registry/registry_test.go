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
	call := func(method string, args ...any) {
		input, err := abi.Pack(method, args...)
		require.NoError(t, err)
		_, _, err = evm.Call(common.Address{}, GetAddress(), input, 1_000_000, new(uint256.Int))
		require.NoError(t, err)
	}

	account := common.Address{0xa}
	for method, slot := range map[string]common.Hash{
		"setFrozen":   GetFrozenSlot(account),
		"setOptedOut": GetOptedOutSlot(account),
	} {
		call(method, account, true)
		require.NotEqual(t, common.Hash{}, db.GetState(GetAddress(), slot), method)
		require.Equal(t, common.Hash{}, db.GetState(implementation, slot), method)
		call(method, account, false)
		require.Equal(t, common.Hash{}, db.GetState(GetAddress(), slot), method)
	}
}
