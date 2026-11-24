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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var (
	frozenAccount   = common.Address{0x1}
	unfrozenAccount = common.Address{0x2}
)

// newState returns a state wrapped for frozen-move tracking in which
// frozenAccount is frozen and both accounts hold 10 wei.
func newState(t *testing.T) (*StateDB, *state.StateDB) {
	inner, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	inner.SetState(registry.GetAddress(), registry.GetFrozenSlot(frozenAccount), common.Hash{31: 1})
	for _, account := range []common.Address{frozenAccount, unfrozenAccount} {
		inner.SetBalance(account, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	}
	return Wrap(inner, NewRegistryChecker(true, inner)), inner
}

func TestStateDB_Prepare_ResetsTracking(t *testing.T) {
	db, inner := newState(t)
	db.frozenMoved = true
	db.frozenMovedAt[1] = true

	db.Prepare(params.Rules{IsEIP2929: true}, unfrozenAccount, common.Address{}, nil, nil, nil)
	require.False(t, db.frozenMoved)
	require.Empty(t, db.frozenMovedAt)
	require.True(t, inner.AddressInAccessList(unfrozenAccount))
}

func TestStateDB_Snapshot_RecordsFrozenMoved(t *testing.T) {
	tests := map[string]struct {
		moved bool
	}{
		"not moved": {false},
		"moved":     {true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, _ := newState(t)
			db.frozenMoved = test.moved
			id := db.Snapshot()
			require.Equal(t, map[int]bool{id: test.moved}, db.frozenMovedAt)
		})
	}
}

func TestStateDB_RevertToSnapshot_RestoresFrozenMovedAndState(t *testing.T) {
	tests := map[string]struct {
		recorded bool
	}{
		"not moved": {false},
		"moved":     {true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			id := db.Snapshot()
			db.frozenMovedAt[id] = test.recorded
			db.frozenMoved = !test.recorded
			inner.AddBalance(frozenAccount, uint256.NewInt(1), tracing.BalanceChangeUnspecified)

			db.RevertToSnapshot(id)
			require.Equal(t, test.recorded, db.frozenMoved)
			require.Equal(t, uint256.NewInt(10), inner.GetBalance(frozenAccount))
		})
	}
}

func TestStateDB_BalanceChanges_TrackFrozenMoves(t *testing.T) {
	tests := map[string]struct {
		sub         bool
		reason      tracing.BalanceChangeReason
		expectMoved bool
	}{
		"add transfer":   {false, tracing.BalanceChangeTransfer, true},
		"add gas refund": {false, tracing.BalanceIncreaseGasReturn, false},
		"sub transfer":   {true, tracing.BalanceChangeTransfer, true},
		"sub gas buy":    {true, tracing.BalanceDecreaseGasBuy, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			one := uint256.NewInt(1)
			balance := uint256.NewInt(10)
			if test.sub {
				db.SubBalance(frozenAccount, one, test.reason)
				balance.Sub(balance, one)
			} else {
				db.AddBalance(frozenAccount, one, test.reason)
				balance.Add(balance, one)
			}
			require.Equal(t, test.expectMoved, db.frozenMoved)
			require.Equal(t, balance, inner.GetBalance(frozenAccount))
		})
	}
}

