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

var sender = common.Address{0x9}

// newEVMWithFrozenAccounts returns a Prague EVM configured like Sonic's, on a
// fresh state where `frozen` and `frozenAccount` are frozen and sender can pay
// for gasLimit gas.
func newEVMWithFrozenAccounts(t *testing.T, frozen ...common.Address) (*vm.EVM, *state.StateDB) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	for _, account := range append(frozen, frozenAccount) {
		statedb.SetState(registry.GetAddress(), registry.GetFrozenSlot(account), common.Hash{31: 1})
	}
	statedb.AddBalance(sender, uint256.NewInt(gasLimit*gasPrice+10), tracing.BalanceChangeUnspecified)
	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		BlockNumber: big.NewInt(1),
		BaseFee:     big.NewInt(0),
		Random:      &common.Hash{}, // enables Prague
	}, statedb, params.MergedTestChainConfig, vm.Config{SkipTipPaymentToCoinbase: true, ChargeExcessGas: true})
	return evm, statedb
}

// apply runs a transaction of `from` to the address `to`, or a creation if `to`
// is nil, through ApplyMessage.
func apply(t *testing.T, evm *vm.EVM, from common.Address, to *common.Address, value uint64, accountFreeze bool) *core.ExecutionResult {
	t.Helper()
	msg := &core.Message{
		From:      from,
		To:        to,
		Nonce:     evm.StateDB.GetNonce(from),
		Value:     uint256.NewInt(value),
		GasLimit:  gasLimit,
		GasPrice:  uint256.NewInt(gasPrice),
		GasFeeCap: uint256.NewInt(gasPrice),
		GasTipCap: uint256.NewInt(gasPrice),
	}
	evm.SetTxContext(vm.TxContext{Origin: from, GasPrice: uint256.NewInt(gasPrice)})
	result, err := ApplyMessage(evm, msg, core.NewGasPool(gasLimit), NewRegistryChecker(accountFreeze, evm.StateDB))
	require.NoError(t, err)
	return result
}

// payCode is a contract that calls `to` with 1 wei, ignores whether the call
// succeeded, and then stores 1 in slot 0. With `revert` set, it reverts instead
// of storing.
func payCode(to common.Address, revert bool) []byte {
	code := append(common.FromHex("0x6000600060006000600173"), to.Bytes()...)
	code = append(code, 0x5a, 0xf1, 0x50) // GAS CALL POP
	if revert {
		return append(code, 0x60, 0x00, 0x60, 0x00, 0xfd)
	}
	return append(code, 0x60, 0x01, 0x60, 0x00, 0x55, 0x00)
}

func TestApplyMessage_RejectsFrozenSenders(t *testing.T) {
	var (
		frozenSender   = common.Address{0x1}
		unfrozenSender = common.Address{0x2}
		zeroAddress    = common.Address{}
	)
	tests := map[string]struct {
		from      common.Address
		enabled   bool
		expectErr error
	}{
		"freeze enabled frozen sender returns error": {frozenSender, true, ErrFrozenAccount},
		"freeze enabled unfrozen sender runs":        {unfrozenSender, true, nil},
		"freeze enabled frozen zero address runs":    {zeroAddress, true, nil},
		"freeze disabled frozen sender runs":         {frozenSender, false, nil},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newEVMWithFrozenAccounts(t, frozenSender, zeroAddress)
			zero := new(uint256.Int)
			_, err := ApplyMessage(evm, &core.Message{
				From: test.from, To: &common.Address{0x3}, Value: zero, GasLimit: 21_000,
				GasPrice: zero, GasFeeCap: zero, GasTipCap: zero,
			}, core.NewGasPool(21_000), NewRegistryChecker(test.enabled, statedb))
			require.ErrorIs(t, err, test.expectErr)
			nonce := uint64(0)
			if test.expectErr == nil {
				nonce = 1
			}
			require.Equal(t, nonce, statedb.GetNonce(test.from))
		})
	}
}

