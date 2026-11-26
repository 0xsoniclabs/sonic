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
	"errors"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ErrFrozenAccount is returned if a message is sent by an account listed as
// frozen in the freeze registry.
var ErrFrozenAccount = errors.New("account is frozen")

// frozenRevertData is the revert data of transactions failed by the freeze, the
// encoding of Error("account is frozen").
var frozenRevertData = func() []byte {
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		panic(err)
	}
	reason, err := abi.Arguments{{Type: stringType}}.Pack(ErrFrozenAccount.Error())
	if err != nil {
		panic(err)
	}
	return append(crypto.Keccak256([]byte("Error(string)"))[:4], reason...)
}()

// ApplyMessage runs msg like core.ApplyMessage. If checker is enabled, a frozen
// sender fails with ErrFrozenAccount, and a message moving frozen funds is fully
// rolled back, EIP-7702 authorizations included, and reverts, keeping only its
// nonce use and gas fee. That fee keeps the gas refund and pays no tip, so evm
// must set SkipTipPaymentToCoinbase.
func ApplyMessage(evm *vm.EVM, msg *core.Message, gp *core.GasPool, checker Checker) (*core.ExecutionResult, error) {
	// before the upgrade, run msg exactly as go-ethereum does
	if !checker.IsEnabled() {
		return core.ApplyMessage(evm, msg, gp)
	}

	// the sender is frozen, so fail immediately
	if checker.IsFrozen(msg.From) {
		return nil, ErrFrozenAccount
	}

	// while msg runs, the EVM's state tracks frozen moves
	db := evm.StateDB
	defer func() { evm.StateDB = db }()
	state := Wrap(evm.StateDB, checker)
	evm.StateDB = state

	snapshot := state.Snapshot()
	nonce := state.GetNonce(msg.From)
	result, err := core.ApplyMessage(evm, msg, gp)

	// the tx failed or did not touch frozen funds, so nothing to do
	if err != nil || !violatesFreeze(state, msg.From) {
		return result, err
	}

	// the tx succeeded but must fail because it moves frozen funds
	// revert all its effects, bump the nonce and charge the gas
	state.RevertToSnapshot(snapshot)
	reason := tracing.NonceChangeEoACall
	if msg.To == nil {
		reason = tracing.NonceChangeContractCreator
	}
	state.SetNonce(msg.From, nonce+1, reason)
	// charge the gas used, keeping the refund a real revert would drop
	fee := new(uint256.Int).Mul(uint256.NewInt(result.UsedGas), msg.GasPrice)
	state.SubBalance(msg.From, fee, tracing.BalanceDecreaseGasBuy)
	return &core.ExecutionResult{
		UsedGas:    result.UsedGas,
		MaxUsedGas: result.MaxUsedGas,
		Err:        vm.ErrExecutionReverted,
		ReturnData: slices.Clone(frozenRevertData),
	}, nil
}

// violatesFreeze reports whether sender's transaction moved frozen funds,
// exempting internal transactions and system calls.
func violatesFreeze(state *StateDB, sender common.Address) bool {
	return state.frozenMoved &&
		sender != (common.Address{}) &&
		sender != params.SystemAddress
}
