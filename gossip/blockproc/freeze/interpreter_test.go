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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// newFreezeEVM returns an EVM running geth's interpreter with the freeze's
// checks, for a transaction of origin. frozenAccount is frozen; it and token
// hold code that stops at once.
func newFreezeEVM(t *testing.T, origin common.Address) (*vm.EVM, *state.StateDB) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	statedb.SetState(registry.GetAddress(), registry.GetFrozenSlot(frozenAccount), common.Hash{31: 1})
	for _, account := range []common.Address{frozenAccount, token} {
		statedb.SetCode(account, []byte{byte(vm.STOP)}, tracing.CodeChangeUnspecified)
	}
	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		BlockNumber: big.NewInt(1),
		Random:      &common.Hash{},
	}, statedb, params.MergedTestChainConfig, vm.Config{
		Interpreter: WrapInterpreter(func(evm *vm.EVM) vm.Interpreter { return vm.NewEvmInterpreter(evm) }),
	})
	evm.SetTxContext(vm.TxContext{Origin: origin})
	return evm, statedb
}

func TestFrozenRevert_DecodesToReason(t *testing.T) {
	reason, err := abi.UnpackRevert(frozenRevert)
	require.NoError(t, err)
	require.Equal(t, "account is frozen", reason)
}

func TestInterpreter_RefusesCallsByOrToFrozenAccountsAndTokenCallsNamingThem(t *testing.T) {
	operator := common.Address{0x2}
	tests := map[string]struct {
		sender, to common.Address
		input      []byte
		refused    bool
	}{
		"call to frozen":           {otherAccount, frozenAccount, nil, true},
		"call by frozen":           {frozenAccount, token, nil, true},
		"token call naming frozen": {otherAccount, token, tokenCall(transferSelector, frozenAccount), true},
		"other call":               {otherAccount, token, tokenCall(transferSelector, otherAccount), false},
		"internal transaction":     {common.Address{}, frozenAccount, nil, false},
		"system call":              {params.SystemAddress, frozenAccount, nil, false},
		"opted-out operator":       {operator, frozenAccount, nil, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newFreezeEVM(t, test.sender)
			statedb.SetState(registry.GetAddress(), registry.GetOptedOutSlot(operator), common.Hash{31: 1})

			ret, _, err := evm.Call(test.sender, test.to, test.input, 100_000, new(uint256.Int))
			if test.refused {
				require.ErrorIs(t, err, vm.ErrExecutionReverted)
				require.Equal(t, frozenRevert, ret)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInterpreter_RefusedCallFailsOnlyForItsCaller(t *testing.T) {
	evm, statedb := newFreezeEVM(t, otherAccount)
	payer := common.Address{0x3}
	statedb.AddBalance(payer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	// CALL(GAS, frozenAccount, 1, 0, 0, 0, 0) POP SSTORE(0, 1) STOP
	code := append(common.FromHex("0x6000600060006000600173"), frozenAccount.Bytes()...)
	statedb.SetCode(payer, append(code, 0x5a, 0xf1, 0x50, 0x60, 0x01, 0x60, 0x00, 0x55, 0x00), tracing.CodeChangeUnspecified)

	_, _, err := evm.Call(otherAccount, payer, nil, 100_000, new(uint256.Int))
	require.NoError(t, err)
	require.Equal(t, common.Hash{31: 1}, statedb.GetState(payer, common.Hash{}))
	require.Equal(t, uint64(1), statedb.GetBalance(payer).Uint64())
}
