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

package evmwriter

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
)

func TestPreCompiledContract_Revert_BurnsExecutionGasAndKeepsReservoir(t *testing.T) {
	origin := common.Address{1}
	word := func(b []byte) []byte { return common.LeftPadBytes(b, 32) }
	call := func(id []byte, args ...[]byte) []byte { return append(bytes.Clone(id), bytes.Join(args, nil)...) }
	other := word(common.Address{2}.Bytes())

	tests := map[string]struct {
		caller common.Address
		input  []byte
	}{
		"caller is not the driver":   {common.Address{3}, call(setBalanceMethodID, other, word([]byte{1}))},
		"input shorter than 4 bytes": {driver.ContractAddress, []byte{1, 2, 3}},
		"unknown method":             {driver.ContractAddress, []byte{1, 2, 3, 4}},
		"setBalance malformed":       {driver.ContractAddress, call(setBalanceMethodID, other)},
		"setBalance of origin":       {driver.ContractAddress, call(setBalanceMethodID, word(origin.Bytes()), word([]byte{1}))},
		"copyCode malformed":         {driver.ContractAddress, call(copyCodeMethodID, other)},
		"swapCode malformed":         {driver.ContractAddress, call(swapCodeMethodID, other)},
		"setStorage malformed":       {driver.ContractAddress, call(setStorageMethodID, other, other)},
		"incNonce malformed":         {driver.ContractAddress, call(incNonceMethodID, other)},
		"incNonce of origin":         {driver.ContractAddress, call(incNonceMethodID, word(origin.Bytes()), word([]byte{1}))},
		"incNonce by 256":            {driver.ContractAddress, call(incNonceMethodID, other, word([]byte{1, 0}))},
		"incNonce by 0":              {driver.ContractAddress, call(incNonceMethodID, other, word(nil))},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			supplied := vm.GasBudget{ExecutionGas: 1_000_000, StateGas: 500}
			_, left, err := PreCompiledContract{}.Run(nil, vm.BlockContext{}, vm.TxContext{Origin: origin}, test.caller, test.input, supplied)
			require.ErrorIs(t, err, vm.ErrExecutionReverted)

			returned := left.Exit(err)
			require.Zero(t, returned.ExecutionGas)
			require.Equal(t, supplied.StateGas, returned.StateGas)
		})
	}
}

func TestPreCompiledContract_OutOfGas_BurnsExecutionGasAndKeepsReservoir(t *testing.T) {
	word := func(b []byte) []byte { return common.LeftPadBytes(b, 32) }
	call := func(id []byte, args ...[]byte) []byte { return append(bytes.Clone(id), bytes.Join(args, nil)...) }
	acc0, acc1 := word(common.Address{2}.Bytes()), word(common.Address{3}.Bytes())
	codeGas := params.CreateDataGas + params.MemoryGas

	tests := map[string]struct {
		input []byte
		gas   uint64
	}{
		"setBalance":       {call(setBalanceMethodID, acc0, word([]byte{1})), params.CallValueTransferGas - 1},
		"copyCode":         {call(copyCodeMethodID, acc0, acc1), params.CreateGas - 1},
		"copyCode of code": {call(copyCodeMethodID, acc0, acc1), params.CreateGas + codeGas - 1},
		"swapCode":         {call(swapCodeMethodID, acc0, acc1), 2*params.CreateGas - 1},
		"swapCode of code": {call(swapCodeMethodID, acc0, acc1), 2*params.CreateGas + codeGas - 1},
		"setStorage":       {call(setStorageMethodID, acc0, acc1, acc1), params.SstoreSetGasEIP2200 - 1},
		"incNonce":         {call(incNonceMethodID, acc0, word([]byte{1})), params.CallValueTransferGas - 1},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			supplied := vm.GasBudget{ExecutionGas: test.gas, StateGas: 500}
			_, left, err := PreCompiledContract{}.Run(codeState{code: []byte{1}}, vm.BlockContext{}, vm.TxContext{}, driver.ContractAddress, test.input, supplied)
			require.ErrorIs(t, err, vm.ErrOutOfGas)

			returned := left.Exit(err)
			require.Zero(t, returned.ExecutionGas)
			require.Equal(t, supplied.StateGas, returned.StateGas)
		})
	}
}

type codeState struct {
	vm.StateDB
	code []byte
}

func (s codeState) GetCode(common.Address) []byte {
	return s.code
}

func TestSign(t *testing.T) {
	require := require.New(t)

	require.Equal([]byte{0xe3, 0x04, 0x43, 0xbc}, setBalanceMethodID)
	require.Equal([]byte{0xd6, 0xa0, 0xc7, 0xaf}, copyCodeMethodID)
	require.Equal([]byte{0x07, 0x69, 0x0b, 0x2a}, swapCodeMethodID)
	require.Equal([]byte{0x39, 0xe5, 0x03, 0xab}, setStorageMethodID)
	require.Equal([]byte{0x79, 0xbe, 0xad, 0x38}, incNonceMethodID)

}