func TestApplyMessage_RestoresTheStateDBOfTheEVM(t *testing.T) {
	payer := common.Address{0x2}
	tests := map[string]struct {
		accountFreeze          bool
		to                     common.Address
		nonce                  uint64
		expectErr, expectVMErr error
	}{
		"disabled":           {accountFreeze: false, to: payer},
		"enabled non-frozen": {accountFreeze: true, to: common.Address{0x3}},
		"enabled frozen":     {accountFreeze: true, to: payer, expectVMErr: vm.ErrExecutionReverted},
		"invalid message":    {accountFreeze: true, to: payer, nonce: 1, expectErr: core.ErrNonceTooHigh},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newEVMWithFrozenAccounts(t)
			statedb.AddBalance(payer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			statedb.SetCode(payer, payCode(frozenAccount, false), tracing.CodeChangeUnspecified)

			zero := new(uint256.Int)
			result, err := ApplyMessage(evm, &core.Message{
				From: sender, To: &test.to, Nonce: test.nonce, Value: zero, GasLimit: gasLimit,
				GasPrice: zero, GasFeeCap: zero, GasTipCap: zero,
			}, core.NewGasPool(gasLimit), NewRegistryChecker(test.accountFreeze, statedb))
			require.ErrorIs(t, err, test.expectErr)
			if err == nil {
				require.ErrorIs(t, result.Err, test.expectVMErr)
			}
			require.Same(t, statedb, evm.StateDB)
		})
	}
}

func TestApplyMessage_SettlesFrozenMovesAsRevertedTransactions(t *testing.T) {
	other, created := common.Address{0x2}, crypto.CreateAddress(sender, 0)
	selfdestructTo := func(beneficiary common.Address) []byte {
		return append(append([]byte{0x73}, beneficiary.Bytes()...), 0xff) // PUSH20 SELFDESTRUCT
	}
	// LOG3(0, 0, Transfer, from, to) STOP
	transferLog := func(from, to common.Address) []byte {
		return slices.Concat([]byte{0x73}, to.Bytes(), []byte{0x73}, from.Bytes(), []byte{0x7f}, transferTopic.Bytes(), []byte{0x60, 0x00, 0x60, 0x00, 0xa3, 0x00})
	}
	// SSTORE(1, 1) SSTORE(1, 0), then the payment
	refundThenPay := append([]byte{0x60, 0x01, 0x60, 0x01, 0x55, 0x60, 0x00, 0x60, 0x01, 0x55}, payCode(frozenAccount, false)...)
	tests := map[string]struct {
		to           *common.Address
		value        uint64
		code         []byte
		expectReason tracing.NonceChangeReason
	}{
		"value to frozen":               {to: &frozenAccount, value: 1, expectReason: tracing.NonceChangeEoACall},
		"value from frozen":             {to: &frozenAccount, code: payCode(other, false), expectReason: tracing.NonceChangeEoACall},
		"creation at frozen":            {to: nil, value: 1, expectReason: tracing.NonceChangeContractCreator},
		"selfdestruct to frozen":        {to: &other, code: selfdestructTo(frozenAccount), expectReason: tracing.NonceChangeEoACall},
		"selfdestruct from frozen":      {to: &frozenAccount, code: selfdestructTo(other), expectReason: tracing.NonceChangeEoACall},
		"contract ignoring the failure": {to: &other, code: payCode(frozenAccount, false), expectReason: tracing.NonceChangeEoACall},
		"token transfer to frozen":      {to: &other, code: transferLog(other, frozenAccount), expectReason: tracing.NonceChangeEoACall},
		"token transfer from frozen":    {to: &other, code: transferLog(frozenAccount, other), expectReason: tracing.NonceChangeEoACall},
		"refund before the move":        {to: &other, code: refundThenPay, expectReason: tracing.NonceChangeEoACall},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			setup := func() (*vm.EVM, *state.StateDB) {
				evm, statedb := newEVMWithFrozenAccounts(t, created)
				if test.to != nil {
					statedb.AddBalance(*test.to, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
					statedb.SetCode(*test.to, test.code, tracing.CodeChangeUnspecified)
				}
				return evm, statedb
			}
			executedEVM, _ := setup()
			executed := apply(t, executedEVM, sender, test.to, test.value, false)
			require.NoError(t, executed.Err)

			evm, statedb := setup()
			var reason tracing.NonceChangeReason
			evm.StateDB = state.NewHookedState(statedb, &tracing.Hooks{
				OnNonceChangeV2: func(_ common.Address, _, _ uint64, r tracing.NonceChangeReason) { reason = r },
			})
			balances := func() []uint64 {
				return []uint64{statedb.GetBalance(frozenAccount).Uint64(), statedb.GetBalance(other).Uint64(), statedb.GetBalance(created).Uint64()}
			}
			before, balance := balances(), statedb.GetBalance(sender).Uint64()

			result := apply(t, evm, sender, test.to, test.value, true)
			require.ErrorIs(t, result.Err, vm.ErrExecutionReverted)
			revertReason, err := abi.UnpackRevert(result.ReturnData)
			require.NoError(t, err)
			require.Equal(t, "account is frozen", revertReason)
			require.Equal(t, before, balances())
			require.Equal(t, common.Hash{}, statedb.GetState(other, common.Hash{}))
			require.Empty(t, statedb.Logs())

			require.Equal(t, uint64(1), statedb.GetNonce(sender))
			require.Equal(t, test.expectReason, reason)
			require.Equal(t, balance-result.UsedGas*gasPrice, statedb.GetBalance(sender).Uint64())
			require.Equal(t, []uint64{executed.UsedGas, executed.MaxUsedGas}, []uint64{result.UsedGas, result.MaxUsedGas})
		})
	}
}

