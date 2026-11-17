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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const (
	gasLimit = 100_000
	gasPrice = 2
)

var sender = common.Address{0x1}

func newFreezeEVM(t *testing.T, hooks *tracing.Hooks, frozen ...common.Address) (*vm.EVM, *state.StateDB) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	for _, account := range append(frozen, frozenAccount) {
		statedb.SetState(registry.GetAddress(), registry.GetFrozenSlot(account), common.Hash{31: 1})
	}
	statedb.AddBalance(sender, uint256.NewInt(gasLimit*gasPrice+10), tracing.BalanceChangeUnspecified)
	evm := NewEVM(vm.BlockContext{
		CanTransfer: func(db vm.StateDB, from common.Address, amount *uint256.Int) bool {
			return db.GetBalance(from).Cmp(amount) >= 0
		},
		Transfer: func(db vm.StateDB, from, to common.Address, amount *uint256.Int, _ *params.Rules) {
			db.SubBalance(from, amount, tracing.BalanceChangeTransfer)
			db.AddBalance(to, amount, tracing.BalanceChangeTransfer)
		},
		BlockNumber: big.NewInt(1),
		BaseFee:     big.NewInt(0),
		GasLimit:    gasLimit,
		Random:      &common.Hash{},
	}, statedb, params.MergedTestChainConfig, vm.Config{Tracer: hooks})
	return evm, statedb
}

// apply runs a transaction of from to the address to, or a creation if to
// is nil, through ApplyMessage.
func apply(t *testing.T, evm *vm.EVM, from common.Address, to *common.Address, value uint64) *core.ExecutionResult {
	t.Helper()
	msg := &core.Message{
		From:      from,
		To:        to,
		Nonce:     evm.StateDB.GetNonce(from),
		Value:     new(big.Int).SetUint64(value),
		GasLimit:  gasLimit,
		GasPrice:  big.NewInt(gasPrice),
		GasFeeCap: big.NewInt(gasPrice),
		GasTipCap: big.NewInt(gasPrice),
	}
	evm.SetTxContext(vm.TxContext{Origin: from, GasPrice: uint256.NewInt(gasPrice)})
	result, err := ApplyMessage(evm, msg, core.NewGasPool(gasLimit))
	require.NoError(t, err)
	return result
}

// payCode is a contract that calls to with 1 wei, ignores whether the call
// succeeded, and then stores 1 in slot 0. With revert set, it reverts instead
// of storing.
func payCode(to common.Address, revert bool) []byte {
	code := append(common.FromHex("0x6000600060006000600173"), to.Bytes()...)
	code = append(code, 0x5a, 0xf1, 0x50) // GAS CALL POP
	if revert {
		return append(code, 0x60, 0x00, 0x60, 0x00, 0xfd)
	}
	return append(code, 0x60, 0x01, 0x60, 0x00, 0x55, 0x00)
}

