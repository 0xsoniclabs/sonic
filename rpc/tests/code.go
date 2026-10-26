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
package tests

import "github.com/ethereum/go-ethereum/core/vm"

// SpinningCode loops until it runs out of gas.
var SpinningCode = []byte{byte(vm.JUMPDEST), byte(vm.PUSH1), 0, byte(vm.JUMP)}

// StorageWritingCode writes 1 into slot 0 and returns.
var StorageWritingCode = []byte{byte(vm.PUSH1), 1, byte(vm.PUSH1), 0, byte(vm.SSTORE), byte(vm.STOP)}

// RevertingCode reverts with the error string "boom".
var RevertingCode = []byte{
	// Error(string) selector, offset 0x20, length 4, "boom"
	byte(vm.PUSH4), 0x08, 0xc3, 0x79, 0xa0, byte(vm.PUSH1), 0xe0, byte(vm.SHL), byte(vm.PUSH1), 0, byte(vm.MSTORE),
	byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 4, byte(vm.MSTORE),
	byte(vm.PUSH1), 4, byte(vm.PUSH1), 0x24, byte(vm.MSTORE),
	byte(vm.PUSH4), 'b', 'o', 'o', 'm', byte(vm.PUSH1), 0xe0, byte(vm.SHL), byte(vm.PUSH1), 0x44, byte(vm.MSTORE),
	byte(vm.PUSH1), 0x64, byte(vm.PUSH1), 0, byte(vm.REVERT),
}

// ReturningCode returns the 32-byte word 42.
var ReturningCode = []byte{byte(vm.PUSH1), 42, byte(vm.PUSH1), 0, byte(vm.MSTORE), byte(vm.PUSH1), 32, byte(vm.PUSH1), 0, byte(vm.RETURN)}
