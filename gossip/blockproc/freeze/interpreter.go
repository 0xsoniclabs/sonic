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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// frozenRevert is the revert data of calls refused by the freeze, the
// encoding of Error("account is frozen").
var frozenRevert = func() []byte {
	stringType, _ := abi.NewType("string", "", nil)
	reason, err := abi.Arguments{{Type: stringType}}.Pack("account is frozen")
	if err != nil {
		panic(err)
	}
	return append(common.FromHex("0x08c379a0"), reason...)
}()

// WrapInterpreter returns factory with the checks of the freeze: a call made
// by or to a frozen account, or an ERC-20 transfer naming one, reverts with
// "account is frozen" before any code runs, and its caller can handle the
// failure. Calls of accounts without code run no interpreter and pass.
func WrapInterpreter(factory vm.InterpreterFactory) vm.InterpreterFactory {
	return func(evm *vm.EVM) vm.Interpreter {
		return interpreter{factory(evm), evm}
	}
}

type interpreter struct {
	vm.Interpreter
	evm *vm.EVM
}

// Interpret runs contract unless the freeze refuses the call. Transactions of
// internal senders (zero address), go-ethereum's system calls and opted-out
// operators are exempt.
func (i interpreter) Interpret(contract *vm.Contract, input []byte, readOnly bool) ([]byte, error) {
	db, origin := i.evm.StateDB, i.evm.Origin
	refused := IsFrozen(db, contract.Address()) ||
		IsFrozen(db, contract.Caller()) ||
		isFrozenTokenCall(db, contract.Address(), input)
	if refused && origin != (common.Address{}) && origin != params.SystemAddress && !IsOptedOut(db, origin) {
		contract.Gas &^= 1 << 63 // Tosca's adapter marks calls from read-only frames in the top bit
		return frozenRevert, vm.ErrExecutionReverted
	}
	return i.Interpreter.Interpret(contract, input, readOnly)
}