func TestApplyMessage_IgnoresMovesThatDoNotViolateTheFreeze(t *testing.T) {
	payer, outer, inner := common.Address{0x2}, common.Address{0x3}, common.Address{0x4}
	tests := map[string]struct {
		disabled                  bool
		from                      common.Address
		to                        common.Address
		expectFailed, expectMoved bool
	}{
		"freeze disabled":         {disabled: true, from: sender, to: payer, expectFailed: false, expectMoved: true},
		"internal sender":         {from: common.Address{}, to: payer, expectFailed: false, expectMoved: true},
		"system sender":           {from: params.SystemAddress, to: payer, expectFailed: false, expectMoved: true},
		"regular sender":          {from: sender, to: payer, expectFailed: true, expectMoved: false},
		"move undone by a revert": {from: sender, to: outer, expectFailed: false, expectMoved: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newEVMWithFrozenAccounts(t)
			statedb.AddBalance(test.from, uint256.NewInt(gasLimit*gasPrice), tracing.BalanceChangeUnspecified)
			statedb.AddBalance(payer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			statedb.SetCode(payer, payCode(frozenAccount, false), tracing.CodeChangeUnspecified)
			statedb.AddBalance(outer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			statedb.SetCode(outer, payCode(inner, false), tracing.CodeChangeUnspecified)
			statedb.SetCode(inner, payCode(frozenAccount, true), tracing.CodeChangeUnspecified)

			result := apply(t, evm, test.from, &test.to, 0, !test.disabled)
			require.Equal(t, test.expectFailed, result.Err != nil)
			require.Equal(t, test.expectMoved, !statedb.GetBalance(frozenAccount).IsZero())
		})
	}
}

func TestApplyMessage_FrozenMoveUndoesItsAuthorizations(t *testing.T) {
	payer, delegate := common.Address{0x2}, common.Address{0x7}
	tests := map[string]struct {
		accountFreeze bool
		expectVMErr   error
		expectNonce   uint64
		expectCode    []byte
	}{
		"freeze disabled": {accountFreeze: false, expectNonce: 1, expectCode: types.AddressToDelegation(delegate)},
		"freeze enabled":  {accountFreeze: true, expectVMErr: vm.ErrExecutionReverted},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evm, statedb := newEVMWithFrozenAccounts(t)
			statedb.AddBalance(payer, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			statedb.SetCode(payer, payCode(frozenAccount, false), tracing.CodeChangeUnspecified)
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			auth, err := types.SignSetCode(key, types.SetCodeAuthorization{
				ChainID: *uint256.MustFromBig(params.MergedTestChainConfig.ChainID),
				Address: delegate,
			})
			require.NoError(t, err)

			zero := new(uint256.Int)
			result, err := ApplyMessage(evm, &core.Message{
				From: sender, To: &payer, Value: zero, GasLimit: 1_000_000,
				GasPrice: zero, GasFeeCap: zero, GasTipCap: zero,
				SetCodeAuthorizations: []types.SetCodeAuthorization{auth},
			}, core.NewGasPool(1_000_000), NewRegistryChecker(test.accountFreeze, statedb))
			require.NoError(t, err)
			require.ErrorIs(t, result.Err, test.expectVMErr)

			authority := crypto.PubkeyToAddress(key.PublicKey)
			require.Equal(t, test.expectNonce, statedb.GetNonce(authority))
			require.Equal(t, test.expectCode, statedb.GetCode(authority))
		})
	}
}