func TestStateDB_EVM_TracksFrozenMoves(t *testing.T) {
	var (
		revert       = program.New().Op(vm.PUSH0, vm.PUSH0, vm.REVERT).Bytes()
		callFrozen   = program.New().Call(nil, frozenAccount, 1, 0, 0, 0, 0).Bytes()
		callUnfrozen = program.New().Call(nil, unfrozenAccount, 1, 0, 0, 0, 0).Bytes()
		create       = program.New().Push(0).Push(0).Push(1).Op(vm.CREATE).Bytes()
		create2      = program.New().Push(0).Push(0).Push(0).Push(1).Op(vm.CREATE2).Bytes()
		payFrozen    = program.New().Selfdestruct(frozenAccount).Bytes()
		payUnfrozen  = program.New().Selfdestruct(unfrozenAccount).Bytes()
		burn         = program.New().Op(vm.ADDRESS, vm.SELFDESTRUCT).Bytes()
		createBurner = program.New().Create2(burn, 0).Bytes()
		newFrozen    = crypto.CreateAddress2(unfrozenAccount, common.Hash{}, crypto.Keccak256(burn))
	)
	tests := map[string]struct {
		to                       common.Address
		value                    uint64
		frozenCode, unfrozenCode []byte
		err                      error
		expectMoved              bool
	}{
		"value call to frozen":                 {frozenAccount, 1, nil, nil, nil, true},
		"zero-value call to frozen":            {frozenAccount, 0, nil, nil, nil, false},
		"value call to unfrozen":               {unfrozenAccount, 1, nil, nil, nil, false},
		"value call to reverting frozen":       {frozenAccount, 1, revert, nil, vm.ErrExecutionReverted, false},
		"inner value call to frozen":           {unfrozenAccount, 0, nil, callFrozen, nil, true},
		"inner value call to reverting frozen": {unfrozenAccount, 0, revert, callFrozen, nil, false},
		"frozen calls with value":              {frozenAccount, 0, callUnfrozen, nil, nil, true},
		"frozen calls itself with value":       {frozenAccount, 0, callFrozen, nil, nil, true},
		"frozen creates with value":            {frozenAccount, 0, create, nil, nil, true},
		"frozen creates2 with value":           {frozenAccount, 0, create2, nil, nil, true},
		"frozen selfdestructs":                 {frozenAccount, 0, payUnfrozen, nil, nil, true},
		"frozen selfdestructs to itself":       {frozenAccount, 0, payFrozen, nil, nil, false},
		"new frozen contract burns balance":    {unfrozenAccount, 0, nil, createBurner, nil, true},
		"selfdestruct pays frozen":             {unfrozenAccount, 0, nil, payFrozen, nil, true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			caller := common.Address{0x3}
			inner.SetBalance(caller, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
			inner.SetState(registry.GetAddress(), registry.GetFrozenSlot(newFrozen), common.Hash{31: 1})
			inner.SetBalance(newFrozen, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
			inner.SetCode(frozenAccount, test.frozenCode, tracing.CodeChangeUnspecified)
			inner.SetCode(unfrozenAccount, test.unfrozenCode, tracing.CodeChangeUnspecified)
			evm := vm.NewEVM(vm.BlockContext{
				CanTransfer: core.CanTransfer, Transfer: core.Transfer,
				BlockNumber: big.NewInt(0), BaseFee: big.NewInt(0), Random: &common.Hash{},
			}, db, params.MergedTestChainConfig, vm.Config{})

			_, _, err := evm.Call(caller, test.to, nil, vm.NewGasBudget(100_000, 0), uint256.NewInt(test.value))
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.expectMoved, db.frozenMoved)
		})
	}
}

func TestIsFrozenMove(t *testing.T) {
	tests := map[string]struct {
		account     common.Address
		amount      uint64
		reason      tracing.BalanceChangeReason
		expectMoved bool
	}{
		"transfer of frozen":            {frozenAccount, 1, tracing.BalanceChangeTransfer, true},
		"selfdestruct payout to frozen": {frozenAccount, 1, tracing.BalanceIncreaseSelfdestruct, true},
		"selfdestruct of frozen":        {frozenAccount, 1, tracing.BalanceDecreaseSelfdestruct, true},
		"selfdestruct burn of frozen":   {frozenAccount, 1, tracing.BalanceDecreaseSelfdestructBurn, true},
		"zero transfer of frozen":       {frozenAccount, 0, tracing.BalanceChangeTransfer, false},
		"account not frozen":            {unfrozenAccount, 1, tracing.BalanceChangeTransfer, false},
		"reason not covered":            {frozenAccount, 1, tracing.BalanceDecreaseGasBuy, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, _ := newState(t)
			require.Equal(t, test.expectMoved, isFrozenMove(db.checker, test.account, uint256.NewInt(test.amount), test.reason))
		})
	}
}