func TestApplyMessage_FailsTheTransactionEvenIfTheContractIgnoresTheFailure(t *testing.T) {
	type exit struct {
		depth int
		err   error
	}
	var exits []exit
	evm, statedb := newFreezeEVM(t, &tracing.Hooks{
		OnExit: func(depth int, _ []byte, _ uint64, err error, _ bool) { exits = append(exits, exit{depth, err}) },
	})
	payer := common.Address{0x2}
	statedb.AddBalance(payer, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	statedb.SetCode(payer, payCode(frozenAccount, false), tracing.CodeChangeUnspecified)
	balance := statedb.GetBalance(sender).Uint64()

	result := apply(t, evm, sender, &payer, 0)
	require.ErrorIs(t, result.Err, vm.ErrExecutionReverted)
	require.Equal(t, frozenRevert, result.ReturnData)
	require.Equal(t, common.Hash{}, statedb.GetState(payer, common.Hash{}), "payer's work undone")
	require.Equal(t, uint64(10), statedb.GetBalance(payer).Uint64())
	require.Zero(t, statedb.GetBalance(frozenAccount).Uint64())
	require.Equal(t, []exit{{1, nil}, {0, vm.ErrExecutionReverted}}, exits, "only the outermost call fails")

	require.Equal(t, uint64(1), statedb.GetNonce(sender), "nonce used")
	require.Equal(t, balance-result.UsedGas*gasPrice, statedb.GetBalance(sender).Uint64(), "gas charged")
	require.Empty(t, statedb.Logs())
}

func TestApplyMessage_IgnoresMovesUndoneByARevert(t *testing.T) {
	evm, statedb := newFreezeEVM(t, nil)
	outer, inner := common.Address{0x2}, common.Address{0x3}
	statedb.AddBalance(inner, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	statedb.SetCode(inner, payCode(frozenAccount, true), tracing.CodeChangeUnspecified)
	// CALL(GAS, inner, 0, 0, 0, 0, 0) POP SSTORE(0, 1) STOP
	code := append(common.FromHex("0x6000600060006000600073"), inner.Bytes()...)
	statedb.SetCode(outer, append(code, 0x5a, 0xf1, 0x50, 0x60, 0x01, 0x60, 0x00, 0x55, 0x00), tracing.CodeChangeUnspecified)

	result := apply(t, evm, sender, &outer, 0)
	require.NoError(t, result.Err)
	require.Equal(t, common.BigToHash(big.NewInt(1)), statedb.GetState(outer, common.Hash{}))
	require.Zero(t, statedb.GetBalance(frozenAccount).Uint64())
}

func TestApplyMessage_ExemptsInternalSystemAndOptedOutSenders(t *testing.T) {
	payer := common.Address{0x2}
	for _, from := range []common.Address{{}, params.SystemAddress, sender} {
		evm, statedb := newFreezeEVM(t, nil)
		statedb.SetState(registry.GetAddress(), registry.GetOptedOutSlot(sender), common.Hash{31: 1})
		statedb.AddBalance(from, uint256.NewInt(gasLimit*gasPrice), tracing.BalanceChangeUnspecified)
		statedb.AddBalance(payer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		statedb.SetCode(payer, payCode(frozenAccount, false), tracing.CodeChangeUnspecified)

		result := apply(t, evm, from, &payer, 0)
		require.NoError(t, result.Err, "sender %v", from)
		require.Equal(t, uint64(1), statedb.GetBalance(frozenAccount).Uint64())
	}
}

func TestApplyMessage_FailedCreationUsesTheNonce(t *testing.T) {
	created := crypto.CreateAddress(sender, 0)
	evm, statedb := newFreezeEVM(t, nil, created)
	balance := statedb.GetBalance(sender).Uint64()

	result := apply(t, evm, sender, nil, 1)
	require.ErrorIs(t, result.Err, vm.ErrExecutionReverted)
	require.Equal(t, uint64(1), statedb.GetNonce(sender))
	require.Zero(t, statedb.GetBalance(created).Uint64())
	require.Equal(t, balance-result.UsedGas*gasPrice, statedb.GetBalance(sender).Uint64())
}

func TestApplyMessage_FailsSelfdestructFromOrToFrozenAccount(t *testing.T) {
	other := common.Address{0x2}
	selfdestructTo := func(beneficiary common.Address) []byte {
		return append(append([]byte{0x73}, beneficiary.Bytes()...), 0xff) // PUSH20 SELFDESTRUCT
	}
	tests := map[string]struct {
		contract, beneficiary common.Address
	}{
		"to frozen":   {other, frozenAccount},
		"from frozen": {frozenAccount, other},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newFreezeEVM(t, nil)
			statedb.AddBalance(test.contract, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			statedb.SetCode(test.contract, selfdestructTo(test.beneficiary), tracing.CodeChangeUnspecified)

			result := apply(t, evm, sender, &test.contract, 0)
			require.ErrorIs(t, result.Err, vm.ErrExecutionReverted)
			require.Equal(t, uint64(1), statedb.GetBalance(test.contract).Uint64())
		})
	}
}

func TestFrozenRevert_DecodesToReason(t *testing.T) {
	reason, err := abi.UnpackRevert(frozenRevert)
	require.NoError(t, err)
	require.Equal(t, "account is frozen", reason)
}
