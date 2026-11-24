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
	"slices"
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
	token           = common.Address{0x70}
	optedOutToken   = common.Address{0x71}
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

func TestStateDB_AddBalance_TracksFrozenMoves(t *testing.T) {
	tests := map[string]struct {
		account     common.Address
		amount      uint64
		expectMoved bool
	}{
		"frozen":      {frozenAccount, 1, true},
		"zero amount": {frozenAccount, 0, false},
		"unfrozen":    {unfrozenAccount, 1, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			db.AddBalance(test.account, uint256.NewInt(test.amount), tracing.BalanceIncreaseGasReturn)
			require.Equal(t, test.expectMoved, db.frozenMoved)
			require.Equal(t, uint256.NewInt(10+test.amount), inner.GetBalance(test.account))
		})
	}
}

func TestStateDB_SubBalance_TracksFrozenMoves(t *testing.T) {
	tests := map[string]struct {
		account     common.Address
		amount      uint64
		expectMoved bool
	}{
		"frozen":      {frozenAccount, 1, true},
		"zero amount": {frozenAccount, 0, false},
		"unfrozen":    {unfrozenAccount, 1, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			db.SubBalance(test.account, uint256.NewInt(test.amount), tracing.BalanceDecreaseGasBuy)
			require.Equal(t, test.expectMoved, db.frozenMoved)
			require.Equal(t, uint256.NewInt(10-test.amount), inner.GetBalance(test.account))
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
		logFrozen    = program.New().Push(unfrozenAccount).Push(frozenAccount).Push(transferTopic).Push(0).Push(0).Op(vm.LOG3).Bytes()
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
		"token log names frozen":               {unfrozenAccount, 0, nil, logFrozen, nil, true},
		"reverted token log names frozen":      {unfrozenAccount, 0, nil, slices.Concat(logFrozen, revert), vm.ErrExecutionReverted, false},
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

func TestStateDB_AddLog_TracksFrozenTokenMoves(t *testing.T) {
	topic := func(a common.Address) common.Hash { return common.BytesToHash(a.Bytes()) }
	tests := map[string]struct {
		from        common.Address
		expectMoved bool
	}{
		"transfer from frozen":   {frozenAccount, true},
		"transfer from unfrozen": {unfrozenAccount, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, inner := newState(t)
			log := &types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(test.from), topic(unfrozenAccount)}}

			db.AddLog(log)
			require.Equal(t, test.expectMoved, db.frozenMoved)
			require.Equal(t, []*types.Log{log}, inner.Logs())
		})
	}
}

func TestIsFrozenTokenLog_DetectsTokenEventsNamingFrozenAccounts(t *testing.T) {
	db, inner := newState(t)
	inner.SetState(registry.GetAddress(), registry.GetOptedOutSlot(optedOutToken), common.Hash{31: 1})
	topic := func(a common.Address) common.Hash { return common.BytesToHash(a.Bytes()) }

	tests := map[string]struct {
		log         types.Log
		expectMoved bool
	}{
		"transfer from frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(unfrozenAccount)}},
			true,
		},
		"approval by frozen": {
			types.Log{Address: token, Topics: []common.Hash{approvalTopic, topic(frozenAccount), topic(unfrozenAccount)}},
			true,
		},
		"transfer to frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(unfrozenAccount), topic(frozenAccount)}},
			true,
		},
		"approval of frozen spender": {
			types.Log{Address: token, Topics: []common.Hash{approvalTopic, topic(unfrozenAccount), topic(frozenAccount)}},
			true,
		},
		"transfer between others": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(unfrozenAccount), topic(unfrozenAccount)}},
			false,
		},
		"erc-721 transfer from frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(unfrozenAccount), {}}},
			true,
		},
		"erc-721 approval of frozen": {
			types.Log{Address: token, Topics: []common.Hash{approvalTopic, topic(unfrozenAccount), topic(frozenAccount), {}}},
			true,
		},
		"erc-721 token id is no address": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(unfrozenAccount), topic(unfrozenAccount), topic(frozenAccount)}},
			false,
		},
		"approval for all by frozen owner": {
			types.Log{Address: token, Topics: []common.Hash{approvalForAllTopic, topic(frozenAccount), topic(unfrozenAccount)}},
			true,
		},
		"approval for all of frozen operator": {
			types.Log{Address: token, Topics: []common.Hash{approvalForAllTopic, topic(unfrozenAccount), topic(frozenAccount)}},
			true,
		},
		"erc-1155 transfer by frozen operator": {
			types.Log{Address: token, Topics: []common.Hash{transferSingleTopic, topic(frozenAccount), topic(unfrozenAccount), topic(unfrozenAccount)}},
			true,
		},
		"erc-1155 transfer from frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferSingleTopic, topic(unfrozenAccount), topic(frozenAccount), topic(unfrozenAccount)}},
			true,
		},
		"erc-1155 batch transfer to frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferBatchTopic, topic(unfrozenAccount), topic(unfrozenAccount), topic(frozenAccount)}},
			true,
		},
		"erc-1155 transfer between others": {
			types.Log{Address: token, Topics: []common.Hash{transferSingleTopic, topic(unfrozenAccount), topic(unfrozenAccount), topic(unfrozenAccount)}},
			false,
		},
		"opted-out token": {
			types.Log{Address: optedOutToken, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(unfrozenAccount)}},
			false,
		},
		"other event": {
			types.Log{Address: token, Topics: []common.Hash{{0x1}, topic(frozenAccount), topic(unfrozenAccount)}},
			false,
		},
		"no topics": {
			types.Log{Address: token},
			false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expectMoved, isFrozenTokenLog(db.checker, &test.log))
		})
	}
}
